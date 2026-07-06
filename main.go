package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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

// clickEvent is one resolve attempt. It is logged as a single JSON line to
// stdout (for live `kubectl logs -f`) and persisted to the click_events table
// for analytics ("who clicked what, when, how often"). The requested path is
// recorded verbatim — this is an intentional access log, so a successful secret
// path does appear here.
type clickEvent struct {
	Time      time.Time `json:"time"`
	Outcome   string    `json:"outcome"` // hit | miss | rejected | error
	Path      string    `json:"path"`
	Label     string    `json:"label,omitempty"`
	ClientIP  string    `json:"client_ip,omitempty"`
	Country   string    `json:"country,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Referrer  string    `json:"referrer,omitempty"`
}

// newEvent captures the non-body request context up front (before we respond),
// so the record() goroutine never touches the *http.Request after the handler
// returns. Cf-Connecting-IP / CF-IPCountry are set by Cloudflare and trustworthy
// here because the cloudflare-only middleware guarantees traffic came via CF.
func newEvent(r *http.Request, path string) *clickEvent {
	return &clickEvent{
		Time:      time.Now().UTC(),
		Path:      path,
		ClientIP:  r.Header.Get("Cf-Connecting-IP"),
		Country:   r.Header.Get("Cf-IPCountry"),
		UserAgent: r.UserAgent(),
		Referrer:  r.Referer(),
	}
}

// record emits the event to stdout immediately, then persists it in the
// background. The DB write is deliberately off the request path: a slow or
// failing insert must never delay or break a redirect (best-effort logging).
func record(ev *clickEvent) {
	if b, err := json.Marshal(ev); err == nil {
		log.Printf("click %s", b)
	}
	go func() {
		if err := insertEvent(ev); err != nil {
			log.Printf("click log insert failed: %v", err)
		}
	}()
}

// insertEvent persists one event. Empty strings are stored as NULL.
func insertEvent(ev *clickEvent) error {
	_, err := db.Exec(
		`INSERT INTO click_events (ts, outcome, path, label, client_ip, country, user_agent, referrer)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		ev.Time, ev.Outcome, ev.Path,
		nullify(ev.Label), nullify(ev.ClientIP), nullify(ev.Country),
		nullify(ev.UserAgent), nullify(ev.Referrer),
	)
	return err
}

func nullify(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	ev := newEvent(r, path)
	defer record(ev)

	// A redirector only ever answers GETs (and HEADs). Reject everything else
	// without touching the database.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		ev.Outcome = "rejected"
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if path == "" || len(path) > maxPathLen {
		ev.Outcome = "rejected"
		serve404(w)
		return
	}

	var redirectURL string
	var label sql.NullString
	err := db.QueryRow("SELECT redirect_url, label FROM redirects WHERE path_hash = $1", hashPath(path)).Scan(&redirectURL, &label)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ev.Outcome = "miss"
			serve404(w)
		} else {
			// Genuine infrastructure error (DB down, etc.) — not an existence signal.
			ev.Outcome = "error"
			log.Printf("lookup failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	ev.Outcome = "hit"
	ev.Label = label.String
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
