package extract

import (
	"reflect"
	"strings"
	"testing"
)

func texts(qs []Quote) []string {
	out := []string{}
	for _, q := range qs {
		out = append(out, q.Text)
	}
	return out
}

func TestStraightAndCurly(t *testing.T) {
	doc := "He said \"one two three\" and “four five six”."
	got := texts(Quotes(doc, Options{MinWords: 1}))
	want := []string{"one two three", "four five six"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLineNumbers(t *testing.T) {
	doc := "# Title\n\nFirst \"a b c\".\nSecond line \"d e f\".\n"
	qs := Quotes(doc, Options{MinWords: 1})
	if len(qs) != 2 || qs[0].Line != 3 || qs[1].Line != 4 {
		t.Fatalf("got %+v", qs)
	}
}

func TestQuoteRunsOverLines(t *testing.T) {
	doc := "Intro \"a quote that\nruns over 2 lines\" end."
	qs := Quotes(doc, Options{MinWords: 1})
	if len(qs) != 1 || qs[0].Line != 1 || qs[0].Text != "a quote that\nruns over 2 lines" {
		t.Fatalf("got %+v", qs)
	}
}

func TestBlankLineEndsAnUnclosedQuote(t *testing.T) {
	doc := "An \"unclosed mark here\n\nThen \"a b c\" closes."
	got := texts(Quotes(doc, Options{MinWords: 1}))
	if !reflect.DeepEqual(got, []string{"a b c"}) {
		t.Fatalf("got %q", got)
	}
}

func TestFencedCodeIsSkipped(t *testing.T) {
	doc := "```\nx = \"in a fence\"\n```\n~~~\ny = \"tilde fence\"\n~~~\nOut \"of the fence\"."
	got := texts(Quotes(doc, Options{MinWords: 1}))
	if !reflect.DeepEqual(got, []string{"of the fence"}) {
		t.Fatalf("got %q", got)
	}
}

func TestInlineCodeMarksAreSkipped(t *testing.T) {
	doc := "Call `F.expr(\"'Senior'\")` and then \"a real quote\"."
	got := texts(Quotes(doc, Options{MinWords: 1}))
	if !reflect.DeepEqual(got, []string{"a real quote"}) {
		t.Fatalf("got %q", got)
	}
}

func TestCodeInsideAQuoteIsKept(t *testing.T) {
	doc := "It says \"the `x` value is kept\"."
	got := texts(Quotes(doc, Options{MinWords: 1}))
	if !reflect.DeepEqual(got, []string{"the `x` value is kept"}) {
		t.Fatalf("got %q", got)
	}
}

func TestMinWords(t *testing.T) {
	doc := "Short \"no\" and \"a b\" and \"a b c\"."
	got := texts(Quotes(doc, Options{MinWords: 3}))
	if !reflect.DeepEqual(got, []string{"a b c"}) {
		t.Fatalf("got %q", got)
	}
}

func TestStraightInsideCurlyIsNested(t *testing.T) {
	doc := "“she wrote \"yes\" twice”"
	got := texts(Quotes(doc, Options{MinWords: 1}))
	want := []string{"she wrote \"yes\" twice", "yes"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCRLF(t *testing.T) {
	got := Quotes("a\r\n\r\nb \"c d e\"\r\n", Options{MinWords: 1})
	if len(got) != 1 || got[0].Line != 3 || got[0].Text != "c d e" {
		t.Fatalf("got %+v", got)
	}
}

func TestSingleMarks(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want []string
	}{
		{"curly", "She said ‘cool it for 10 minutes’ first.", []string{"cool it for 10 minutes"}},
		{"straight", "She said 'cool it for 10 minutes' first.", []string{"cool it for 10 minutes"}},
		{"straight at start", "'A quote' opens the line.", []string{"A quote"}},
		{"after a bracket", "(see 'the manual' here)", []string{"the manual"}},
		{"after an em dash", "it says—'run it dry'—twice", []string{"run it dry"}},
		{"punctuation inside", "'Yes, wait,' she said.", []string{"Yes, wait,"}},
		{"over 2 lines", "He wrote 'a quote that\nruns on' here.", []string{"a quote that\nruns on"}},
		{"apostrophes are text", "Don't stop, it's the 90's and they're here.", []string{}},
		{"curly apostrophes are text", "Don’t stop, it’s here.", []string{}},
		{"apostrophe inside a quote", "He said 'don't stop the pump' twice.", []string{"don't stop the pump"}},
		{"curly apostrophe inside a quote", "He said ‘don’t stop the pump’ twice.", []string{"don’t stop the pump"}},
		{"rock 'n' roll", "It was rock 'n' roll all night.", []string{"n"}},
		{"elision never closes", "'Tis true and that is all.", []string{}},
		{"elision pairs with a later closing mark", "'Tis the season,' she said.", []string{"Tis the season,"}},
		{"plural possessive ends a quote early", "He said 'the students' books are here' today.", []string{"the students"}},
		{"a curly mark does not pair with a straight 1", "‘one two three' ends here.", []string{}},
	}
	for _, c := range cases {
		got := texts(Quotes(c.doc, Options{MinWords: 1}))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestNestedQuotes(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want []string
	}{
		{"single in curly double", "“The manual says ‘cool it’ first.”", []string{"The manual says ‘cool it’ first.", "cool it"}},
		{"single in straight double", "\"The manual says 'cool it' first.\"", []string{"The manual says 'cool it' first.", "cool it"}},
		{"double in single", "'He said \"go now\" twice'", []string{"He said \"go now\" twice", "go now"}},
		{"3 levels", "“a ‘b \"c d\" e’ f”", []string{"a ‘b \"c d\" e’ f", "b \"c d\" e", "c d"}},
		{"unclosed inner is dropped", "“He said ‘wait now”", []string{"He said ‘wait now"}},
		{"unclosed inner ends with its outer quote", "“He said ‘wait now” and then it ended.’", []string{"He said ‘wait now"}},
		{"2 inner quotes", "“‘one’ and ‘two’”", []string{"‘one’ and ‘two’", "one", "two"}},
	}
	for _, c := range cases {
		got := texts(Quotes(c.doc, Options{MinWords: 1}))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestNestedLineNumbers(t *testing.T) {
	doc := "Intro.\n\n“Outer starts here\nand ‘inner on line 4’ ends.”"
	qs := Quotes(doc, Options{MinWords: 1})
	if len(qs) != 2 || qs[0].Line != 3 || qs[1].Line != 4 {
		t.Fatalf("got %+v", qs)
	}
}

func TestMinWordsAppliesToEachLevel(t *testing.T) {
	doc := "“The manual says ‘cool it’ before the next run.”"
	got := texts(Quotes(doc, Options{MinWords: 3}))
	if !reflect.DeepEqual(got, []string{"The manual says ‘cool it’ before the next run."}) {
		t.Fatalf("got %q", got)
	}
}

func TestSingleMarksInCodeAreSkipped(t *testing.T) {
	doc := "Run `echo 'a b c'` and then 'd e f' here."
	got := texts(Quotes(doc, Options{MinWords: 1}))
	if !reflect.DeepEqual(got, []string{"d e f"}) {
		t.Fatalf("got %q", got)
	}
}

// FuzzQuotes checks rules that hold for any input: no panic, every quote
// is a part of the document, every line number lies inside it, the
// quotes come in the order of their lines.
func FuzzQuotes(f *testing.F) {
	for _, s := range []string{
		"“a ‘b \"c\" d’ e”", "'x' y 'z", "don't 'a' it's", "```\n'a'\n```\n'b'",
		"\"a\n\nb\"", "‘’’‘''\"\"“”", "`'` 'a b'", "‘\x80’",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, doc string) {
		qs := Quotes(doc, Options{MinWords: 1})
		// Each invalid UTF-8 byte becomes U+FFFD, as in a range loop.
		norm := string([]rune(strings.ReplaceAll(doc, "\r\n", "\n")))
		lines := strings.Count(norm, "\n") + 1
		last := 0
		for _, q := range qs {
			if !strings.Contains(norm, q.Text) {
				t.Fatalf("quote %q is not in the document", q.Text)
			}
			if q.Line < 1 || q.Line > lines || q.Line < last {
				t.Fatalf("line %d out of place (last %d, lines %d)", q.Line, last, lines)
			}
			last = q.Line
		}
	})
}
