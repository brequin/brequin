package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/brequin/brequin/scrape/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const socUrl = "https://sa.ucla.edu/ro/public/soc/"

func main() {
	response, err := http.Get(socUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	quarterOptions := document.Find("select#optSelectTerm")
	codes := quarterOptions.Find("option").Map(func(i int, option *goquery.Selection) string {
		code, exists := option.Attr("value")
		if !exists {
			log.Fatal("Unable to determine quarter code")
		}
		return code
	})
	names := quarterOptions.Find("option").Map(func(i int, option *goquery.Selection) string {
		name, err := option.Html()
		if err != nil {
			log.Fatal(err)
		}
		return name
	})
	if len(names) != len(codes) {
		log.Fatal("Quarter names and codes do not match")
	}

	var quarters []db.Quarter
	for i := range codes {
		quarters = append(quarters, db.Quarter{Code: codes[i], Name: names[i]})
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_CONNECTION_STRING"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	database := db.Database{Pool: pool}

	if err := database.InsertQuarters(quarters); err != nil {
		log.Fatal(err)
	}
}
