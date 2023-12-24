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

	/*
		var nodes []db.Node
		var courses []db.Course
		var relations []db.Relation
		var nodesMutex sync.Mutex
		var coursesMutex sync.Mutex
		var relationsMutex sync.Mutex
	*/

	classInfoDivs := document.Find("div.class-not-checked.class-info")
	var wg sync.WaitGroup
	for _, courseInfoDivNode := range classInfoDivs.Nodes {
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

			split := strings.Split(after, " ")
			name := strings.Join(split[:len(split)-2], " ")

			fmt.Println(subjectArea.Code, catalogNumber, name)

			classDetailPath, exists := classInfoDiv.Find("div#" + fakeClassId + "-section").Find("a").Attr("href")
			if !exists {
				log.Print("Unable to determine class detail path")
				return
			}

			classDetailTooltipUrl := strings.Replace("https://sa.ucla.edu"+classDetailPath, "ClassDetail", "ClassDetailTooltip", 1)
			fmt.Println(classDetailTooltipUrl)
		}(courseInfoDivNode)
	}
	wg.Wait()

	return nil, nil, nil, nil
}

func main() {
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
