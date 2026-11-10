// Package cli holds the quotelint command-line interface.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jkanie/quotelint/internal/extract"
	"github.com/jkanie/quotelint/internal/match"
	"github.com/jkanie/quotelint/internal/source"
)

// Version is the program version.
const Version = "0.1.0"

const usage = `Usage: quotelint [OPTIONS] DOC SOURCE...
       quotelint [OPTIONS] -s SOURCE [-s SOURCE]... DOC...

Check every quoted passage in each DOC against the SOURCE files.
A quoted passage sits between double or single quotation marks. A
quote inside a quote is checked as well as the quote around it.
A DOC is a Markdown or text file, or a folder of them (.md, .markdown,
.txt, .text). A SOURCE is a text file, an HTML file (.html, .htm) that
is turned into text first, or a folder of such files. Folders are read
recursively, skipping names that start with a dot. A SOURCE that starts
with http:// or https:// is a web page: it is fetched once into the
cache, and later runs read it from there, offline.
Files are read as UTF-8 text, or as UTF-16 when they start with a
UTF-16 byte order mark (as Notepad and Windows PowerShell 5.1 save them).

With no -s, the 1st path is the DOC and the rest are SOURCEs.

Options:
  -s, --source PATH  Add a SOURCE file or folder (repeatable). Every
                     other path is then a DOC.
  --min-words N      Check only quotes of N or more words (default 3).
  --refresh          Fetch every web page SOURCE again, then update the
                     cache.
  --cache-dir DIR    Keep fetched pages in DIR (default: quotelint in
                     the user cache folder).
  --fold-quotes      Treat curly and straight quotation marks as equal.
  --json             Print a JSON object instead of text.
  -h, --help         Print this help.
  -V, --version      Print the version.

Exit status: 0 when every quote is verbatim, 1 when any quote differs
or is not found, 2 on an error.
`

type config struct {
	docs     []string
	sources  []string
	minWords int
	fold     bool
	json     bool
	refresh  bool
	cacheDir string
}

var errHelp = errors.New("help")
var errVersion = errors.New("version")

func parse(args []string) (config, error) {
	c := config{minWords: 3}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			pos = append(pos, args[i+1:]...)
			i = len(args)
		case a == "-h" || a == "--help":
			return c, errHelp
		case a == "-V" || a == "--version":
			return c, errVersion
		case a == "--json":
			c.json = true
		case a == "--fold-quotes":
			c.fold = true
		case a == "--refresh":
			c.refresh = true
		case a == "--cache-dir" || strings.HasPrefix(a, "--cache-dir="):
			v, ok := strings.CutPrefix(a, "--cache-dir=")
			if !ok {
				if i+1 >= len(args) {
					return c, errors.New("--cache-dir needs a folder")
				}
				i++
				v = args[i]
			}
			if v == "" {
				return c, errors.New("--cache-dir needs a folder")
			}
			c.cacheDir = v
		case a == "-s" || a == "--source" || strings.HasPrefix(a, "--source="):
			v, ok := strings.CutPrefix(a, "--source=")
			if !ok {
				if i+1 >= len(args) {
					return c, fmt.Errorf("%s needs a path", a)
				}
				i++
				v = args[i]
			}
			c.sources = append(c.sources, v)
		case a == "--min-words" || strings.HasPrefix(a, "--min-words="):
			v, ok := strings.CutPrefix(a, "--min-words=")
			if !ok {
				if i+1 >= len(args) {
					return c, errors.New("--min-words needs a number")
				}
				i++
				v = args[i]
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return c, fmt.Errorf("--min-words needs a whole number of 1 or more, got %q", v)
			}
			c.minWords = n
		case strings.HasPrefix(a, "-") && a != "-":
			return c, fmt.Errorf("unknown option %s", a)
		default:
			pos = append(pos, a)
		}
	}
	if len(c.sources) > 0 {
		if len(pos) == 0 {
			return c, errors.New("need at least 1 DOC")
		}
		c.docs = pos
		return c, nil
	}
	if len(pos) < 2 {
		return c, errors.New("need a DOC and at least 1 SOURCE")
	}
	c.docs, c.sources = pos[:1], pos[1:]
	return c, nil
}

type jsonChange struct {
	Quote  string `json:"quote"`
	Source string `json:"source"`
}

type jsonQuote struct {
	Changes  []jsonChange `json:"changes"`
	Distance int          `json:"distance"`
	Doc      string       `json:"doc"`
	Line     int          `json:"line"`
	Passage  string       `json:"passage"`
	Quote    string       `json:"quote"`
	Source   string       `json:"source"`
	Status   string       `json:"status"`
}

type summary struct {
	Differs  int `json:"differs"`
	Docs     int `json:"docs"`
	NotFound int `json:"not_found"`
	Quotes   int `json:"quotes"`
	Verbatim int `json:"verbatim"`
}

// jsonPage records where a web page SOURCE came from in this run.
type jsonPage struct {
	FetchedAt string `json:"fetched_at"`
	FinalURL  string `json:"final_url"`
	FromCache bool   `json:"from_cache"`
	URL       string `json:"url"`
}

type report struct {
	Docs    []string    `json:"docs"`
	Match   bool        `json:"match"`
	Pages   []jsonPage  `json:"pages,omitempty"`
	Quotes  []jsonQuote `json:"quotes"`
	Sources []string    `json:"sources"`
	Summary summary     `json:"summary"`
}

