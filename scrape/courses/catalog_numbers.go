package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

const subjectCoursesUrl = "https://sa.ucla.edu/ro/public/soc/Results"
const courseTitlesViewUrl = "https://sa.ucla.edu/ro/public/soc/Results/CourseTitlesView"

func ScrapePageCourseCatalogNumbers(quarterCode string, subjectAreaCode string, pageNumber int) ([]string, error) {
	request, err := http.NewRequest("GET", courseTitlesViewUrl, nil)
	if err != nil {
		return nil, err
	}

	const modelTemplate = `{"term_cd":"%v","subj_area_cd":"%v"}`
	model := fmt.Sprintf(modelTemplate, quarterCode, subjectAreaCode)

	query := request.URL.Query()
	query.Add("search_by", "subject")
	query.Add("model", model)
	query.Add("pageNumber", strconv.Itoa(pageNumber))
	query.Add("filterFlags", "{}")
	request.URL.RawQuery = query.Encode()

	// Required
	request.Header.Add("X-Requested-With", "XMLHttpRequest")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	var catalogNumbers []string

	courseTitleButtons := document.Find("div.class-title").Find("button")
	courseTitleButtons.Each(func(i int, courseTitleButton *goquery.Selection) {
		courseTitle, err := courseTitleButton.Html()
		if err != nil {
			log.Println("Unable to determine course title")
			return
		}
		courseTitle = html.UnescapeString(courseTitle)

		catalogNumber, _, found := strings.Cut(courseTitle, " - ")
		if !found {
			log.Println("Unable to determine course catalog number")
			return
		}

		catalogNumbers = append(catalogNumbers, catalogNumber)
	})

	return catalogNumbers, nil
}

func ScrapeCourseCatalogNumbers(quarterCode string, subjectAreaCode string) ([]string, error) {
	request, err := http.NewRequest("GET", subjectCoursesUrl, nil)
	if err != nil {
		return nil, err
	}

	query := request.URL.Query()
	query.Add("t", quarterCode)
	query.Add("sBy", "subject")
	query.Add("subj", subjectAreaCode)
	request.URL.RawQuery = query.Encode()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	pageCountInput := document.Find("input#pageCount")
	pageCountText, exists := pageCountInput.Attr("value")
	if !exists {
		msg := fmt.Sprintf("Unable to determine page count for quarter %v, subject area %v; assuming 0", quarterCode, subjectAreaCode)
		log.Println(msg)

		return []string{}, nil
	}
	pageCount, err := strconv.Atoi(pageCountText)
	if err != nil {
		msg := fmt.Sprintf("Unable to convert page count text '%v' to int", pageCountText)
		return nil, errors.New(msg)
	}

	var courseCatalogNumbers []string
	var coursesMutex sync.Mutex

	var wg sync.WaitGroup
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		wg.Add(1)

		go func(p int) {
			defer wg.Done()

			pageCourseCatalogNumbers, err := ScrapePageCourseCatalogNumbers(quarterCode, subjectAreaCode, p)
			if err != nil {
				log.Println(err)
				return
			}

			coursesMutex.Lock()
			courseCatalogNumbers = append(courseCatalogNumbers, pageCourseCatalogNumbers...)
			coursesMutex.Unlock()
		}(pageNumber)
	}
	wg.Wait()

	return courseCatalogNumbers, nil
}

func FormatCatalogNumber(catalogNumber string) string {
	re := regexp.MustCompile(`([[:upper:]]*)([[:digit:]]*)([[:upper:]]*)`)
	submatches := re.FindStringSubmatch(catalogNumber)
	prefix := submatches[1]
	suffix := submatches[3]
	number, err := strconv.Atoi(submatches[2])
	if err != nil {
		log.Println("Unable to determine course catalog number digits; assuming 0")
		number = 0
		suffix = prefix
		prefix = ""
	}
	return fmt.Sprintf("%04d%-2s%-2s", number, suffix, prefix)
}
