package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests drive the real redirectHandler against the database opened by
// init() (point it at a local Postgres via POSTGRES_HOST/PORT env vars). They
// verify the security-relevant behaviour: hashed lookup, the 404-vs-303
// existence handling, the method guard, and that the secret path is never
// written to the logs.

const (
	secretPath = "wWD56zKM3ft5sr7p8xGjmQ" // a sample 128-bit path
	destURL    = "https://example.com/destination"
)

func seed(t *testing.T) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS redirects (
		path_hash CHAR(64) PRIMARY KEY, redirect_url TEXT NOT NULL, label VARCHAR(255))`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO redirects (path_hash, redirect_url) VALUES ($1, $2)
		ON CONFLICT (path_hash) DO UPDATE SET redirect_url = EXCLUDED.redirect_url`,
		hashPath(secretPath), destURL); err != nil {
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

// The secret path must never be logged: logs and backups would otherwise leak
// the very thing that makes the link unguessable.
func TestSecretPathNotLogged(t *testing.T) {
	seed(t)
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	for _, p := range []string{"/" + secretPath, "/missing-secret-xyz"} {
		redirectHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, p, nil))
	}
	if bytes.Contains(buf.Bytes(), []byte(secretPath)) || bytes.Contains(buf.Bytes(), []byte("missing-secret-xyz")) {
		t.Fatalf("secret path leaked into logs: %q", buf.String())
	}
}
