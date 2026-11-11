// Package source loads the texts that quotes are checked against.
package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Source is 1 loaded source file.
type Source struct {
	Name string
	Text string
}

// Load reads a source file. A file that starts with a UTF-16 byte order
// mark is read as UTF-16 (see UTF16). Any other file is read as UTF-8 text.
// A file ending in .html or .htm is then turned into text with HTMLText.
func Load(path string) (Source, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}
	s, isUTF16, err := UTF16(b)
	if err != nil {
		return Source{}, fmt.Errorf("%s: %v", path, err)
	}
	if !isUTF16 {
		if !utf8.Valid(b) {
			return Source{}, fmt.Errorf("%s: not UTF-8 text", path)
		}
		s = string(b)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		s, err = HTMLText(s)
		if err != nil {
			return Source{}, fmt.Errorf("%s: %v", path, err)
		}
	}
	return Source{Name: path, Text: s}, nil
}

// UTF16 decodes b when it starts with a UTF-16 byte-order mark: FF FE for
// little-endian (what Notepad saves as "UTF-16 LE" and what Windows
// PowerShell 5.1 writes with >), FE FF for big-endian. The mark is not
// part of the text. ok is false (with no error) when b has no such mark.
// An odd number of bytes or a surrogate without its pair is an error.
func UTF16(b []byte) (text string, ok bool, err error) {
	var big bool
	switch {
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE:
	case len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF:
		big = true
	default:
		return "", false, nil
	}
	b = b[2:]
	if len(b)%2 != 0 {
		return "", true, errors.New("not UTF-16 text (odd number of bytes)")
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		if big {
			units[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			units[i] = uint16(b[2*i+1])<<8 | uint16(b[2*i])
		}
	}
	for i := 0; i < len(units); i++ {
		u := rune(units[i])
		switch {
		case utf16.IsSurrogate(u) && u < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] <= 0xDFFF:
			i++
		case utf16.IsSurrogate(u):
			return "", true, fmt.Errorf("not UTF-16 text (unpaired surrogate at byte %d)", 2+2*i)
		}
	}
	return string(utf16.Decode(units)), true, nil
}

// Text turns the bytes of a document into text: UTF-16 when the bytes
// start with a UTF-16 byte-order mark, else the bytes as they are.
func Text(b []byte) (string, error) {
	s, ok, err := UTF16(b)
	if err != nil || ok {
		return s, err
	}
	return string(b), nil
}

// inline lists the elements whose tags are dropped with no space, so that
// "LAG<em>n</em>" reads as "LAGn", the way a browser shows it.
var inline = map[string]bool{
	"a": true, "abbr": true, "b": true, "bdi": true, "bdo": true,
	"cite": true, "code": true, "data": true, "del": true, "dfn": true,
	"em": true, "font": true, "i": true, "ins": true, "kbd": true,
	"mark": true, "q": true, "s": true, "samp": true, "small": true,
	"span": true, "strong": true, "sub": true, "sup": true, "time": true,
	"tt": true, "u": true, "var": true, "wbr": true,
}

// skipped lists the elements whose content is never page text.
var skipped = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true,
}

// HTMLText turns an HTML page into its visible text. The page is read by
// the HTML5 parser of golang.org/x/net/html, so unclosed tags, a "<"
// inside an attribute value, a "<" that does not start a tag are read the
// way a browser reads them. Comments, the content of script, style,
// noscript, template elements, every element with a hidden attribute or
// with aria-hidden="true" are dropped. An inline tag (em, span, a ...)
// leaves no gap, any other tag or comment leaves a space. Character
// references such as &amp; are decoded. Outside pre, each run of HTML
// white space becomes 1 space. Inside pre, the text keeps its own white
// space and line breaks, and a br tag gives a line break. The parser
// returns an error for a page nested deeper than 512 elements.
func HTMLText(s string) (string, error) {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return "", err
	}
	var w textWriter
	w.walk(doc, false)
	return w.b.String(), nil
}

// textWriter builds the text of a page. A gap is written only between 2
// pieces of text, so the text never starts or ends with a gap and never
// holds 2 gaps in a row.
type textWriter struct {
	b   strings.Builder
	gap bool
}

func (w *textWriter) write(s string) {
	if s == "" {
		return
	}
	if w.gap && w.b.Len() > 0 {
		w.b.WriteByte(' ')
	}
	w.gap = false
	w.b.WriteString(s)
}

// walk writes the text of n and its children. pre is true inside a pre
// element.
func (w *textWriter) walk(n *html.Node, pre bool) {
	switch n.Type {
	case html.TextNode:
		if pre {
			w.write(n.Data)
			return
		}
		if strings.IndexFunc(n.Data, isHTMLSpace) == 0 {
			w.gap = true
		}
		for i, f := range strings.FieldsFunc(n.Data, isHTMLSpace) {
			if i > 0 {
				w.gap = true
			}
			w.write(f)
		}
		if n.Data != "" && isHTMLSpace(rune(n.Data[len(n.Data)-1])) {
			w.gap = true
		}
		return
	case html.CommentNode, html.DoctypeNode:
		w.gap = true
		return
	case html.ElementNode:
		if n.Data == "br" && pre && !hidden(n) {
			w.gap = false
			w.write("\n")
			return
		}
		if !inline[n.Data] {
			w.gap = true
		}
		if skipped[n.Data] || hidden(n) {
			return
		}
		pre = pre || n.Data == "pre"
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.walk(c, pre)
	}
	if n.Type == html.ElementNode && !inline[n.Data] {
		w.gap = true
	}
}

// hidden reports whether an element has a hidden attribute (with any
// value) or aria-hidden="true" (in any case), which keep it off the page
// or away from screen readers.
func hidden(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Namespace != "" {
			continue
		}
		if a.Key == "hidden" {
			return true
		}
		if a.Key == "aria-hidden" && strings.EqualFold(strings.TrimSpace(a.Val), "true") {
			return true
		}
	}
	return false
}

// isHTMLSpace reports whether r is HTML white space: space, tab, line
// feed, form feed, carriage return. A no-break space is not.
func isHTMLSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\f' || r == '\r'
}
