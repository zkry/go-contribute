package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/alecthomas/template"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

// TODO: Change labeldata value names
// TODO: Selectable labels
// TODO: Improve UI
// TODO: Add gopher
// TODO: Unit tests
// TODO: JS -> GopherJS

type Config struct {
	dbDir   string
	lang    string
	ghToken string
	port    string
}

func getConfig() Config {
	c := Config{}

	c.dbDir = os.Getenv("GC_DB_DIR")
	if c.dbDir == "" {
		c.dbDir = "./tmp_db"
	}

	c.lang = os.Getenv("GC_LANG")
	if c.lang == "" {
		c.lang = "go"
	}

	c.ghToken = os.Getenv("GC_TOKEN")
	if c.ghToken == "" {
		log.Fatal("Must provide a github token in env var GC_TOKEN to proceed")
	}

	c.port = os.Getenv("PORT")
	if c.port == "" {
		c.port = ":80"
	} else if !strings.HasPrefix(c.port, ":") {
		c.port = ":" + c.port
	}

	return c
}

func labelScraper(cl *github.Client, repoStream <-chan string, db *sql.DB) {
	for {
		repo := <-repoStream
		repoPts := strings.Split(repo, "/")
		log.Printf("Getting the labels for %s\n", repo)

		// Wait for our api limit to not be zero.
		waitForRemainingLimit(cl, true, 3)

		// Get the list if labels for a repo
		opt := &github.ListOptions{Page: 0, PerPage: 50}
		labels, _, err := cl.Issues.ListLabels(context.TODO(), repoPts[0], repoPts[1], opt)
		if err != nil {
			fmt.Println("Could not obtain label information")
		}

		for _, l := range labels {
			if isHelpfulLabel(l.GetName()) {
				opt := &github.IssueListByRepoOptions{
					State:  "open",
					Labels: []string{l.GetName()},
					ListOptions: github.ListOptions{
						Page:    0,
						PerPage: 50,
					},
				}

				issues, _, err := cl.Issues.ListByRepo(context.TODO(), repoPts[0], repoPts[1], opt)
				if err != nil {
					log.Println("could not search issues:" + err.Error())
				}
				issueCt := len(issues)

				err = insertLabel(db, repo, l.GetName(), l.GetColor(), issueCt)
				if err != nil {
					log.Println("unable to execute label query:", err)
				}
			}
		}

	}
}

func query(cl *github.Client, db *sql.DB, pgNo, starNo int, repoStream chan<- string, lang string) (lastPage int, atStars int, err error) {
	opt := &github.SearchOptions{Sort: "stars", Order: "desc"}

	opt.ListOptions.Page = pgNo
	opt.ListOptions.PerPage = 10

	q := fmt.Sprintf("language:%s", lang)
	if starNo >= 0 {
		q += fmt.Sprintf(" stars:<=%d", starNo)
	}
	log.Println("Search query:", q)
	sRes, resp, err := cl.Search.Repositories(context.TODO(), q, opt)
	if err != nil {
		return 0, 0, err
	}

	for _, r := range sRes.Repositories {
		repoStream <- r.GetOwner().GetLogin() + "/" + r.GetName()

		if err := insertRepo(db, r.GetFullName(), r.GetStargazersCount(), r.GetForksCount(), r.GetDescription()); err != nil {
			log.Println("unable to insert repo into DB:", err.Error())
		}
		atStars = r.GetStargazersCount()
	}

	return resp.LastPage, atStars, nil
}

func repoScraper(cl *github.Client, db *sql.DB, repoStream chan<- string, lang string) {
	atPage := 0
	atStars := -1
	log.Println("starting from page", atPage)
	for {
		waitForRemainingLimit(cl, false, 5)

		lastPage, ls, err := query(cl, db, atPage, atStars, repoStream, lang)
		if err != nil {
			log.Printf("Unable to query github: %v\n", err)
		}
		log.Printf("Queried page %d/%d\n", atPage, lastPage)

		// If we finished all of the pages...
		atPage++
		if atPage == lastPage {
			atPage = 0
			// This is to prevent an infinite loop: force the searched star ct to
			// go down if it is the same. This happens for low star numbers
			if atStars == ls {
				ls--
			}
			atStars = ls
		}
	}
}

func initializeDB(cfg Config) *sql.DB {
	filename := cfg.dbDir + "/helpwanted.db"
	_, existErr := os.Stat(filename)
	db, err := sql.Open("sqlite3", cfg.dbDir+"/helpwanted.db")
	if err != nil {
		panic("Could not open db:" + err.Error())
	}

	if os.IsNotExist(existErr) {
		if err := createTables(db); err != nil {
			panic(err)
		}
		return db
	}

	return db
}

func main() {
	config := getConfig()

	// Setup github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Setup sqldb
	db := initializeDB(config)

	// Set up the concurrent routines to gather the data
	log.Println("Starting up scraper")
	repoStream := make(chan string, 10)
	go repoScraper(client, db, repoStream, config.lang)
	go labelScraper(client, repoStream, db)

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler(db))
	r.PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	log.Fatal(http.ListenAndServe(config.port, r))
}

type labelData struct {
	LabelName     string
	LabelCt       int
	LabelColor    string
	LabelTxtColor string
}
type repositoryData struct {
	Name        string
	StarCt      int
	ForkCt      int
	Description string
	Labels      []labelData
}
type HelpPageData struct {
	Repos []repositoryData
}

// HomeHandler manages the simple index page of website
func HomeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := getHelpPageData(db)
		if err != nil {
			log.Println("an error occured when querying db:", err)
		}
		t, _ := template.ParseFiles("./templates/index.html")
		t.Execute(w, data)
	}
}
