// Package extract finds the quoted passages in a Markdown or text document.
package extract

import "strings"

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

// Quotes returns every passage between a pair of double quotation marks,
// straight (" ") or curly (“ ”), in document order. Marks inside fenced
// code blocks and inline code spans are ignored. A passage may run over
// several lines, but not past a blank line, so an unclosed mark never
// swallows the rest of the document.
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

// scan finds the quotes in 1 paragraph whose 1st line is line number first.
func scan(lines []string, first int, opt Options) []Quote {
	var out []Quote
	var buf strings.Builder
	line, start := first, 0
	inCode := false
	var open rune
	for _, r := range strings.Join(lines, "\n") {
		switch {
		case r == '`':
			inCode = !inCode
			if open != 0 {
				buf.WriteRune(r)
			}
		case !inCode && open == 0 && (r == '"' || r == '“'):
			open, start = r, line
			buf.Reset()
		case !inCode && ((open == '"' && r == '"') || (open == '“' && r == '”')):
			q := buf.String()
			if len(strings.Fields(q)) >= opt.MinWords {
				out = append(out, Quote{Line: start, Text: q})
			}
			open = 0
		default:
			if open != 0 {
				buf.WriteRune(r)
			}
		}
		if r == '\n' {
			line++
		}
	}
	return out
}
