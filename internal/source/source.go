// Package source loads the texts that quotes are checked against.
package source

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Source is 1 loaded source file.
type Source struct {
	Name string
	Text string
}

// Load reads a source file. A file ending in .html or .htm is turned into
// text with HTMLText. Any other file is read as UTF-8 text.
func Load(path string) (Source, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}
	if !utf8.Valid(b) {
		return Source{}, fmt.Errorf("%s: not UTF-8 text", path)
	}
	s := string(b)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		s = HTMLText(s)
	}
	return Source{Name: path, Text: s}, nil
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

// HTMLText turns an HTML page into its visible text. Comments and the
// content of script, style, noscript and template elements are dropped.
// An inline tag (em, span, a ...) leaves no gap, any other tag leaves a
// space, and character references such as &amp; are decoded. A "<" that
// does not start a tag stays as text.
func HTMLText(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '<' {
			j := strings.IndexByte(s[i:], '<')
			if j < 0 {
				j = len(s) - i
			}
			b.WriteString(html.UnescapeString(s[i : i+j]))
			i += j
			continue
		}
		if strings.HasPrefix(s[i:], "<!--") {
			j := strings.Index(s[i+4:], "-->")
			if j < 0 {
				break
			}
			i += 4 + j + 3
			b.WriteByte(' ')
			continue
		}
		if !startsTag(s[i+1:]) {
			b.WriteByte('<')
			i++
			continue
		}
		j := strings.IndexByte(s[i:], '>')
		if j < 0 {
			break
		}
		tag := s[i+1 : i+j]
		i += j + 1
		name := tagName(tag)
		if skipped[name] && !strings.HasPrefix(tag, "/") && !strings.HasSuffix(tag, "/") {
			k := strings.Index(strings.ToLower(s[i:]), "</"+name)
			if k < 0 {
				break
			}
			i += k
			m := strings.IndexByte(s[i:], '>')
			if m < 0 {
				break
			}
			i += m + 1
			b.WriteByte(' ')
			continue
		}
		if !inline[name] {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// startsTag reports whether the text after a "<" opens a tag, a closing
// tag or a declaration such as <!DOCTYPE html>.
func startsTag(rest string) bool {
	if rest == "" {
		return false
	}
	c := rest[0]
	if c == '/' || c == '!' || c == '?' {
		return true
	}
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

// tagName returns the lower-case element name of a tag's inner text.
func tagName(tag string) string {
	tag = strings.TrimPrefix(tag, "/")
	end := strings.IndexAny(tag, " \t\r\n/>")
	if end >= 0 {
		tag = tag[:end]
	}
	return strings.ToLower(tag)
}
