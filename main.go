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

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request for path: %s", r.URL.Path)

	// Remove the leading slash from path and check for empty path
	path := r.URL.Path[1:]
	if path == "" {
		http.Error(w, "No path specified", http.StatusBadRequest)
		return
	}

	// Query the database for the redirect URL based on the path
	var redirectURL string
	err := db.QueryRow("SELECT redirect_url FROM redirects WHERE path = $1", path).Scan(&redirectURL)
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

	// Redirect to the fetched URL
	log.Printf("redirecting to: %v", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func main() {
	http.HandleFunc("/", redirectHandler) // Set the handler for the root URL

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Start the server on port 8080
}
