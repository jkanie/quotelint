package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const page = `<html><head><style>p{}</style></head><body>
<p>An occurrence of the LAG<em>n</em> function that is executed
conditionally stores and return values only from the observations for
which the condition is satisfied.</p></body></html>`

func setup(t *testing.T, doc string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	d := filepath.Join(dir, "notes.md")
	s := filepath.Join(dir, "lag.html")
	if err := os.WriteFile(d, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	return d, s
}

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestAllVerbatimExitsZero(t *testing.T) {
	d, s := setup(t, "SAS: \"stores and return values only from the observations\".\n")
	code, out, _ := run(d, s)
	if code != 0 || !strings.Contains(out, "notes.md:1: verbatim in "+s) ||
		!strings.Contains(out, "summary: 1 doc, 1 quote, 1 verbatim, 0 differs, 0 not found") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
}

func TestDiffersExitsOne(t *testing.T) {
	d, s := setup(t, "Old: \"executed conditionally will store and return values\".\n")
	code, out, _ := run(d, s)
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	for _, w := range []string{
		"notes.md:1: differs from " + s + " (2 word edits)",
		"  source: executed conditionally stores and return values",
		`  change: "will store" -> "stores"`,
	} {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in:\n%s", w, out)
		}
	}
}

func TestNotFoundExitsOne(t *testing.T) {
	d, s := setup(t, "\"the moon is made of cheese\"\n")
	code, out, _ := run(d, s)
	if code != 1 || !strings.Contains(out, "not found in any source") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
}

func TestJSONIsValidAndSorted(t *testing.T) {
	d, s := setup(t, "\"executed conditionally will store and return values\"\n")
	code, out, _ := run("--json", d, s)
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, out)
	}
	keys := []string{`"docs"`, `"match"`, `"quotes"`, `"sources"`, `"summary"`}
	last := -1
	for _, k := range keys {
		i := strings.Index(out, k)
		if i <= last {
			t.Fatalf("key %s out of order in:\n%s", k, out)
		}
		last = i
	}
	if v["match"] != false {
		t.Fatalf("match = %v", v["match"])
	}
}

func TestJSONSameBytesTwice(t *testing.T) {
	d, s := setup(t, "\"stores and return values\" and \"the moon is cheese\"\n")
	_, a, _ := run("--json", d, s)
	_, b, _ := run("--json", d, s)
	if a != b {
		t.Fatal("2 runs printed different JSON")
	}
}

func TestMinWords(t *testing.T) {
	d, s := setup(t, "\"made up words\" and \"the moon is made of cheese\"\n")
	_, out, _ := run("--min-words", "4", d, s)
	if !strings.Contains(out, "summary: 1 doc, 1 quote,") {
		t.Fatalf("out:\n%s", out)
	}
	_, out, _ = run("--min-words=1", d, s)
	if !strings.Contains(out, "summary: 1 doc, 2 quotes,") {
		t.Fatalf("out:\n%s", out)
	}
}

func TestFoldQuotes(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d.md")
	s := filepath.Join(dir, "s.txt")
	os.WriteFile(d, []byte("“SAS writes \"x\" for it's value”\n"), 0o644)
	os.WriteFile(s, []byte("SAS writes “x” for it’s value"), 0o644)
	if code, _, _ := run(d, s); code != 1 {
		t.Fatalf("strict code %d", code)
	}
	if code, out, _ := run("--fold-quotes", d, s); code != 0 {
		t.Fatalf("folded code %d:\n%s", code, out)
	}
}

func TestErrorsExitTwo(t *testing.T) {
	d, s := setup(t, "x\n")
	cases := [][]string{
		{},
		{d},
		{"--bogus", d, s},
		{"--min-words", "0", d, s},
		{"--min-words"},
		{filepath.Join(filepath.Dir(d), "missing.md"), s},
		{d, filepath.Join(filepath.Dir(d), "missing.html")},
	}
	for _, c := range cases {
		code, out, errOut := run(c...)
		if code != 2 || out != "" || !strings.HasPrefix(errOut, "quotelint: ") {
			t.Fatalf("%q: code %d, out %q, err %q", c, code, out, errOut)
		}
	}
}

