package main

import (
	"errors"
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
	TokenEnd
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

func (requisiteExpression RequisiteExpression) Tokenize() *[]Token {
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
	tokens = append(tokens, Token{Type: TokenEnd, Value: "$"})
	return &tokens
}

func Eat(tokens *[]Token, tokenType TokenType) (string, error) {
	if len(*tokens) < 1 {
		return "", errors.New("No token to eat")
	}
	if (*tokens)[0].Type != tokenType {
		return "", errors.New("Invalid token")
	}

	token := (*tokens)[0]
	*tokens = (*tokens)[1:]
	return token.Value, nil
}

func Start(tokens *[]Token) (string, []db.Node, []db.Course, []db.Relation, error) {
	var id string
	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	Expression(tokens)
	Eat(tokens, TokenEnd)

	return id, nodes, courses, relations, nil
}

func Expression(tokens *[]Token) (string, []db.Node, []db.Course, []db.Relation, error) {
	var id string
	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	Term(tokens)
	Terms(tokens)

	return id, nodes, courses, relations, nil
}

func Term(tokens *[]Token) (string, []db.Node, []db.Course, []db.Relation, error) {
	var id string
	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation
	/*
		factorId, factorNodes, factorCourses, factorRelations, err := Factor(tokens)
		if err != nil {
			return "", nil, nil, nil, err
		}

		factorsId, factorsNodes, factorsCourses, factorsRelations, err := Factors(tokens)
		if err != nil {
			return "", nil, nil, nil, err
		}

		id = factorId + factorsId
		termNode := db.Node{Id: id, Type: db.NodeTypeAnd}
		nodes = append(nodes, factorNodes...)
		nodes = append(nodes, factorsNodes...)

		for _, node := range nodes {
		}

		nodes = append(nodes, termNode)
	*/
	return id, nodes, courses, relations, nil
}

func Terms(tokens *[]Token) (string, []db.Node, []db.Course, []db.Relation, error) {
	var id string
	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	if (*tokens)[0].Type == TokenOr {
		Eat(tokens, TokenOr)
		Term(tokens)
		Terms(tokens)
	}

	return id, nodes, courses, relations, nil
}

func Factor(tokens *[]Token) (string, []db.Node, []db.Course, []db.Relation, error) {
	var id string
	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	switch (*tokens)[0].Type {
	case TokenRequisite:
		requisiteIdFlags, err := Eat(tokens, TokenRequisite)
		if err != nil {
			return "", nil, nil, nil, err
		}

		nodeId, flags, found := strings.Cut(requisiteIdFlags, "{")
		if !found {
			return "", nil, nil, nil, errors.New("Unable to determine requisite node id and flags")
		}

		id = nodeId
		nodes = append(nodes, db.Node{Id: nodeId, Type: db.NodeTypeValue})

		if db.Unflag(flags[0]) { // Requisite is a course
			subjectAreaPart, catalogNumber, found := strings.Cut(nodeId, "#")
			if !found {
				return "", nil, nil, nil, errors.New("Unable to determine requisite subject area and course catalog number")
			}
			course := db.Course{SubjectAreaCode: subjectAreaIdCodeMap[subjectAreaPart], CatalogNumber: catalogNumber, NodeId: nodeId}
			courses = append(courses, course)
		}
	case TokenLParen:
		Eat(tokens, TokenLParen)

		expressionId, expressionNodes, expressionCourses, expressionRelations, err := Expression(tokens)
		if err != nil {
			return "", nil, nil, nil, err
		}

		id = "(" + expressionId + ")" // Parentheses necessary for conjunction in term
		nodes = append(nodes, expressionNodes...)
		courses = append(courses, expressionCourses...)
		relations = append(relations, expressionRelations...)

		_, err = Eat(tokens, TokenRParen)
		if err != nil {
			return "", nil, nil, nil, err
		}
	}

	return id, nodes, courses, relations, nil
}

func Factors(tokens *[]Token) (string, []db.Node, []db.Course, []db.Relation, error) {
	var id string
	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	if (*tokens)[0].Type == TokenAnd {
		Eat(tokens, TokenAnd)

		factorId, factorNodes, factorCourses, factorRelations, err := Factor(tokens)
		if err != nil {
			return "", nil, nil, nil, err
		}

		factorsId, factorsNodes, factorsCourses, factorsRelations, err := Factors(tokens)
		if err != nil {
			return "", nil, nil, nil, err
		}

		id = "&" + factorId + factorsId
		nodes = append(nodes, factorNodes...)
		nodes = append(nodes, factorsNodes...)
		courses = append(courses, factorCourses...)
		courses = append(courses, factorsCourses...)
		relations = append(relations, factorRelations...)
		relations = append(relations, factorsRelations...)
	}

	return id, nodes, courses, relations, nil
}

func (requisiteExpression RequisiteExpression) EvaluateForCourse(course db.Course) ([]db.Node, []db.Course, []db.Relation, error) {
	tokens := requisiteExpression.Tokenize()
	Start(tokens)

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

		isCourse := db.Flag(false)
		isEnforced := db.Flag(goquery.NewDocumentFromNode(requisiteDataNodes[4]).Find("div.icon-exclamation-sign").Length() == 1)
		isPrereq := db.Flag(prereqText == "Yes")
		isCoreq := db.Flag(coreqText == "Yes")

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
			isCourse = db.Flag(true)
			requisiteId = db.ValueNodeId(subjectAreaCode, catalogNumber)
			expPart = strings.Replace(expPart, subjectAreaName+" "+catalogNumber, requisiteId, 1)
		}

		requisiteFlags := fmt.Sprintf("{%c%c%c%c%v}", isCourse, isEnforced, isPrereq, isCoreq, minimumGrade)

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
