// Package match checks a quote against source texts, verbatim first and
// then by the nearest passage, measured in word edits.
package match

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Status values of a Result.
const (
	Verbatim = "verbatim"
	Differs  = "differs"
	NotFound = "not found"
)

// Options controls how quote and source text are compared.
type Options struct {
	// FoldQuotes treats curly quotation marks and apostrophes as their
	// straight forms. Off by default, because a quote must keep the
	// source's own characters.
	FoldQuotes bool
}

// Change is 1 run of differing words between a quote and its nearest
// passage. An empty Quote means the source has words that the quote
// lacks. An empty Source means the quote has words that the source lacks.
type Change struct {
	Quote  string
	Source string
}

// Result is the outcome for 1 quote.
type Result struct {
	Status   string
	Source   string // name of the source that holds the passage
	Passage  string // the matching or nearest passage, normalised
	Distance int    // word edits between quote and passage
	Changes  []Change
}

// Text is a source prepared for repeated checks.
type Text struct {
	Name  string
	norm  string
	words []string
}

var folder = strings.NewReplacer(
	"‘", "'", "’", "'", "‚", "'", "‛", "'",
	"“", "\"", "”", "\"", "„", "\"", "‟", "\"",
)

// Normalize collapses every run of white space (no-break spaces
// included) to 1 space and trims both ends. With FoldQuotes, it also maps
// curly quotation marks to straight ones. No other character changes.
func Normalize(s string, opt Options) string {
	if opt.FoldQuotes {
		s = folder.Replace(s)
	}
	return strings.Join(strings.Fields(s), " ")
}

// Prepare normalises a source text once for many checks.
func Prepare(name, text string, opt Options) Text {
	n := Normalize(text, opt)
	return Text{Name: name, norm: n, words: strings.Fields(n)}
}

// Check looks for quote in the sources. A quote is verbatim when its
// normalised text occurs in a source with no letter or digit glued to
// either end. Otherwise the nearest passage in any source is the one with
// the fewest word edits (ties go to the earlier source and the earlier
// place). A passage that needs more edits than half the quote's words
// counts as not found.
func Check(quote string, sources []Text, opt Options) Result {
	q := Normalize(quote, opt)
	for _, s := range sources {
		if containsBounded(s.norm, q) {
			return Result{Status: Verbatim, Source: s.Name, Passage: q, Changes: []Change{}}
		}
	}
	qw := strings.Fields(q)
	best := Result{Status: NotFound, Distance: -1, Changes: []Change{}}
	for _, s := range sources {
		d, start, end := nearest(qw, s.words)
		if d < 0 {
			continue
		}
		if best.Distance < 0 || d < best.Distance {
			pw := s.words[start:end]
			best = Result{
				Status:   Differs,
				Source:   s.Name,
				Passage:  strings.Join(pw, " "),
				Distance: d,
				Changes:  changes(qw, pw),
			}
		}
	}
	if best.Distance < 0 || 2*best.Distance > len(qw) {
		return Result{Status: NotFound, Changes: []Change{}}
	}
	if best.Distance == 0 { // equal up to edge punctuation
		best.Status, best.Passage = Verbatim, q
	}
	return best
}

// containsBounded reports whether q occurs in s with no letter or digit
// directly before or after it.
func containsBounded(s, q string) bool {
	if q == "" {
		return false
	}
	for off := 0; ; {
		i := strings.Index(s[off:], q)
		if i < 0 {
			return false
		}
		i += off
		before, _ := utf8.DecodeLastRuneInString(s[:i])
		after, _ := utf8.DecodeRuneInString(s[i+len(q):])
		if !isWordRune(before) && !isWordRune(after) {
			return true
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		off = i + size
	}
}

// wordEq compares quote word number i (of n) with a source word. The
// words must be equal, except that the quote's 1st word may drop leading
// punctuation of the source word and its last word may drop trailing
// punctuation, the same leeway that a verbatim match gets at its ends.
func wordEq(i, n int, q, s string) bool {
	if q == s {
		return true
	}
	if i == n-1 && strings.HasPrefix(s, q) && onlyPunct(s[len(q):]) {
		return true
	}
	if i == 0 && strings.HasSuffix(s, q) && onlyPunct(s[:len(s)-len(q)]) {
		return true
	}
	return false
}

func onlyPunct(s string) bool {
	for _, r := range s {
		if isWordRune(r) {
			return false
		}
	}
	return s != ""
}

func isWordRune(r rune) bool {
	return r != utf8.RuneError && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

// nearest finds the run of source words with the smallest word-level edit
// distance to q (an approximate substring search with a free start and a
// free end). It returns the distance and the run's bounds, or -1 when the
// source has no words.
func nearest(q, src []string) (dist, start, end int) {
	if len(src) == 0 || len(q) == 0 {
		return -1, 0, 0
	}
	n := len(src)
	prev := make([]int, n+1)   // distances for quote word i-1
	cur := make([]int, n+1)    // distances for quote word i
	prevSt := make([]int, n+1) // start of the run behind each cell
	curSt := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j], prevSt[j] = 0, j
	}
	for i := 1; i <= len(q); i++ {
		cur[0], curSt[0] = i, 0
		for j := 1; j <= n; j++ {
			sub := prev[j-1]
			if !wordEq(i-1, len(q), q[i-1], src[j-1]) {
				sub++
			}
			d, st := sub, prevSt[j-1]
			if del := prev[j] + 1; del < d { // quote word with no source word
				d, st = del, prevSt[j]
			}
			if ins := cur[j-1] + 1; ins < d { // source word with no quote word
				d, st = ins, curSt[j-1]
			}
			cur[j], curSt[j] = d, st
		}
		prev, cur = cur, prev
		prevSt, curSt = curSt, prevSt
	}
	dist, end = prev[1], 1
	for j := 2; j <= n; j++ {
		if prev[j] < dist {
			dist, end = prev[j], j
		}
	}
	return dist, prevSt[end], end
}

// changes lists the runs of differing words between q and p.
func changes(q, p []string) []Change {
	m, n := len(q), len(p)
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			c := d[i-1][j-1]
			if !wordEq(i-1, m, q[i-1], p[j-1]) {
				c++
			}
			c = min(c, d[i-1][j]+1, d[i][j-1]+1)
			d[i][j] = c
		}
	}
	// Walk back from the end, then reverse, grouping adjacent edits.
	type op struct{ q, p string }
	var ops []op // a zero op marks an equal word
	i, j := m, n
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && wordEq(i-1, m, q[i-1], p[j-1]) && d[i][j] == d[i-1][j-1]:
			ops = append(ops, op{})
			i, j = i-1, j-1
		case i > 0 && j > 0 && d[i][j] == d[i-1][j-1]+1:
			ops = append(ops, op{q[i-1], p[j-1]})
			i, j = i-1, j-1
		case i > 0 && d[i][j] == d[i-1][j]+1:
			ops = append(ops, op{q: q[i-1]})
			i--
		default:
			ops = append(ops, op{p: p[j-1]})
			j--
		}
	}
	out := []Change{}
	var cq, cp []string
	flush := func() {
		if cq != nil || cp != nil {
			out = append(out, Change{Quote: strings.Join(cq, " "), Source: strings.Join(cp, " ")})
		}
		cq, cp = nil, nil
	}
	for k := len(ops) - 1; k >= 0; k-- {
		o := ops[k]
		if o == (op{}) {
			flush()
			continue
		}
		if o.q != "" {
			cq = append(cq, o.q)
		}
		if o.p != "" {
			cp = append(cp, o.p)
		}
	}
	flush()
	return out
}