// Run runs quotelint with args (without the program name) and returns
// the exit status.
func Run(args []string, stdout, stderr io.Writer) int {
	c, err := parse(args)
	switch {
	case errors.Is(err, errHelp):
		fmt.Fprint(stdout, usage)
		return 0
	case errors.Is(err, errVersion):
		fmt.Fprintln(stdout, "quotelint", Version)
		return 0
	case err != nil:
		fmt.Fprintf(stderr, "quotelint: %v\nTry quotelint --help.\n", err)
		return 2
	}
	r, err := check(c)
	if err != nil {
		fmt.Fprintf(stderr, "quotelint: %v\n", err)
		return 2
	}
	if c.json {
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			fmt.Fprintf(stderr, "quotelint: %v\n", err)
			return 2
		}
	} else {
		writeText(stdout, r)
	}
	if r.Match {
		return 0
	}
	return 1
}

func check(c config) (report, error) {
	docs, err := source.Expand(c.docs, source.DocExts)
	if err != nil {
		return report{}, err
	}
	var paths, urls []string
	for _, s := range c.sources {
		if source.IsURL(s) {
			urls = append(urls, s)
		} else {
			paths = append(paths, s)
		}
	}
	var srcs []string
	if len(paths) > 0 {
		srcs, err = source.Expand(paths, source.SourceExts)
		if err != nil {
			return report{}, err
		}
		srcs = source.Without(srcs, docs)
	}
	if len(srcs) == 0 && len(urls) == 0 {
		return report{}, errors.New("no SOURCE left once the DOCs are left out")
	}
	opt := match.Options{FoldQuotes: c.fold}
	var texts []match.Text
	for _, p := range srcs {
		s, err := source.Load(p)
		if err != nil {
			return report{}, err
		}
		texts = append(texts, match.Prepare(s.Name, s.Text, opt))
	}
	r := report{Docs: docs, Quotes: []jsonQuote{}}
	if len(urls) > 0 {
		dir := c.cacheDir
		if dir == "" {
			if dir, err = source.DefaultCacheDir(); err != nil {
				return report{}, err
			}
		}
		cache := source.Cache{Dir: dir, Refresh: c.refresh}
		seen := map[string]bool{}
		for _, u := range urls {
			if seen[u] {
				continue
			}
			seen[u] = true
			p, fetched, err := cache.Get(u)
			if err != nil {
				return report{}, err
			}
			s, err := source.LoadPage(p)
			if err != nil {
				return report{}, err
			}
			texts = append(texts, match.Prepare(s.Name, s.Text, opt))
			srcs = append(srcs, u)
			r.Pages = append(r.Pages, jsonPage{
				FetchedAt: p.FetchedAt.Format(time.RFC3339),
				FinalURL:  p.FinalURL,
				FromCache: !fetched,
				URL:       u,
			})
		}
	}
	r.Sources = srcs
	for _, d := range docs {
		b, err := os.ReadFile(d)
		if err != nil {
			return report{}, err
		}
		text, err := source.Text(b)
		if err != nil {
			return report{}, fmt.Errorf("%s: %v", d, err)
		}
		for _, q := range extract.Quotes(text, extract.Options{MinWords: c.minWords}) {
			res := match.Check(q.Text, texts, opt)
			jq := jsonQuote{
				Changes:  []jsonChange{},
				Distance: res.Distance,
				Doc:      d,
				Line:     q.Line,
				Passage:  res.Passage,
				Quote:    q.Text,
				Source:   res.Source,
				Status:   res.Status,
			}
			for _, ch := range res.Changes {
				jq.Changes = append(jq.Changes, jsonChange(ch))
			}
			r.Quotes = append(r.Quotes, jq)
			switch res.Status {
			case match.Verbatim:
				r.Summary.Verbatim++
			case match.Differs:
				r.Summary.Differs++
			default:
				r.Summary.NotFound++
			}
		}
	}
	r.Summary.Docs = len(docs)
	r.Summary.Quotes = len(r.Quotes)
	r.Match = r.Summary.Differs == 0 && r.Summary.NotFound == 0
	return r, nil
}

func writeText(w io.Writer, r report) {
	for _, q := range r.Quotes {
		head := fmt.Sprintf("%s:%d: ", q.Doc, q.Line)
		switch q.Status {
		case match.Verbatim:
			fmt.Fprintf(w, "%sverbatim in %s: %s\n", head, q.Source, preview(q.Quote))
		case match.Differs:
			edits := "edits"
			if q.Distance == 1 {
				edits = "edit"
			}
			fmt.Fprintf(w, "%sdiffers from %s (%d word %s)\n", head, q.Source, q.Distance, edits)
			fmt.Fprintf(w, "  quote:  %s\n", match.Normalize(q.Quote, match.Options{}))
			fmt.Fprintf(w, "  source: %s\n", q.Passage)
			for _, ch := range q.Changes {
				fmt.Fprintf(w, "  change: %s -> %s\n", show(ch.Quote), show(ch.Source))
			}
		default:
			fmt.Fprintf(w, "%snot found in any source: %s\n", head, preview(q.Quote))
		}
	}
	for _, p := range r.Pages {
		how := "fetched"
		if p.FromCache {
			how = "from the cache, fetched"
		}
		fmt.Fprintf(w, "page: %s (%s %s)\n", p.URL, how, p.FetchedAt)
	}
	s := r.Summary
	fmt.Fprintf(w, "summary: %s, %s, %d verbatim, %d differs, %d not found\n",
		count(s.Docs, "doc"), count(s.Quotes, "quote"), s.Verbatim, s.Differs, s.NotFound)
}

// count writes n and word, with an s on word unless n is 1.
func count(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// show quotes a changed run, or names a missing one.
func show(s string) string {
	if s == "" {
		return "(nothing)"
	}
	return strconv.Quote(s)
}

// preview gives the 1st 60 characters of a quote on 1 line.
func preview(q string) string {
	q = match.Normalize(q, match.Options{})
	r := []rune(q)
	if len(r) > 60 {
		return strconv.Quote(string(r[:60]) + "...")
	}
	return strconv.Quote(q)
}
