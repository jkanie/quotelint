package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxPageBytes is the largest page body that is read from the web.
var MaxPageBytes int64 = 20 << 20

// FetchTimeout limits 1 fetch, from the request to the last byte.
var FetchTimeout = 30 * time.Second

// UserAgent names quotelint and where it comes from on every request.
var UserAgent = "quotelint (+https://github.com/jkanie/quotelint)"

// IsURL reports whether a SOURCE argument names a web page instead of a
// path: it starts with http:// or https:// (any case).
func IsURL(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}

// Page is 1 web page as it is kept in the cache.
type Page struct {
	URL         string    `json:"url"`
	FinalURL    string    `json:"final_url"`
	FetchedAt   time.Time `json:"fetched_at"`
	ContentType string    `json:"content_type"`
	Body        string    `json:"body"`
}

// Cache keeps fetched pages in a folder, 1 JSON file per URL, named by the
// SHA-256 of the URL. A page is fetched once. Later runs read the file,
// so a check can be run again offline, and gives the same result after
// the page changes, until Refresh asks for a new fetch.
type Cache struct {
	Dir     string
	Refresh bool
	Client  *http.Client
	// Now gives the fetch time. Tests set it.
	Now func() time.Time
}

// DefaultCacheDir is quotelint inside the user's cache folder, such as
// ~/.cache/quotelint on Linux.
func DefaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no cache folder: %v (name 1 with --cache-dir)", err)
	}
	return filepath.Join(base, "quotelint"), nil
}

func (c *Cache) file(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(c.Dir, hex.EncodeToString(sum[:])+".json")
}

// Get returns the page for rawURL from the cache, or fetches it when the
// cache has none or Refresh is set. fetched is true when this call went
// to the web.
func (c *Cache) Get(rawURL string) (p Page, fetched bool, err error) {
	if !c.Refresh {
		b, err := os.ReadFile(c.file(rawURL))
		switch {
		case err == nil:
			if err := json.Unmarshal(b, &p); err != nil || p.URL != rawURL {
				return Page{}, false, fmt.Errorf("%s: broken cache file %s (delete it, or use --refresh)", rawURL, c.file(rawURL))
			}
			return p, false, nil
		case !errors.Is(err, os.ErrNotExist):
			return Page{}, false, err
		}
	}
	p, err = c.fetch(rawURL)
	if err != nil {
		return Page{}, false, err
	}
	if err := c.store(p); err != nil {
		return Page{}, false, err
	}
	return p, true, nil
}

func (c *Cache) fetch(rawURL string) (Page, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Page{}, fmt.Errorf("%s: not an http or https URL", rawURL)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: FetchTimeout}
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return Page{}, fmt.Errorf("%s: %v", rawURL, err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html, text/plain;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return Page{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("%s: the server answered %s", rawURL, resp.Status)
	}
	ct := resp.Header.Get("Content-Type")
	media, params, err := mime.ParseMediaType(ct)
	if err != nil || (media != "text/html" && media != "text/plain") {
		return Page{}, fmt.Errorf("%s: content type %q is not text/html or text/plain", rawURL, ct)
	}
	if cs := strings.ToLower(params["charset"]); cs != "" && cs != "utf-8" && cs != "utf8" && cs != "us-ascii" {
		return Page{}, fmt.Errorf("%s: charset %q, only UTF-8 pages are read", rawURL, params["charset"])
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxPageBytes+1))
	if err != nil {
		return Page{}, fmt.Errorf("%s: %v", rawURL, err)
	}
	if int64(len(b)) > MaxPageBytes {
		return Page{}, fmt.Errorf("%s: page is larger than %d MiB", rawURL, MaxPageBytes>>20)
	}
	if !utf8.Valid(b) {
		return Page{}, fmt.Errorf("%s: not UTF-8 text", rawURL)
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	return Page{
		URL:         rawURL,
		FinalURL:    resp.Request.URL.String(),
		FetchedAt:   now().UTC().Truncate(time.Second),
		ContentType: media,
		Body:        string(b),
	}, nil
}

// store writes the page to a temporary file that then replaces the cache
// file, so a run that stops half-way never leaves half a page.
func (c *Cache) store(p Page) error {
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.Dir, ".page-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), c.file(p.URL)); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// LoadPage turns a cached page into a Source named by its URL. An HTML page
// goes through HTMLText, a plain-text page is used as it is.
func LoadPage(p Page) (Source, error) {
	if p.ContentType != "text/html" {
		return Source{Name: p.URL, Text: p.Body}, nil
	}
	s, err := HTMLText(p.Body)
	if err != nil {
		return Source{}, fmt.Errorf("%s: %v", p.URL, err)
	}
	return Source{Name: p.URL, Text: s}, nil
}
