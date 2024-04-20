package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	var err error

	// Retrieve environment variables.
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB") // Database name

	// Set up the database connection string using environment variables.
	connectionString := fmt.Sprintf("postgres://%s:%s@db/%s?sslmode=disable", postgresUser, postgresPassword, postgresDB)

	// Open the database connection.
	db, err = sql.Open("postgres", connectionString)
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

	path := r.URL.Path[1:]
	if path == "" {
		http.Error(w, "No path specified", http.StatusBadRequest)
		return
	}

	var redirectURL string
	err := db.QueryRow("SELECT redirect_url FROM redirects WHERE path = $1", path).Scan(&redirectURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			serve404(w)
		} else {
			log.Printf("Failed to execute query: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Redirecting to: %v", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func serve404(w http.ResponseWriter) {
	log.Printf("Serving 404 page")
	page, err := os.ReadFile("static/404.html")
	if err != nil {
		log.Printf("Error loading 404 page: %v", err)
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	if _, err := w.Write(page); err != nil {
		// Log the error if the response can't be written
		log.Printf("Failed to write 404 page to response: %v", err)
		// Return a standard HTTP 500 Internal Server Error as a fallback
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", redirectHandler) // Set the handler for the root URL
	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Start the server on port 8080
}
