package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func norm(s string) string { return strings.Join(strings.Fields(s), " ") }

// text runs HTMLText and stops the test on an error.
func text(t *testing.T, s string) string {
	t.Helper()
	got, err := HTMLText(s)
	if err != nil {
		t.Fatalf("HTMLText(%q): %v", s, err)
	}
	return got
}

func TestInlineTagsLeaveNoGap(t *testing.T) {
	got := norm(text(t, "<p>The LAG<em>n</em> function and <a href=x>a link</a>.</p>"))
	if got != "The LAGn function and a link." {
		t.Fatalf("got %q", got)
	}
}

func TestBlockTagsLeaveASpace(t *testing.T) {
	got := norm(text(t, "<table><tr><td>Default</td><td>12</td></tr></table><br>end"))
	if got != "Default 12 end" {
		t.Fatalf("got %q", got)
	}
	// Outside a table, the parser drops td tags, as a browser does.
	got = norm(text(t, "<td>Default</td><td>12</td><br>end"))
	if got != "Default12 end" {
		t.Fatalf("td outside a table: got %q", got)
	}
}

func TestScriptStyleAndCommentsAreDropped(t *testing.T) {
	in := "<head><style>p{x:1}</style><script>var a = '<p>';</script></head>" +
		"<!-- hidden --><p>shown</p><SCRIPT type=x>bad()</SCRIPT>"
	if got := norm(text(t, in)); got != "shown" {
		t.Fatalf("got %q", got)
	}
}

func TestEntitiesAreDecoded(t *testing.T) {
	got := text(t, "a&amp;b &lt;c&gt; &#8722;1 x&nbsp;y")
	if got != "a&b <c> −1 x\u00a0y" {
		t.Fatalf("got %q", got)
	}
}

func TestLoneLessThanStaysText(t *testing.T) {
	if got := norm(text(t, "<p>x < 5 and y</p>")); got != "x < 5 and y" {
		t.Fatalf("got %q", got)
	}
}

func TestUnclosedTags(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<p>one<p>two<li>three", "one two three"},
		{"<p>The <b>bold <i>both</b> rest</i> end.", "The bold both rest end."},
		{"<div>open <em>to the end", "open to the end"},
		{"<p>before</p><style>p{}", "before"},
		{"<p>before</p><!-- never closed <p>after</p>", "before"},
		// An attribute quote that never closes runs to the end of the page.
		{`<p>a <span title="x>b</span> c</p><p>d</p>`, "a"},
	}
	for _, c := range cases {
		if got := norm(text(t, c.in)); got != c.want {
			t.Errorf("%q: got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLessThanInsideAttributeValues(t *testing.T) {
	in := `<p title="a > b">The <a href="x" data-x='<p>'>pump</a> is <span title="1<2>3">off</span>.</p>`
	if got := norm(text(t, in)); got != "The pump is off." {
		t.Fatalf("got %q", got)
	}
}

func TestScriptStyleNoscriptTemplateContentIsDropped(t *testing.T) {
	in := "<p>a</p><script>if (x < 1 && y > 2) { document.write('<p>no</p>') }</script>" +
		"<style>p::after { content: \"no\" }</style><noscript><p>no</p></noscript>" +
		"<template><p>no</p></template><p>b</p>"
	if got := norm(text(t, in)); got != "a b" {
		t.Fatalf("got %q", got)
	}
}

func TestPreKeepsLineBreaks(t *testing.T) {
	in := "<p>Run  it\n   now.</p><pre>\nline 1\n  line 2<br>line 3</pre><p>After\nall.</p>"
	want := "Run it now. line 1\n  line 2\nline 3 After all."
	if got := text(t, in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	// An inline tag inside pre keeps the line breaks around it.
	in = "<pre>a\n<em>b</em>\nc</pre>"
	if got := text(t, in); got != "a\nb\nc" {
		t.Fatalf("inline in pre: got %q", got)
	}
}

func TestHiddenElementsAreDropped(t *testing.T) {
	in := "<p>Cool the motor<span hidden> secret</span> for 10 minutes.</p>" +
		"<div hidden=\"until-found\"><p>no</p></div>" +
		"<p aria-hidden=\"true\">no</p><p aria-hidden=\"TRUE\">no</p>" +
		"<p aria-hidden=\"false\">kept</p><p HIDDEN>no</p>"
	if got := norm(text(t, in)); got != "Cool the motor for 10 minutes. kept" {
		t.Fatalf("got %q", got)
	}
}

func TestTooDeepIsAnError(t *testing.T) {
	in := strings.Repeat("<div>", 600) + "x"
	if _, err := HTMLText(in); err == nil || !strings.Contains(err.Error(), "512") {
		t.Fatalf("got %v", err)
	}
	p := filepath.Join(t.TempDir(), "deep.html")
	os.WriteFile(p, []byte(in), 0o644)
	if _, err := Load(p); err == nil || !strings.HasPrefix(err.Error(), p+": ") {
		t.Fatalf("Load: got %v", err)
	}
}

func TestLoadPicksHTMLByExtension(t *testing.T) {
	dir := t.TempDir()
	h := filepath.Join(dir, "page.HTM")
	p := filepath.Join(dir, "page.txt")
	os.WriteFile(h, []byte("<b>bold</b> text"), 0o644)
	os.WriteFile(p, []byte("<b>bold</b> text"), 0o644)
	sh, err := Load(h)
	if err != nil || norm(sh.Text) != "bold text" {
		t.Fatalf("html: %q %v", sh.Text, err)
	}
	sp, err := Load(p)
	if err != nil || sp.Text != "<b>bold</b> text" {
		t.Fatalf("text: %q %v", sp.Text, err)
	}
}

func TestLoadRejectsNonUTF8(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.txt")
	os.WriteFile(p, []byte{0xff, 0xfe, 'a'}, 0o644)
	if _, err := Load(p); err == nil || !strings.Contains(err.Error(), "not UTF-8") {
		t.Fatalf("got %v", err)
	}
}

func TestExpandSortsDedupesAndSkipsDotNames(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"b.md", "a.TXT", "sub/c.html", ".hidden/d.md", ".e.md", "f.pdf"} {
		full := filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(full), 0o755)
		os.WriteFile(full, []byte("x"), 0o644)
	}
	got, err := Expand([]string{dir, filepath.Join(dir, "b.md")}, SourceExts)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "a.TXT"), filepath.Join(dir, "b.md"), filepath.Join(dir, "sub", "c.html")}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := Expand([]string{filepath.Join(dir, "nope")}, SourceExts); err == nil {
		t.Fatal("a missing path should be an error")
	}
	if got := Without(want, []string{filepath.Join(dir, ".", "b.md")}); len(got) != 2 {
		t.Fatalf("Without: %q", got)
	}
}
