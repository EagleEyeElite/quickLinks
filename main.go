package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	var err error
	// Set up the database connection.
	db, err = sql.Open("postgres", "postgres://user:password@db/quick_links?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	// Check the database connection.
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("handling hello request...")

	// Query the database.
	var redirectURL string
	err := db.QueryRow("SELECT redirect_url FROM redirects WHERE path = $1", "home").Scan(&redirectURL)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			http.Error(w, "Not Found", http.StatusNotFound)
		default:
			log.Printf("Failed to execute query: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Send back the redirect URL.
	_, err = fmt.Fprintf(w, "Redirect to: %s", redirectURL)
	if err != nil {
		log.Printf("Failed to write response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", helloHandler) // Set the handler for the root URL

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Start the server on port 8080
}
