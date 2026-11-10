package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// server serves body with contentType at every path and counts requests.
func server(t *testing.T, contentType string, body *atomic.Value) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.UserAgent() != UserAgent {
			http.Error(w, "no user agent", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(body.Load().(string)))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestPageIsFetchedOnceThenReadFromTheCache(t *testing.T) {
	var body atomic.Value
	body.Store("<p>The <em>first</em> version.</p>")
	srv, hits := server(t, "text/html; charset=utf-8", &body)
	dir := t.TempDir()
	at := time.Date(2026, 11, 9, 23, 40, 5, 0, time.UTC)
	c := Cache{Dir: dir, Now: func() time.Time { return at }}

	p, fetched, err := c.Get(srv.URL + "/page")
	if err != nil || !fetched || hits.Load() != 1 {
		t.Fatalf("1st get: fetched %v, hits %d, err %v", fetched, hits.Load(), err)
	}
	if p.ContentType != "text/html" || !p.FetchedAt.Equal(at) || p.FinalURL != srv.URL+"/page" {
		t.Fatalf("page %+v", p)
	}
	s, err := LoadPage(p)
	if err != nil || s.Text != "The first version." || s.Name != srv.URL+"/page" {
		t.Fatalf("source %+v, err %v", s, err)
	}

	// The page changes, and the cache still answers with the 1st version.
	body.Store("<p>The second version.</p>")
	c2 := Cache{Dir: dir}
	p, fetched, err = c2.Get(srv.URL + "/page")
	if err != nil || fetched || hits.Load() != 1 || !strings.Contains(p.Body, "first") || !p.FetchedAt.Equal(at) {
		t.Fatalf("2nd get: fetched %v, hits %d, body %q, err %v", fetched, hits.Load(), p.Body, err)
	}

	// Refresh fetches again and replaces the cache file.
	c3 := Cache{Dir: dir, Refresh: true}
	p, fetched, err = c3.Get(srv.URL + "/page")
	if err != nil || !fetched || hits.Load() != 2 || !strings.Contains(p.Body, "second") {
		t.Fatalf("refresh: fetched %v, hits %d, body %q, err %v", fetched, hits.Load(), p.Body, err)
	}
	p, _, _ = c2.Get(srv.URL + "/page")
	if !strings.Contains(p.Body, "second") {
		t.Fatalf("the refresh did not reach the cache: %q", p.Body)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*"))
	if len(files) != 1 || !strings.HasSuffix(files[0], ".json") {
		t.Fatalf("cache folder holds %v", files)
	}
}

func TestPlainTextPageIsUsedAsItIs(t *testing.T) {
	var body atomic.Value
	body.Store("Line <1>\n  kept as text.")
	srv, _ := server(t, "text/plain", &body)
	c := Cache{Dir: t.TempDir()}
	p, _, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := LoadPage(p)
	if s.Text != "Line <1>\n  kept as text." {
		t.Fatalf("text %q", s.Text)
	}
}

func TestFetchRefusals(t *testing.T) {
	big := strings.Repeat("a", 2048)
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
		want                    string
	}{
		{"status", "text/html", "gone", http.StatusNotFound, "the server answered 404 Not Found"},
		{"pdf", "application/pdf", "%PDF", http.StatusOK, `content type "application/pdf" is not text/html or text/plain`},
		{"charset", "text/html; charset=windows-1252", "x", http.StatusOK, `charset "windows-1252", only UTF-8 pages are read`},
		{"bytes", "text/plain", "caf\xe9", http.StatusOK, "not UTF-8 text"},
		{"size", "text/plain", big, http.StatusOK, "page is larger than"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := MaxPageBytes
			MaxPageBytes = 1024
			defer func() { MaxPageBytes = old }()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			dir := t.TempDir()
			_, _, err := (&Cache{Dir: dir}).Get(srv.URL)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err %v, want %q", err, tc.want)
			}
			if files, _ := os.ReadDir(dir); len(files) != 0 {
				t.Fatalf("a refused page was cached: %v", files)
			}
		})
	}
}

func TestRedirectIsFollowedAndRecorded(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("moved here"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	p, _, err := (&Cache{Dir: t.TempDir()}).Get(srv.URL + "/old")
	if err != nil || p.URL != srv.URL+"/old" || p.FinalURL != srv.URL+"/new" || p.Body != "moved here" {
		t.Fatalf("page %+v, err %v", p, err)
	}
}

func TestBrokenCacheFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	c := Cache{Dir: dir}
	u := "https://example.com/a"
	if err := os.WriteFile(c.file(u), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Get(u); err == nil || !strings.Contains(err.Error(), "broken cache file") {
		t.Fatalf("err %v", err)
	}
}

func TestIsURL(t *testing.T) {
	for s, want := range map[string]bool{
		"https://example.com": true, "HTTP://example.com/x": true,
		"ftp://example.com": false, "notes/https.html": false, "./http://x": false,
	} {
		if IsURL(s) != want {
			t.Errorf("IsURL(%q) = %v", s, !want)
		}
	}
}
