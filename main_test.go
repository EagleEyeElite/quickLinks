package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// These tests drive the real redirectHandler against the database opened by
// init() (point it at a local Postgres via POSTGRES_HOST/PORT env vars). They
// verify the security-relevant behaviour (hashed lookup, 404-vs-303, method
// guard) and the access logging (stdout JSON + click_events persistence).

const (
	secretPath = "wWD56zKM3ft5sr7p8xGjmQ" // a sample 128-bit path
	destURL    = "https://example.com/destination"
	sampleLbl  = "sample-secret"
)

func seed(t *testing.T) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS redirects (
		path_hash CHAR(64) PRIMARY KEY, redirect_url TEXT NOT NULL, label VARCHAR(255))`); err != nil {
		t.Fatalf("create redirects: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS click_events (
		id BIGSERIAL PRIMARY KEY, ts TIMESTAMPTZ NOT NULL, outcome VARCHAR(16) NOT NULL,
		path TEXT, label VARCHAR(255), client_ip TEXT, country VARCHAR(8),
		user_agent TEXT, referrer TEXT)`); err != nil {
		t.Fatalf("create click_events: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO redirects (path_hash, redirect_url, label) VALUES ($1, $2, $3)
		ON CONFLICT (path_hash) DO UPDATE SET redirect_url = EXCLUDED.redirect_url, label = EXCLUDED.label`,
		hashPath(secretPath), destURL, sampleLbl); err != nil {
		t.Fatalf("seed row: %v", err)
	}
}

func TestExistingPathRedirects(t *testing.T) {
	seed(t)
	rr := httptest.NewRecorder()
	redirectHandler(rr, httptest.NewRequest(http.MethodGet, "/"+secretPath, nil))
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("want 303, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != destURL {
		t.Fatalf("want Location %q, got %q", destURL, loc)
	}
}

func TestMissingPathIs404(t *testing.T) {
	seed(t)
	rr := httptest.NewRecorder()
	redirectHandler(rr, httptest.NewRequest(http.MethodGet, "/definitely-not-a-real-link", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestNonGetIsRejected(t *testing.T) {
	rr := httptest.NewRecorder()
	redirectHandler(rr, httptest.NewRequest(http.MethodPost, "/"+secretPath, nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestOverlongPathIs404(t *testing.T) {
	rr := httptest.NewRecorder()
	long := "/" + string(bytes.Repeat([]byte("a"), maxPathLen+1))
	redirectHandler(rr, httptest.NewRequest(http.MethodGet, long, nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 for overlong path, got %d", rr.Code)
	}
}

// A hit is logged to stdout as JSON carrying the path, outcome and label.
func TestClickLoggedToStdout(t *testing.T) {
	seed(t)
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	redirectHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/"+secretPath, nil))

	out := buf.String()
	for _, want := range []string{"click", secretPath, `"outcome":"hit"`, sampleLbl} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Fatalf("log line missing %q; got: %s", want, out)
		}
	}
}

// insertEvent persists a full event row (empty fields become NULL).
func TestInsertEventPersists(t *testing.T) {
	seed(t)
	ev := &clickEvent{
		Time: time.Now().UTC(), Outcome: "hit", Path: secretPath, Label: sampleLbl,
		ClientIP: "203.0.113.7", Country: "DE", UserAgent: "test-agent", Referrer: "https://example.org/",
	}
	if err := insertEvent(ev); err != nil {
		t.Fatalf("insertEvent: %v", err)
	}
	var outcome, path, ip, country string
	err := db.QueryRow(
		`SELECT outcome, path, client_ip, country FROM click_events
		 WHERE path=$1 AND user_agent='test-agent' ORDER BY id DESC LIMIT 1`, secretPath).
		Scan(&outcome, &path, &ip, &country)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if outcome != "hit" || path != secretPath || ip != "203.0.113.7" || country != "DE" {
		t.Fatalf("row mismatch: outcome=%s path=%s ip=%s country=%s", outcome, path, ip, country)
	}
}
