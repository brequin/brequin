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
const modelTemplate = `{"Term":"%v","SubjectAreaCode":"%v","CatalogNumber":"%v","IsRoot":true,"Path":"0"}`

var subjectAreaNameCodeMap map[string]string

func ScrapeNodesCoursesRelations(quarter db.Quarter, subjectArea db.SubjectArea) ([]db.Node, []db.Course, []db.Relation, error) {
	catalogNumbers, err := ScrapeCourseCatalogNumbers(quarter.Code, subjectArea.Code)
	if err != nil {
		log.Println("Unable to determine course catalog numbers")
		return nil, nil, nil, err
	}

	var nodes []db.Node
	var courses []db.Course
	var relations []db.Relation
	var nodesMutex sync.Mutex
	var coursesMutex sync.Mutex
	var relationsMutex sync.Mutex

	var wg sync.WaitGroup
	for _, catalogNumber := range catalogNumbers {
		wg.Add(1)

		go func(n string) {
			defer wg.Done()

			request, err := http.NewRequest("GET", courseSummaryUrl, nil)
			if err != nil {
				log.Println("Unable to make new course summary request")
				return
			}

			formattedCatalogNumber, err := FormatCatalogNumber(n)
			if err != nil {
				log.Printf("Unable to format course catalog number: %v\n", n)
				return
			}
			model := fmt.Sprintf(modelTemplate, quarter.Code, subjectArea.Code, formattedCatalogNumber)

			query := request.URL.Query()
			query.Add("model", model)
			query.Add("filterFlags", "{}")
			request.URL.RawQuery = query.Encode()

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				log.Println("Unable to get course summary")
				return
			}
			defer response.Body.Close()

			document, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				log.Println("Unable to make document from course summary")
				return
			}

			classInfoDiv := document.Find("div.class-not-checked.class-info").First()

			fakeClassId, exists := classInfoDiv.Attr("id")
			if !exists {
				log.Printf("Unable to determine fake class id for class: %v %v\n", subjectArea.Code, n)
				return
			}

			label, err := classInfoDiv.Find("div#" + fakeClassId + "-enroll").Find("label").Html()
			if err != nil {
				log.Println("Unable to determine course label")
				return
			}
			label = html.UnescapeString(label)

			_, after, found := strings.Cut(label, n+" - ")
			if !found {
				log.Println("Unable to determine course name")
				return
			}

			nodeId := db.ValueNodeId(subjectArea.Code, n)
			nodesMutex.Lock()
			nodes = append(nodes, db.Node{Id: nodeId, Type: "value"})
			nodesMutex.Unlock()

			split := strings.Split(after, " ")
			name := strings.Join(split[:len(split)-2], " ")

			course := db.Course{SubjectAreaCode: subjectArea.Code, CatalogNumber: n, Name: name, NodeId: nodeId}
			coursesMutex.Lock()
			courses = append(courses, course)
			coursesMutex.Unlock()

			classDetailPath, exists := classInfoDiv.Find("div#" + fakeClassId + "-section").Find("a").Attr("href")
			if !exists {
				log.Println("Unable to determine class detail path")
				return
			}

			classDetailTooltipUrl := strings.Replace("https://sa.ucla.edu"+classDetailPath, "ClassDetail", "ClassDetailTooltip", 1)
			requisiteExpression, err := ScrapeRequisiteExpression(classDetailTooltipUrl)
			if err != nil {
				log.Println("Unable to determine requisite expression from class detail tooltip")
				return
			}

			fmt.Println(requisiteExpression)
			tooltipNodes, tooltipCourses, tooltipRelations, err := ParseRequisites(course, requisiteExpression)
			if err != nil {
				log.Println("Unable to parse requisite expression")
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
		}(catalogNumber)
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

	nodes, courses, relations, err := ScrapeNodesCoursesRelations(db.Quarter{Code: "24W", Name: "Winter 2024"}, db.SubjectArea{Code: "EC ENGR", Name: "Electrical and Computer Engineering"})
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
					log.Println(err)
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
