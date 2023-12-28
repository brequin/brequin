package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/brequin/brequin/scrape/db"
	"golang.org/x/net/html"
)

type RequisiteExpression struct {
	string
}

type TokenType int

const (
	TokenRequisite TokenType = iota
	TokenLParen
	TokenRParen
	TokenAnd
	TokenOr
)

type Token struct {
	Type  TokenType
	Value string
}

type LexerState int

const (
	LexerStart LexerState = iota
	LexerRequisiteNodeId
	LexerRequisiteFlags
)

func (requisiteExpression RequisiteExpression) Tokenize() []Token {
	initialPos := 0
	state := LexerStart

	var tokens []Token
	for pos, char := range requisiteExpression.string {
		switch state {
		case LexerStart:
			switch char {
			case '(':
				tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
				initialPos = pos + 1
			case ')':
				tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
				initialPos = pos + 1
			case '&':
				tokens = append(tokens, Token{Type: TokenAnd, Value: "&"})
				initialPos = pos + 1
			case '|':
				tokens = append(tokens, Token{Type: TokenOr, Value: "|"})
				initialPos = pos + 1
			default:
				state = LexerRequisiteNodeId
			}
		case LexerRequisiteNodeId:
			if char == '{' {
				state = LexerRequisiteFlags
			}
		case LexerRequisiteFlags:
			if char == '}' {
				tokens = append(tokens, Token{Type: TokenRequisite, Value: requisiteExpression.string[initialPos : pos+1]})
				initialPos = pos + 1
				state = LexerStart
			}
		}
	}

	return tokens
}

func (requisiteExpression RequisiteExpression) EvaluateFor(course db.Course) ([]db.Node, []db.Course, []db.Relation, error) {
	tokens := requisiteExpression.Tokenize()
	for _, token := range tokens {
		fmt.Print(`"` + token.Value + `" `)
	}
	fmt.Println()

	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	return nodes, courses, relations, nil
}

func ScrapeRequisiteExpression(classDetailTooltipUrl string) (RequisiteExpression, error) {
	request, err := http.NewRequest("GET", classDetailTooltipUrl, nil)
	if err != nil {
		return RequisiteExpression{""}, err
	}

	// Required
	request.Header.Add("X-Requested-With", "XMLHttpRequest")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return RequisiteExpression{""}, err
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return RequisiteExpression{""}, err
	}

	var reqExpBuilder strings.Builder
	requisiteRows := document.Find("table.requisites_content").Find("tbody").Find("tr.requisite")
	for _, root := range requisiteRows.Nodes {
		requisiteRow := goquery.NewDocumentFromNode(root)

		requisiteDataNodes := requisiteRow.Find("td").Nodes

		expPart, err := goquery.NewDocumentFromNode(requisiteDataNodes[0]).Html()
		if err != nil {
			log.Println("Unable to determine requisite expression")
			return RequisiteExpression{""}, err
		}
		expPart = html.UnescapeString(expPart)

		minimumGrade, err := goquery.NewDocumentFromNode(requisiteDataNodes[1]).Html()
		if err != nil {
			log.Println("Unable to determine requisite minimum grade")
			return RequisiteExpression{""}, err
		}
		minimumGrade = html.UnescapeString(minimumGrade)

		prereqText, err := goquery.NewDocumentFromNode(requisiteDataNodes[2]).Html()
		if err != nil {
			log.Println("Unable to determine whether requisite is a prerequisite")
			return RequisiteExpression{""}, err
		}
		coreqText, err := goquery.NewDocumentFromNode(requisiteDataNodes[3]).Html()
		if err != nil {
			log.Println("Unable to determine whether requisite is a corequisite")
			return RequisiteExpression{""}, err
		}

		enforced := db.Flag(goquery.NewDocumentFromNode(requisiteDataNodes[4]).Find("div.icon-exclamation-sign").Length() == 1)
		prereq := db.Flag(prereqText == "Yes")
		coreq := db.Flag(coreqText == "Yes")

		requisiteFlags := fmt.Sprintf("{%c%c%c%v}", enforced, prereq, coreq, minimumGrade)

		beforeAnd, foundAnd := strings.CutSuffix(expPart, " and")
		if foundAnd {
			expPart = beforeAnd
		}
		beforeOr, foundOr := strings.CutSuffix(expPart, " or")
		if foundOr {
			expPart = beforeOr
		}

		// This works for non-course requisites such as diagnostic tests
		requisiteId := strings.Trim(expPart, "( )")

		splitId := strings.Split(requisiteId, " ")
		catalogNumber := splitId[len(splitId)-1]
		subjectAreaName := strings.Trim(strings.TrimSuffix(requisiteId, catalogNumber), " ")
		subjectAreaCode, okay := subjectAreaNameCodeMap[subjectAreaName]
		if okay {
			requisiteId = db.ValueNodeId(subjectAreaCode, catalogNumber)
			expPart = strings.Replace(expPart, subjectAreaName+" "+catalogNumber, requisiteId, 1)
		}

		expPart = strings.ReplaceAll(expPart, requisiteId, requisiteId+requisiteFlags)
		expPart = strings.ReplaceAll(expPart, " ", "")
		fmt.Fprint(&reqExpBuilder, expPart)
		if foundAnd {
			fmt.Fprint(&reqExpBuilder, "&")
		}
		if foundOr {
			fmt.Fprint(&reqExpBuilder, "|")
		}
	}

	return RequisiteExpression{reqExpBuilder.String()}, nil
}
