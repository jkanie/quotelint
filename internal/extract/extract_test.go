package extract

import (
	"reflect"
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

func TestStraightInsideCurlyIsText(t *testing.T) {
	doc := "“she wrote \"yes\" twice”"
	got := texts(Quotes(doc, Options{MinWords: 1}))
	if !reflect.DeepEqual(got, []string{"she wrote \"yes\" twice"}) {
		t.Fatalf("got %q", got)
	}
}

func TestCRLF(t *testing.T) {
	got := Quotes("a\r\n\r\nb \"c d e\"\r\n", Options{MinWords: 1})
	if len(got) != 1 || got[0].Line != 3 || got[0].Text != "c d e" {
		t.Fatalf("got %+v", got)
	}
}
