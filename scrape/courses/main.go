package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/brequin/brequin/scrape/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/html"
)

const courseSummaryUrl = "https://sa.ucla.edu/ro/public/soc/Results/GetCourseSummary"

// Path is required filler
const modelTemplate = `{"Term":"%v","SubjectAreaCode":"%v","IsRoot":true,"Path":"0"}`

type Requisite struct {
	NodeId       string
	Enforced     bool
	Prereq       bool
	Coreq        bool
	MinimumGrade string
}

var subjectAreaNameCodeMap map[string]string

func ScrapeClassDetailTooltip(course db.Course, classDetailTooltipUrl string) ([]db.Node, []db.Course, []db.Relation, error) {
	request, err := http.NewRequest("GET", classDetailTooltipUrl, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	// Required
	request.Header.Add("X-Requested-With", "XMLHttpRequest")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, nil, err
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, nil, nil, err
	}

	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation

	//var requisites []Requisite
	var reqExpBuilder strings.Builder

	requisiteRows := document.Find("table.requisites_content").Find("tbody").Find("tr.requisite")
	for _, root := range requisiteRows.Nodes {
		requisiteRow := goquery.NewDocumentFromNode(root)

		requisiteDataNodes := requisiteRow.Find("td").Nodes

		expPart, err := goquery.NewDocumentFromNode(requisiteDataNodes[0]).Html()
		if err != nil {
			fmt.Print("Unable to determine requisite expression")
			return nil, nil, nil, err
		}
		/*
			minimumGrade, err := goquery.NewDocumentFromNode(requisiteDataNodes[1]).Html()
			if err != nil {
				fmt.Print("Unable to determine requisite minimum grade")
				return nil, nil, nil, err
			}
			prereqText, err := goquery.NewDocumentFromNode(requisiteDataNodes[2]).Html()
			if err != nil {
				log.Print("Unable to determine whether requisite is a prerequisite")
				return nil, nil, nil, err
			}
			coreqText, err := goquery.NewDocumentFromNode(requisiteDataNodes[3]).Html()
			if err != nil {
				log.Print("Unable to determine whether requisite is a corequisite")
				return nil, nil, nil, err
			}
		*/

		//enforced := goquery.NewDocumentFromNode(requisiteDataNodes[4]).Find("div.icon-exclamation-sign").Length() == 1
		//prereq := prereqText == "Yes"
		//coreq := coreqText == "Yes"

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

		expPart = strings.ReplaceAll(expPart, " ", "")
		fmt.Fprint(&reqExpBuilder, expPart)
		if foundAnd {
			fmt.Fprint(&reqExpBuilder, "&&")
		}
		if foundOr {
			fmt.Fprint(&reqExpBuilder, "||")
		}
	}
	fmt.Printf("%v: %v\n", course.CatalogNumber, reqExpBuilder.String())

	return nodes, courses, relations, nil
}

