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

type ParseNode struct {
	*db.Node
	Enforced     *bool
	Prereq       *bool
	Coreq        *bool
	MinimumGrade *string
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

func Start(course db.Course, tokens *[]Token) (nodes []db.Node, courses []db.Course, relations []db.Relation, err error) {
	expressionId, _, nodes, courses, relations, err := Expression(tokens)
	if err != nil {
		return nil, nil, nil, err
	}

	Eat(tokens, TokenEnd)

	relation := db.Relation{SourceId: course.NodeId, TargetId: expressionId}
	relations = append(relations, relation)

	return nodes, courses, relations, nil
}

func Expression(tokens *[]Token) (id string, expression ParseNode, nodes []db.Node, courses []db.Course, relations []db.Relation, err error) {
	termId, headTerm, termNodes, termCourses, termRelations, err := Term(tokens)
	if err != nil {
		return "", ParseNode{}, nil, nil, nil, err
	}

	termsId, tailTerms, termsNodes, termsCourses, termsRelations, err := Terms(tokens)
	if err != nil {
		return "", ParseNode{}, nil, nil, nil, err
	}

	nodes = append(nodes, termNodes...)
	nodes = append(nodes, termsNodes...)
	courses = append(courses, termCourses...)
	courses = append(courses, termsCourses...)
	relations = append(relations, termRelations...)
	relations = append(relations, termsRelations...)

	terms := append(tailTerms, headTerm)
	switch len(terms) {
	case 0:
		return "", ParseNode{}, nil, nil, nil, errors.New("Expression has no terms")
	case 1:
		id = headTerm.Id
		expression = headTerm
	default:
		id = termId + termsId
		expressionNode := db.Node{Id: id, Type: db.NodeTypeOr}
		expression = ParseNode{Node: &expressionNode}
		for _, term := range append(tailTerms, headTerm) {
			relation := db.Relation{SourceId: expression.Id, TargetId: term.Id, Enforced: term.Enforced, Prereq: term.Prereq, Coreq: term.Coreq, MinimumGrade: term.MinimumGrade}
			relations = append(relations, relation)
		}
		nodes = append(nodes, expressionNode)
	}
	return id, expression, nodes, courses, relations, nil
}

func Term(tokens *[]Token) (id string, term ParseNode, nodes []db.Node, courses []db.Course, relations []db.Relation, err error) {
	factorId, headFactor, factorNodes, factorCourses, factorRelations, err := Factor(tokens)
	if err != nil {
		return "", ParseNode{}, nil, nil, nil, err
	}

	factorsId, tailFactors, factorsNodes, factorsCourses, factorsRelations, err := Factors(tokens)
	if err != nil {
		return "", ParseNode{}, nil, nil, nil, err
	}

	nodes = append(nodes, factorNodes...)
	nodes = append(nodes, factorsNodes...)
	courses = append(courses, factorCourses...)
	courses = append(courses, factorsCourses...)
	relations = append(relations, factorRelations...)
	relations = append(relations, factorsRelations...)

	factors := append(tailFactors, headFactor)
	switch len(factors) {
	case 0:
		return "", ParseNode{}, nil, nil, nil, errors.New("Term has no factors")
	case 1:
		id = headFactor.Id
		term = headFactor
	default:
		id = factorId + factorsId
		termNode := db.Node{Id: id, Type: db.NodeTypeAnd}
		term = ParseNode{Node: &termNode}
		for _, factor := range append(tailFactors, headFactor) {
			relation := db.Relation{SourceId: termNode.Id, TargetId: factor.Id, Enforced: factor.Enforced, Prereq: factor.Prereq, Coreq: factor.Coreq, MinimumGrade: factor.MinimumGrade}
			relations = append(relations, relation)
		}
		nodes = append(nodes, termNode)
	}

	return id, term, nodes, courses, relations, nil
}

func Terms(tokens *[]Token) (id string, terms []ParseNode, nodes []db.Node, courses []db.Course, relations []db.Relation, err error) {
	if (*tokens)[0].Type == TokenOr {
		Eat(tokens, TokenOr)

		termId, headTerm, termNodes, termCourses, termRelations, err := Term(tokens)
		if err != nil {
			return "", nil, nil, nil, nil, err
		}

		termsId, tailTerms, termsNodes, termsCourses, termsRelations, err := Terms(tokens)
		if err != nil {
			return "", nil, nil, nil, nil, err
		}

		terms = append(tailTerms, headTerm)
		id = "|" + termId + termsId
		nodes = append(nodes, termNodes...)
		nodes = append(nodes, termsNodes...)
		courses = append(courses, termCourses...)
		courses = append(courses, termsCourses...)
		relations = append(relations, termRelations...)
		relations = append(relations, termsRelations...)
	}

	return id, terms, nodes, courses, relations, nil
}

func Factor(tokens *[]Token) (id string, factor ParseNode, nodes []db.Node, courses []db.Course, relations []db.Relation, err error) {
	switch (*tokens)[0].Type {
	case TokenRequisite:
		requisiteIdFlags, err := Eat(tokens, TokenRequisite)
		if err != nil {
			return "", ParseNode{}, nil, nil, nil, err
		}

		nodeId, flags, found := strings.Cut(requisiteIdFlags, "{")
		if !found {
			return "", ParseNode{}, nil, nil, nil, errors.New("Unable to determine requisite node id and flags")
		}
		flags = flags[:len(flags)-1]

		isCourse := db.Unflag(flags[0])
		isEnforced := db.Unflag(flags[1])
		isPrereq := db.Unflag(flags[2])
		isCoreq := db.Unflag(flags[3])
		minimumGrade := flags[4:]

		factorNode := db.Node{Id: nodeId, Type: db.NodeTypeValue}
		factor = ParseNode{Node: &factorNode, Enforced: &isEnforced, Prereq: &isPrereq, Coreq: &isCoreq, MinimumGrade: &minimumGrade}
		id = nodeId
		nodes = append(nodes, factorNode)

		if isCourse {
			subjectAreaPart, catalogNumber, found := strings.Cut(nodeId, "#")
			if !found {
				return "", ParseNode{}, nil, nil, nil, errors.New("Unable to determine requisite subject area and course catalog number")
			}
			course := db.Course{SubjectAreaCode: subjectAreaIdCodeMap[subjectAreaPart], CatalogNumber: catalogNumber, NodeId: nodeId}
			courses = append(courses, course)
		}
	case TokenLParen:
		Eat(tokens, TokenLParen)

		expressionId, expression, expressionNodes, expressionCourses, expressionRelations, err := Expression(tokens)
		if err != nil {
			return "", ParseNode{}, nil, nil, nil, err
		}

		_, err = Eat(tokens, TokenRParen)
		if err != nil {
			return "", ParseNode{}, nil, nil, nil, err
		}

		factor = expression
		id = "(" + expressionId + ")" // Parentheses necessary for conjunction in term
		nodes = append(nodes, expressionNodes...)
		courses = append(courses, expressionCourses...)
		relations = append(relations, expressionRelations...)
	default:
		return "", ParseNode{}, nil, nil, nil, errors.New("Invalid token")
	}

	return id, factor, nodes, courses, relations, nil
}

func Factors(tokens *[]Token) (id string, factors []ParseNode, nodes []db.Node, courses []db.Course, relations []db.Relation, err error) {
	if (*tokens)[0].Type == TokenAnd {
		Eat(tokens, TokenAnd)

		factorId, headFactor, factorNodes, factorCourses, factorRelations, err := Factor(tokens)
		if err != nil {
			return "", nil, nil, nil, nil, err
		}

		factorsId, tailFactors, factorsNodes, factorsCourses, factorsRelations, err := Factors(tokens)
		if err != nil {
			return "", nil, nil, nil, nil, err
		}

		factors = append(tailFactors, headFactor)
		id = "&" + factorId + factorsId
		nodes = append(nodes, factorNodes...)
		nodes = append(nodes, factorsNodes...)
		courses = append(courses, factorCourses...)
		courses = append(courses, factorsCourses...)
		relations = append(relations, factorRelations...)
		relations = append(relations, factorsRelations...)
	}

	return id, factors, nodes, courses, relations, nil
}

func (requisiteExpression RequisiteExpression) EvaluateForCourse(course db.Course) ([]db.Node, []db.Course, []db.Relation, error) {
	if len(requisiteExpression.string) == 0 {
		return nil, nil, nil, nil
	}

	tokens := requisiteExpression.Tokenize()
	nodes, courses, relations, err := Start(course, tokens)
	if err != nil {
		return nil, nil, nil, err
	}

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
