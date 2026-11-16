package match

import (
	"reflect"
	"testing"
)

func src(name, text string) []Text { return []Text{Prepare(name, text, Options{})} }

func TestNormalize(t *testing.T) {
	got := Normalize("  a\t b\n c  ", Options{})
	if got != "a b c" {
		t.Fatalf("got %q", got)
	}
	if got := Normalize("“it’s”", Options{FoldQuotes: true}); got != "\"it's\"" {
		t.Fatalf("fold: got %q", got)
	}
	if got := Normalize("“it’s”", Options{}); got != "“it’s”" {
		t.Fatalf("no fold: got %q", got)
	}
}

func TestVerbatimAcrossLineBreaks(t *testing.T) {
	r := Check("the default method\nis EXACT.", src("s", "Unless you use it, the default\n  method is EXACT. More."), Options{})
	if r.Status != Verbatim || r.Source != "s" {
		t.Fatalf("got %+v", r)
	}
}

func TestVerbatimNeedsWordBoundaries(t *testing.T) {
	r := Check("tores and return", src("s", "it stores and return values"), Options{})
	if r.Status != Differs || !reflect.DeepEqual(r.Changes, []Change{{"tores", "stores"}}) {
		t.Fatalf("got %+v", r)
	}
	r = Check("is EXACT", src("s", "method is EXACT."), Options{})
	if r.Status != Verbatim {
		t.Fatalf("punctuation after: got %+v", r)
	}
}

func TestReplacedWords(t *testing.T) {
	q := "a function that is executed conditionally will store and return values"
	s := "Note that a function that is executed conditionally stores and return values only."
	r := Check(q, src("lag", s), Options{})
	if r.Status != Differs || r.Distance != 2 {
		t.Fatalf("got %+v", r)
	}
	if r.Passage != "a function that is executed conditionally stores and return values" {
		t.Fatalf("passage %q", r.Passage)
	}
	if !reflect.DeepEqual(r.Changes, []Change{{"will store", "stores"}}) {
		t.Fatalf("changes %+v", r.Changes)
	}
}

func TestMissingAndExtraWords(t *testing.T) {
	r := Check("one two four five six", src("s", "zero one two three four five six seven"), Options{})
	if r.Status != Differs || r.Distance != 1 || !reflect.DeepEqual(r.Changes, []Change{{"", "three"}}) {
		t.Fatalf("missing: got %+v", r)
	}
	r = Check("one two extra three four five", src("s", "zero one two three four five six"), Options{})
	if r.Status != Differs || r.Distance != 1 || !reflect.DeepEqual(r.Changes, []Change{{"extra", ""}}) {
		t.Fatalf("extra: got %+v", r)
	}
}

func TestNotFound(t *testing.T) {
	r := Check("the moon is made of cheese", src("s", "SAS stores all numeric values in 8-byte floating-point representation."), Options{})
	if r.Status != NotFound || r.Source != "" {
		t.Fatalf("got %+v", r)
	}
	if r := Check("anything at all", nil, Options{}); r.Status != NotFound {
		t.Fatalf("no sources: got %+v", r)
	}
}

func TestBestSourceWins(t *testing.T) {
	ts := []Text{
		Prepare("far", "alpha beta x y z", Options{}),
		Prepare("near", "alpha beta gamma delta epsilon zeta", Options{}),
	}
	r := Check("alpha beta gamma delta epsilon theta", ts, Options{})
	if r.Source != "near" || r.Distance != 1 {
		t.Fatalf("got %+v", r)
	}
}

func TestFoldQuotesMakesVerbatim(t *testing.T) {
	s := "SAS writes “missing” for it’s value."
	q := "SAS writes \"missing\" for it's value."
	if r := Check(q, src("s", s), Options{}); r.Status == Verbatim {
		t.Fatalf("strict: got %+v", r)
	}
	ts := []Text{Prepare("s", s, Options{FoldQuotes: true})}
	if r := Check(q, ts, Options{FoldQuotes: true}); r.Status != Verbatim {
		t.Fatalf("folded: got %+v", r)
	}
}

func TestFoldDashesMakesVerbatim(t *testing.T) {
	// En dash, em dash, minus sign, non-breaking hyphen in the source.
	s := "pages 10\u201312 \u2014 not \u22121 \u2014 in the long\u2011term plan"
	q := "pages 10-12 - not -1 - in the long-term plan"
	if r := Check(q, src("s", s), Options{}); r.Status == Verbatim {
		t.Fatalf("strict: got %+v", r)
	}
	opt := Options{FoldDashes: true}
	if r := Check(q, []Text{Prepare("s", s, opt)}, opt); r.Status != Verbatim {
		t.Fatalf("folded: got %+v", r)
	}
	// The spaces around a dash still count.
	if r := Check("a-b c d", []Text{Prepare("s", "a \u2014 b c d", opt)}, opt); r.Status == Verbatim {
		t.Fatalf("spaces: got %+v", r)
	}
	if got := Normalize("\u2010\u2011\u2012\u2013\u2014\u2015\u2212", opt); got != "-------" {
		t.Fatalf("Normalize: got %q", got)
	}
}

func TestFoldCaseMakesVerbatim(t *testing.T) {
	s := "the method stores and returns values only from the observations."
	q := "The method stores and returns values only from the observations."
	if r := Check(q, src("s", s), Options{}); r.Status == Verbatim {
		t.Fatalf("strict: got %+v", r)
	}
	opt := Options{FoldCase: true}
	if r := Check(q, []Text{Prepare("s", s, opt)}, opt); r.Status != Verbatim {
		t.Fatalf("folded: got %+v", r)
	}
	// A real word change still differs with case folded.
	r := Check("The method stores and keeps values only from the observations.", []Text{Prepare("s", s, opt)}, opt)
	if r.Status != Differs || r.Distance != 1 || !reflect.DeepEqual(r.Changes, []Change{{"keeps", "returns"}}) {
		t.Fatalf("change: got %+v", r)
	}
}

func TestEdgePunctuationIsLeeway(t *testing.T) {
	s := "(The filter should be rinsed after every 5 runs.) Next."
	r := Check("filter must be rinsed after every 5 runs", src("s", s), Options{})
	if r.Status != Differs || r.Distance != 1 || !reflect.DeepEqual(r.Changes, []Change{{"must", "should"}}) {
		t.Fatalf("got %+v", r)
	}
	if r.Passage != "filter should be rinsed after every 5 runs.)" {
		t.Fatalf("passage %q", r.Passage)
	}
	// Inside the quote, punctuation still counts.
	r = Check("The filter should, be rinsed", src("s", s), Options{})
	if r.Status != Differs || !reflect.DeepEqual(r.Changes, []Change{{"should,", "should"}}) {
		t.Fatalf("inner: got %+v", r)
	}
}