func TestHelpAndVersion(t *testing.T) {
	code, out, _ := run("--help")
	if code != 0 || !strings.HasPrefix(out, "Usage: quotelint [OPTIONS] DOC SOURCE...\n       quotelint [OPTIONS] -s SOURCE") {
		t.Fatalf("help: %d %q", code, out)
	}
	code, out, _ = run("-V")
	if code != 0 || out != "quotelint "+Version+"\n" {
		t.Fatalf("version: %d %q", code, out)
	}
}

func TestDoubleDashEndsOptions(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "--json")
	s := filepath.Join(dir, "s.txt")
	os.WriteFile(d, []byte("\"a b c\"\n"), 0o644)
	os.WriteFile(s, []byte("a b c"), 0o644)
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)
	code, out, _ := run("--", "--json", "s.txt")
	if code != 0 || !strings.HasPrefix(out, "--json:1: verbatim") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestNoQuotesMatches(t *testing.T) {
	d, s := setup(t, "No quotes here.\n")
	code, out, _ := run(d, s)
	if code != 0 || out != "summary: 1 doc, 0 quotes, 0 verbatim, 0 differs, 0 not found\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceFolderIsWalked(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "notes.md")
	write(t, d, "\"alpha beta gamma\" and \"delta epsilon zeta\"\n")
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "a.txt"), "x alpha beta gamma y")
	write(t, filepath.Join(src, "sub", "b.html"), "<p>delta <em>epsilon</em> zeta</p>")
	write(t, filepath.Join(src, "skip.pdf"), "delta epsilon zeta")
	write(t, filepath.Join(src, ".git", "c.txt"), "never read")
	code, out, errOut := run(d, src)
	if code != 0 {
		t.Fatalf("code %d, out:\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out, "verbatim in "+filepath.Join(src, "a.txt")) ||
		!strings.Contains(out, "verbatim in "+filepath.Join(src, "sub", "b.html")) {
		t.Fatalf("out:\n%s", out)
	}
	_, js, _ := run("--json", d, src)
	if strings.Contains(js, "skip.pdf") || strings.Contains(js, ".git") {
		t.Fatalf("read a file it should skip:\n%s", js)
	}
}

func TestSeveralDocsWithSourceFlag(t *testing.T) {
	dir := t.TempDir()
	s := filepath.Join(dir, "s.txt")
	write(t, s, "the quick brown fox jumps over the lazy dog")
	d1 := filepath.Join(dir, "docs", "1.md")
	d2 := filepath.Join(dir, "docs", "deep", "2.txt")
	write(t, d1, "\"quick brown fox\"\n")
	write(t, d2, "\n\"the lazy cat\"\n")
	code, out, _ := run("-s", s, filepath.Join(dir, "docs"))
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	for _, w := range []string{
		d1 + ":1: verbatim in " + s,
		d2 + ":2: differs from " + s,
		"summary: 2 docs, 2 quotes, 1 verbatim, 1 differs, 0 not found",
	} {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in:\n%s", w, out)
		}
	}
	code, out, _ = run("--source="+s, d1, d2)
	if code != 1 || !strings.Contains(out, "summary: 2 docs") {
		t.Fatalf("2 files: code %d\n%s", code, out)
	}
}

func TestDocIsNotItsOwnSource(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "notes.md")
	write(t, d, "\"made up words here\"\n")
	write(t, filepath.Join(dir, "real.txt"), "nothing alike at all")
	code, out, _ := run("-s", dir, d)
	if code != 1 || !strings.Contains(out, "not found in any source") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
	code, _, errOut := run("-s", d, d)
	if code != 2 || !strings.Contains(errOut, "no SOURCE left") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestEmptyFolderIsAnError(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "notes.md")
	write(t, d, "\"a b c\"\n")
	empty := filepath.Join(dir, "empty")
	os.Mkdir(empty, 0o755)
	code, _, errOut := run(d, empty)
	if code != 2 || !strings.Contains(errOut, "no file ending in") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	code, _, errOut = run("-s", d)
	if code != 2 || !strings.Contains(errOut, "need at least 1 DOC") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
