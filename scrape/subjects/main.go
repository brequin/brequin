package main

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/brequin/brequin/scrape/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const subjectSearchUrl = "https://sa.ucla.edu/ro/ClassSearch/Public/Search/GetSimpleSearchData"

type SubjectAreaOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func ParseSubjectAreas(content []byte) ([]db.SubjectArea, error) {
	re := regexp.MustCompile(`SearchPanelSetup\('(\[\{.*\}\])'`)
	encodedOptions := []byte(html.UnescapeString(string(re.FindSubmatch(content)[1])))

	var subjectAreaOptions []SubjectAreaOption
	if err := json.Unmarshal(encodedOptions, &subjectAreaOptions); err != nil {
		return nil, err
	}

	var subjectAreas []db.SubjectArea
	for _, subjectAreaOption := range subjectAreaOptions {
		code := strings.TrimSpace(subjectAreaOption.Value)

		labelCode := "(" + code + ")"
		name := strings.TrimSpace(strings.ReplaceAll(subjectAreaOption.Label, labelCode, ""))

		subjectAreas = append(subjectAreas, db.SubjectArea{Code: code, Name: name})
	}

	return subjectAreas, nil
}

func ScrapeSubjectAreas(quarterCode string) ([]db.SubjectArea, error) {
	request, err := http.NewRequest("GET", subjectSearchUrl, nil)
	if err != nil {
		return nil, err
	}

	query := request.URL.Query()
	query.Add("term_cd", quarterCode)
	query.Add("search_type", "subject")
	request.URL.RawQuery = query.Encode()

	// Required
	request.Header.Add("X-Requested-With", "XMLHttpRequest")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	subjectAreas, err := ParseSubjectAreas(content)
	if err != nil {
		return nil, err
	}
	return subjectAreas, nil
}

func main() {
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

	var wg sync.WaitGroup
	for _, quarter := range quarters {
		wg.Add(1)

		go func(q db.Quarter) {
			defer wg.Done()

			subjectAreas, err := ScrapeSubjectAreas(q.Code)
			if err != nil {
				log.Fatal(err)
			}

			if err := database.InsertSubjectAreas(subjectAreas); err != nil {
				log.Fatal(err)
			}

			if err := database.InsertQuarterSubjectAreas(q, subjectAreas); err != nil {
				log.Fatal(err)
			}
		}(quarter)
	}
	wg.Wait()
}
