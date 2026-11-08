package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func norm(s string) string { return strings.Join(strings.Fields(s), " ") }

func TestInlineTagsLeaveNoGap(t *testing.T) {
	got := norm(HTMLText("<p>The LAG<em>n</em> function and <a href=x>a link</a>.</p>"))
	if got != "The LAGn function and a link." {
		t.Fatalf("got %q", got)
	}
}

func TestBlockTagsLeaveASpace(t *testing.T) {
	got := norm(HTMLText("<td>Default</td><td>12</td><br>end"))
	if got != "Default 12 end" {
		t.Fatalf("got %q", got)
	}
}

func TestScriptStyleAndCommentsAreDropped(t *testing.T) {
	in := "<head><style>p{x:1}</style><script>var a = '<p>';</script></head>" +
		"<!-- hidden --><p>shown</p><SCRIPT type=x>bad()</SCRIPT>"
	if got := norm(HTMLText(in)); got != "shown" {
		t.Fatalf("got %q", got)
	}
}

func TestEntitiesAreDecoded(t *testing.T) {
	got := HTMLText("a&amp;b &lt;c&gt; &#8722;1 x&nbsp;y")
	if got != "a&b <c> −1 x y" {
		t.Fatalf("got %q", got)
	}
}

func TestLoneLessThanStaysText(t *testing.T) {
	if got := norm(HTMLText("<p>x < 5 and y</p>")); got != "x < 5 and y" {
		t.Fatalf("got %q", got)
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
