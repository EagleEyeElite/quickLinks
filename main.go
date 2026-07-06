package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// maxPathLen caps the incoming path length. A base64url-encoded 128-bit secret
// is 22 chars; vanity links are short. Anything much longer is junk or an
// enumeration probe — reject it before touching the database. Also bounds the
// work an unauthenticated client can force us to do (cheap DoS guard).
const maxPathLen = 128

func init() {
	var err error

	// Retrieve environment variables.
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB") // Database name
	// Host defaults to "db" (the in-cluster / compose service name) but is
	// overridable via POSTGRES_HOST — which the k8s Deployment already sets, and
	// which lets tests point at a local database. Optional POSTGRES_PORT too.
	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "db"
	}
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}

	// Set up the database connection string using environment variables.
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

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

// hashPath returns the lowercase hex SHA-256 of the path. The database stores
// only this hash (never the plaintext path), so a DB or backup leak cannot
// recover working secret links: reversing the hash of a 128-bit random path is
// infeasible. Lookups are therefore always an exact match on a fixed-length
// 64-char key, which also makes the query's work independent of whether the
// path exists — no timing side channel distinguishes a hit from a miss.
func hashPath(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	// A redirector only ever answers GETs (and HEADs). Reject everything else
	// without touching the database.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Deliberately do NOT log the requested path. The path IS the secret for
	// unguessable links; logging it would turn every log sink and backup into a
	// disclosure channel. We log only coarse, non-sensitive outcomes below.
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || len(path) > maxPathLen {
		serve404(w)
		return
	}

	var redirectURL string
	err := db.QueryRow("SELECT redirect_url FROM redirects WHERE path_hash = $1", hashPath(path)).Scan(&redirectURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			serve404(w)
		} else {
			// Genuine infrastructure error (DB down, etc.) — not an existence
			// signal. Log without the secret path.
			log.Printf("lookup failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func serve404(w http.ResponseWriter) {
	// Deliberately a plain-text "404 Not Found" returned as a proper HTTP 404
	// response (status code + text/plain body) — preferred over a styled HTML
	// error page. http.Error sets the status and writes the text in one call,
	// so there's no static asset to ship (see Dockerfile / removed static/).
	http.Error(w, "404 Not Found", http.StatusNotFound)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", redirectHandler)

	// Explicit timeouts instead of http.ListenAndServe's zero-value (no limit)
	// defaults, which leave the server open to slowloris-style resource
	// exhaustion from clients that trickle bytes and never finish a request.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Println("Server starting on port 8080...")
	log.Fatal(srv.ListenAndServe())
}
