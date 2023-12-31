package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/brequin/brequin/scrape/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const subjectAreasUrl = "https://api.ucla.edu/sis/publicapis/course/getallcourses"
const courseDetailsUrl = "https://api.ucla.edu/sis/publicapis/course/getcoursedetail"

type SubjectAreaEntry struct {
	Code string `json:"subj_area_cd"`
}

type CourseEntry struct {
	Title       string `json:"course_title"`
	Units       string `json:"unt_rng"`
	Level       string `json:"crs_career_lvl_nm"`
	Description string `json:"crs_desc"`
}

func ScrapeCurrentSubjectAreas() ([]SubjectAreaEntry, error) {
	response, err := http.Get(subjectAreasUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseJson, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var subjectAreaEntries []SubjectAreaEntry
	if err := json.Unmarshal(responseJson, &subjectAreaEntries); err != nil {
		return nil, err
	}

	return subjectAreaEntries, nil
}

func ScrapeCoursesDetails(subjectAreaCode string) ([]db.CourseDetails, error) {
	request, err := http.NewRequest("GET", courseDetailsUrl, nil)
	if err != nil {
		return nil, err
	}

	query := request.URL.Query()
	query.Add("subjectarea", subjectAreaCode)
	request.URL.RawQuery = query.Encode()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseJson, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var courseEntries []CourseEntry
	if err := json.Unmarshal(responseJson, &courseEntries); err != nil {
		return nil, err
	}

	var coursesDetails []db.CourseDetails
	for _, courseEntry := range courseEntries {
		catalogNumber, name, found := strings.Cut(courseEntry.Title, ". ")
		if !found {
			return nil, errors.New("Unable to determine course catalog number and name")
		}

		level := strings.TrimSuffix(courseEntry.Level, " Courses")

		courseDetails := db.CourseDetails{
			SubjectAreaCode: strings.TrimSpace(subjectAreaCode),
			CatalogNumber:   strings.TrimSpace(catalogNumber),
			Name:            strings.TrimSpace(name),
			Units:           strings.TrimSpace(courseEntry.Units),
			Level:           strings.TrimSpace(level),
			Description:     strings.TrimSpace(courseEntry.Description),
		}
		coursesDetails = append(coursesDetails, courseDetails)
	}

	return coursesDetails, nil
}

func main() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	database := db.Database{Pool: pool}

	subjectAreaEntries, err := ScrapeCurrentSubjectAreas()
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, subjectAreaEntry := range subjectAreaEntries {
		wg.Add(1)

		go func(s SubjectAreaEntry) {
			defer wg.Done()

			coursesDetails, err := ScrapeCoursesDetails(s.Code)
			if err != nil {
				log.Println("Unable to get course details for subject area: " + s.Code)
				return
			}

			msg := fmt.Sprintf("%v: Scraped details for %v courses", s.Code, len(coursesDetails))
			log.Println(msg)

			if err := database.InsertCoursesDetails(coursesDetails); err != nil {
				log.Fatal(err)
			}
		}(subjectAreaEntry)
	}
	wg.Wait()
}
