package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adesokanayo/ultimateservice/schema"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Product struct {
	ID          string    `db:"product_id"json:"id"`
	Name        string    `db:"name"json:"name"`
	Cost        int       `db:"cost"json:"cost"`
	Quantity    int       `db:"quantity"json:"quantity"`
	DateCreated time.Time `db:"date_created"json:"date_created"`
	DateUpdated time.Time `db:"date_updated"json:"date_updated"`
}

func main() {

	// =================================
	log.Printf("main: Started")
	defer log.Println("main :Completed")
	// =================================

	//Setup dependencies

	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	flag.Parse()
	switch flag.Arg(0) {
	case "migrate":
		if err := schema.Migrate(db); err != nil {
			log.Fatal(err)
		}
		log.Println("Migration Completed")
		return
	case "seed":
		if err := schema.Seed(db); err != nil {
			log.Fatal(err)
			return
		}
		log.Println("Migration Completed")
	}

	ps := ProductService{db}
	// =================================
	api := http.Server{
		Addr:         "localhost:8000",
		Handler:      http.HandlerFunc(ps.List),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("main : API Listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		log.Fatalf("main: error listening and serving: %s", err)
	case <-shutdown:
		log.Println("main: start shudown")

		const timeout = 5 * time.Second

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := api.Shutdown(ctx)
		if err != nil {
			log.Printf("main:Graceful shutdown did not completed in %v: %s", timeout, err)
			err = api.Close()

		}

		if err != nil {
			log.Fatalf("main: could not stop server gracefully: %v", err)
		}

	}
}

func EchoSample(w http.ResponseWriter, r *http.Request) {

	n := rand.Intn(1000)
	log.Println("Start", n)
	defer log.Println("end", n)

	time.Sleep(3 * time.Second)
	w.Write([]byte(fmt.Sprintf("You called %s on %s", r.Method, r.URL.Path)))
}

type ProductService struct {
	db *sqlx.DB
}

func (p *ProductService) List(w http.ResponseWriter, r *http.Request) {
	var list []Product

	const q = `SELECT product_id, name, cost, quantity, date_updated, date_created FROM products`
	if err := p.db.Select(&list, q); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error querying database", err)
		return
	}
	if false {
		list = []Product{
			{Name: "Comic Books", Cost: 75, Quantity: 50},
			{Name: "Mcdonals", Cost: 25, Quantity: 120},
		}
	}
	data, err := json.Marshal(list)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error marshalling", err)
		return
	}

	w.Header().Add("content-type", "application/json")

	if _, err := w.Write(data); err != nil {
		log.Println("error writing", err)
	}

	w.WriteHeader(http.StatusOK)
}

func openDB() (*sqlx.DB, error) {
	q := url.Values{}
	q.Set("sslmode", "disable")
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("postgres", "postgres"),
		Host:     "localhost",
		Path:     "postgres",
		RawQuery: q.Encode(),
	}

	return sqlx.Open("postgres", u.String())
}