func ScrapeNodesCoursesRelations(quarter db.Quarter, subjectArea db.SubjectArea) ([]db.Node, []db.Course, []db.Relation, error) {
	request, err := http.NewRequest("GET", courseSummaryUrl, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	model := fmt.Sprintf(modelTemplate, quarter.Code, subjectArea.Code)

	query := request.URL.Query()
	query.Add("model", model)
	query.Add("filterFlags", "{}")
	request.URL.RawQuery = query.Encode()

	fmt.Println(request.URL.RawQuery)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, nil, err
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, nil, nil, err
	}

	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation
	var nodesMutex sync.Mutex
	var coursesMutex sync.Mutex
	var relationsMutex sync.Mutex

	classInfoDivs := document.Find("div.class-not-checked.class-info")
	var wg sync.WaitGroup
	for _, classInfoDivNode := range classInfoDivs.Nodes {
		wg.Add(1)

		go func(root *html.Node) {
			defer wg.Done()

			classInfoDiv := goquery.NewDocumentFromNode(root)

			fakeClassId, exists := classInfoDiv.Attr("id")
			if !exists {
				log.Print("Unable to determine fake class id")
				return
			}

			label, err := classInfoDiv.Find("div#" + fakeClassId + "-enroll").Find("label").Html()
			if err != nil {
				log.Print("Unable to determine course label")
				return
			}

			// Skip all but first section
			if !strings.HasSuffix(label, " 1") {
				return
			}

			before, after, found := strings.Cut(label, " - ")
			if !found {
				log.Print("Unable to determine course catalog number and name")
				return
			}

			prefix := fmt.Sprintf("Select %v (%v) ", subjectArea.Name, subjectArea.Code)
			catalogNumber, found := strings.CutPrefix(before, prefix)
			if !found {
				log.Print("Unable to determine course catalog number")
				return
			}

			nodeId := db.ValueNodeId(subjectArea.Code, catalogNumber)
			nodesMutex.Lock()
			nodes = append(nodes, db.Node{Id: nodeId, Type: "value"})
			nodesMutex.Unlock()

			split := strings.Split(after, " ")
			name := strings.Join(split[:len(split)-2], " ")

			course := db.Course{SubjectAreaCode: subjectArea.Code, CatalogNumber: catalogNumber, Name: name, NodeId: nodeId}
			coursesMutex.Lock()
			courses = append(courses, course)
			coursesMutex.Unlock()

			classDetailPath, exists := classInfoDiv.Find("div#" + fakeClassId + "-section").Find("a").Attr("href")
			if !exists {
				log.Print("Unable to determine class detail path")
				return
			}

			classDetailTooltipUrl := strings.Replace("https://sa.ucla.edu"+classDetailPath, "ClassDetail", "ClassDetailTooltip", 1)
			tooltipNodes, tooltipCourses, tooltipRelations, err := ScrapeClassDetailTooltip(course, classDetailTooltipUrl)
			if err != nil {
				log.Print("Unable to scrape class detail tooltip")
				return
			}

			nodesMutex.Lock()
			nodes = append(nodes, tooltipNodes...)
			nodesMutex.Unlock()

			coursesMutex.Lock()
			courses = append(courses, tooltipCourses...)
			coursesMutex.Unlock()

			relationsMutex.Lock()
			relations = append(relations, tooltipRelations...)
			relationsMutex.Unlock()
		}(classInfoDivNode)
	}
	wg.Wait()

	return nodes, courses, relations, nil
}

func main() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	database := db.Database{Pool: pool}

	subjectAreas, err := database.ListSubjectAreas()
	if err != nil {
		log.Fatal(err)
	}

	subjectAreaNameCodeMap = make(map[string]string)
	for _, subjectArea := range subjectAreas {
		subjectAreaNameCodeMap[subjectArea.Name] = subjectArea.Code
	}

	nodes, courses, relations, err := ScrapeNodesCoursesRelations(db.Quarter{Code: "24W", Name: "Winter 2024"}, db.SubjectArea{Code: "MATH", Name: "Mathematics"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(len(nodes), len(courses), len(relations))
}

func foo() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	database := db.Database{Pool: pool}

	quarters, err := database.ListQuarters()
	if err != nil {
		log.Fatal(err)
	}

	subjectAreas, err := database.ListSubjectAreas()
	if err != nil {
		log.Fatal(err)
	}

	subjectAreaNameCodeMap = make(map[string]string)
	for _, subjectArea := range subjectAreas {
		subjectAreaNameCodeMap[subjectArea.Name] = subjectArea.Code
	}

	for _, quarter := range quarters {
		subjectAreas, err := database.ListQuarterSubjectAreas(quarter)
		if err != nil {
			log.Fatal(err)
		}

		var wg sync.WaitGroup
		for _, subjectArea := range subjectAreas {
			wg.Add(1)

			go func(s db.SubjectArea) {
				defer wg.Done()

				nodes, courses, relations, err := ScrapeNodesCoursesRelations(quarter, s)
				if err != nil {
					log.Print(err)
					return
				}
				fmt.Print(len(nodes), len(courses), len(relations))
				/*
					if err := database.InsertNodes(nodes); err != nil {
						log.Fatal(err)
					}

					if err := database.InsertCourses(courses); err != nil {
						log.Fatal(err)
					}

					if err := database.InsertRelations(relations); err != nil {
						log.Fatal(err)
					}
				*/
			}(subjectArea)
		}
		wg.Wait()
	}
}
