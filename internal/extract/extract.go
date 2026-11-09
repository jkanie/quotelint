// Package extract finds the quoted passages in a Markdown or text document.
package extract

import (
	"sort"
	"strings"
	"unicode"
)

// Quote is 1 quoted passage and the line that holds its opening mark.
type Quote struct {
	Line int
	Text string
}

// Options controls which passages count as quotes.
type Options struct {
	// MinWords drops passages with fewer words. Values below 1 count as 1.
	MinWords int
}

// Quotes returns every passage between a pair of quotation marks, in the
// order of their opening marks. Double marks are straight (" ") or curly
// (“ ”). Single marks are curly (‘ ’) or straight (' '). A single mark
// between 2 letters or digits is an apostrophe, not a mark (see
// opensSingle and closesSingle). A quote inside a quote is returned as
// well as the quote around it, which keeps the inner marks in its text.
// Marks inside fenced code blocks and inline code spans are ignored. A
// passage may run over several lines, but not past a blank line, so an
// unclosed mark never swallows the rest of the document.
func Quotes(doc string, opt Options) []Quote {
	if opt.MinWords < 1 {
		opt.MinWords = 1
	}
	var out []Quote
	lines := strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n")
	var para []string
	paraStart := 0
	fence := ""
	flush := func() {
		if len(para) > 0 {
			out = append(out, scan(para, paraStart, opt)...)
		}
		para = nil
	}
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			flush()
			fence = t[:3]
			continue
		}
		if t == "" {
			flush()
			continue
		}
		if len(para) == 0 {
			paraStart = i + 1
		}
		para = append(para, ln)
	}
	flush()
	return out
}

// open is 1 quotation mark that is still waiting for its closing mark.
type open struct {
	mark  rune // the opening mark
	line  int  // the line that holds the opening mark
	pos   int  // rune offset of the opening mark, for document order
	start int  // rune offset where the quote's text begins
}

// found is a closed quote, with the offset of its opening mark.
type found struct {
	pos int
	q   Quote
}

// scan finds the quotes in 1 paragraph whose 1st line is line number first.
func scan(lines []string, first int, opt Options) []Quote {
	text := []rune(strings.Join(lines, "\n"))
	var stack []open
	var got []found
	line := first
	inCode := false
	at := func(i int) rune {
		if i < 0 || i >= len(text) {
			return 0
		}
		return text[i]
	}
	// closeAt closes the open mark at stack index k with the closing mark
	// at offset i. Marks opened after it are left unclosed and dropped.
	closeAt := func(k, i int) {
		o := stack[k]
		q := string(text[o.start:i])
		if len(strings.Fields(q)) >= opt.MinWords {
			got = append(got, found{pos: o.pos, q: Quote{Line: o.line, Text: q}})
		}
		stack = stack[:k]
	}
	push := func(r rune, i int) {
		stack = append(stack, open{mark: r, line: line, pos: i, start: i + 1})
	}
	for i, r := range text {
		prev, next := at(i-1), at(i+1)
		switch {
		case r == '`':
			inCode = !inCode
		case inCode:
		case r == '"':
			if k := find(stack, '"'); k >= 0 {
				closeAt(k, i)
			} else {
				push(r, i)
			}
		case r == '“':
			push(r, i)
		case r == '”':
			if k := find(stack, '“'); k >= 0 {
				closeAt(k, i)
			}
		case r == '‘':
			push(r, i)
		case r == '’':
			if k := find(stack, '‘'); k >= 0 && closesSingle(prev, next) {
				closeAt(k, i)
			}
		case r == '\'':
			if k := find(stack, '\''); k >= 0 && closesSingle(prev, next) {
				closeAt(k, i)
			} else if opensSingle(prev, next) {
				push(r, i)
			}
		}
		if r == '\n' {
			line++
		}
	}
	sort.SliceStable(got, func(a, b int) bool { return got[a].pos < got[b].pos })
	out := make([]Quote, 0, len(got))
	for _, f := range got {
		out = append(out, f.q)
	}
	return out
}

// find returns the stack index of the innermost open mark m, or -1.
func find(stack []open, m rune) int {
	for k := len(stack) - 1; k >= 0; k-- {
		if stack[k].mark == m {
			return k
		}
	}
	return -1
}

// opensSingle reports whether a straight single mark between prev and
// next opens a quote. It must follow 1 of these: the start of the
// paragraph, white space, an opening bracket, an opening mark, an em
// dash, an en dash, "/". A letter or digit must follow it. The
// apostrophes in "don't" and "the 90's" therefore never open a quote.
func opensSingle(prev, next rune) bool {
	before := prev == 0 || unicode.IsSpace(prev) || strings.ContainsRune("([{\"“‘—–/", prev)
	return before && isWord(next)
}

// closesSingle reports whether a single mark (straight or curly) between
// prev and next closes a quote. It must follow a character that is not
// white space. No letter or digit may follow it. The apostrophes in
// "don't" and "it’s" therefore never close a quote.
func closesSingle(prev, next rune) bool {
	return prev != 0 && !unicode.IsSpace(prev) && !isWord(next)
}

func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
