# quotelint

A command-line tool that checks every quoted passage in your documents against their source files. It lists each quote that does not match its source word for word, with the nearest passage in the sources and the words that differ.

## What it does

- Finds every passage between quotation marks in a Markdown or text file, with the line number of its opening mark:
  - Double marks, straight (`" "`) or curly (`“ ”`).
  - Single marks, curly (`‘ ’`) or straight (`' '`).
  - A quote inside a quote, such as `“the manual says ‘cool it’ first”`. Both are checked: the inner quote on its own, the outer quote with the inner marks in its text. The results come in the order of the opening marks.
- Tells a single mark from an apostrophe by the characters on either side of it:
  - A straight `'` opens a quote only at the start of a paragraph or after 1 of these: white space, an opening bracket, an opening mark, an em dash, an en dash, `/`. A letter or digit must follow it.
  - A single mark (`'` or `’`) closes a quote only after a character that is not white space and only when no letter or digit follows it.
  - A mark between 2 letters or digits is therefore an apostrophe, so `don't`, `it’s`, `the 90's` never start or end a quote.
  - A curly mark pairs only with a curly mark. A straight mark pairs only with a straight 1.
- Does not yet handle these single-mark cases:
  - A plural possessive inside a single-quoted passage ends the quote early. `'the students' books are here'` gives the quote `the students`.
  - An elision at the start of a word (such as `'tis` or `'90s`) opens a quote. A later closing mark in the same paragraph pairs with it. When no mark closes it, it is dropped at the end of the paragraph.
  - `rock 'n' roll` gives a 1-word quote (`n`), which the default `--min-words 3` skips.
- Checks 1 document or many. A document can be a file or a folder of `.md`, `.markdown`, `.txt`, `.text` files.
- Takes 1 source or many. A source can be a file, a folder of text and HTML files (`.html`, `.htm` added to the document endings), a web page URL that is fetched once and kept in a cache.
- Reads folders recursively, in sorted order, and skips every name that starts with a dot (such as `.git`).
- Never checks a document against itself when the document also sits in a source folder.
- Stops with an error when a folder holds no file to read, so that a typo in a path never passes as "nothing to check".
- Skips quotation marks inside fenced code blocks and inline code spans.
- Ends an unclosed quote at the next blank line so that 1 stray mark never swallows the rest of the document.
- Turns an HTML source into its visible text first:
  - Reads the page with the HTML5 parser from `golang.org/x/net/html`, which follows the parsing rules that browsers use. Unclosed tags, misnested tags, a `<` inside an attribute value are therefore read the way a browser reads them.
  - Inline tags (such as `em`, `span`, `a`) leave no gap, so `LAG<em>n</em>` reads as `LAGn`, the way a browser shows it.
  - Other tags leave a space.
  - Comments and the content of `script`, `style`, `noscript`, `template` are dropped.
  - Elements with a `hidden` attribute or with `aria-hidden="true"` are dropped with their content.
  - Inside `pre`, the text keeps its line breaks, and a `br` gives a line break. Elsewhere, each run of spaces, tabs, line breaks becomes 1 space. The comparison counts a line break as a space either way.
  - Character references such as `&amp;` and `&nbsp;` are decoded.
  - A page nested deeper than 512 elements stops the run with an error (exit status 2). 512 is the limit of the parser.
- Compares quote and source after it collapses each run of white space (no-break spaces included) to 1 space. No other character changes unless `--fold-quotes` is given.
- Marks a quote as verbatim when its text occurs in a source with no letter or digit glued to either end.
- Otherwise, finds the nearest passage in any source (the run of source words with the fewest word edits) and lists the changed words. A quote that needs more edits than half its words counts as not found.
- Prints a JSON object instead of text with `--json`. Keys come out in sorted order so that 2 runs on the same files print the same bytes.
- Exits with 0 when every quote is verbatim, 1 when any quote differs or is not found, 2 on an error.

## Who it is for

Anyone who quotes sources in their writing (reports, reviews, letters, documentation) and needs every quote to match its source exactly. It answers 1 question fast -- does each quote say what the source says?

## Built with

- Go, built and tested with Go 1.27.3 and Go 1.26.0. The module needs Go 1.26.0 or later.
- The Go standard library, plus `golang.org/x/net` v0.62.0 for its HTML5 parser:
  - v0.62.0 holds the fixes for the parser bugs GO-2026-5025, GO-2026-5027, GO-2026-5028, GO-2026-5029, GO-2026-5030 (fixed in v0.55.0), GO-2026-6860 (fixed in v0.62.0). See [pkg.go.dev/vuln](https://pkg.go.dev/vuln/) for each ID.
  - v0.62.0 needs Go 1.26.0, which sets the minimum of this module. The last release that supported Go 1.24, v0.50.0, still has these bugs.

The argument parser is hand-written, with no parser package.

## Build and run

```sh
go build
./quotelint DOC SOURCE...
./quotelint -s SOURCE DOC...
```

Run the tests:

```sh
go test ./...
```

On every push and pull request, GitHub Actions runs `gofmt -l`, `go vet ./...`, `staticcheck ./...`, `go test ./...` on Linux, Windows, macOS with Go 1.27.3, then `go test ./...` on Linux with Go 1.26.0, the oldest release that the module supports (see [.github/workflows/ci.yml](.github/workflows/ci.yml)).

## Options

```text
$ quotelint --help
Usage: quotelint [OPTIONS] DOC SOURCE...
       quotelint [OPTIONS] -s SOURCE [-s SOURCE]... DOC...

Check every quoted passage in each DOC against the SOURCE files.
A quoted passage sits between double or single quotation marks. A
quote inside a quote is checked as well as the quote around it.
A DOC is a Markdown or text file, or a folder of them (.md, .markdown,
.txt, .text). A SOURCE is a text file, an HTML file (.html, .htm) that
is turned into text first, or a folder of such files. Folders are read
recursively, skipping names that start with a dot. A SOURCE that starts
with http:// or https:// is a web page: it is fetched once into the
cache, and later runs read it from there, offline.

With no -s, the 1st path is the DOC and the rest are SOURCEs.

Options:
  -s, --source PATH  Add a SOURCE file or folder (repeatable). Every
                     other path is then a DOC.
  --min-words N      Check only quotes of N or more words (default 3).
  --refresh          Fetch every web page SOURCE again, then update the
                     cache.
  --cache-dir DIR    Keep fetched pages in DIR (default: quotelint in
                     the user cache folder).
  --fold-quotes      Treat curly and straight quotation marks as equal.
  --json             Print a JSON object instead of text.
  -h, --help         Print this help.
  -V, --version      Print the version.

Exit status: 0 when every quote is verbatim, 1 when any quote differs
or is not found, 2 on an error.
```

- `-s` lets 1 run check a whole folder of drafts (or several files) against the same sources.
- `--min-words` skips short quoted words and titles (such as "the" or "Run"), which are not quotes of a source.
- `--fold-quotes` maps the curly marks `‘ ’ ‚ ‛ “ ” „ ‟` to `'` and `"` on both sides. It is off by default because a quote should keep the source's own characters.
- `--` ends the options, so that a file whose name starts with `-` can be checked.
- A SOURCE that starts with `http://` or `https://` is a web page. See [Web page sources](#web-page-sources).

## Web page sources

```sh
quotelint notes.md https://example.com/manual/lag.html
```

- The 1st run fetches the page and keeps it in the cache: `quotelint` in the user cache folder (`~/.cache/quotelint` on Linux, `~/Library/Caches/quotelint` on macOS, `%LocalAppData%\quotelint` on Windows), or the folder named by `--cache-dir`. Each URL gets 1 JSON file, named by the SHA-256 of the URL, with the page, its content type, the URL it ended on after redirects, the fetch time in UTC.
- Later runs read the page from the cache, with no network. A re-run therefore gives the same result after the page changes. `--refresh` fetches every web page SOURCE of the run again and replaces its cache file.
- The text report lists each page before the summary, such as `page: https://example.com/a (from the cache, fetched 2026-11-09T23:40:05Z)`. The JSON report has a `pages` list with `url`, `final_url`, `fetched_at`, `from_cache`, only when a web page SOURCE was given.
- A page must answer 200 with `text/html` or `text/plain`, in UTF-8 (or with no charset and valid UTF-8 bytes), at 20 MiB or less, within 30 seconds. Anything else stops the run with exit code 2, and nothing is cached. An HTML page is turned into text the same way as an HTML file.
- Each request sends the User-Agent `quotelint (+https://github.com/jkanie/quotelint)`.
- A cache file that does not parse stops the run with exit code 2 and names the file. Delete it, or run with `--refresh`.

## Sample output

These runs are pasted from real runs on made-up files.

- `sources/manual.html` is a 2-paragraph page that says to cool the motor for 10 minutes and that the filter "should" be rinsed.
- `sources/warranty.txt` is a 3-line warranty.
- `review.md` quotes the manual 3 times, gets 2 of the quotes wrong and quotes a box claim that is not in the manual.
- The folder `drafts` holds `email.md`, which quotes the warranty twice, and `faq.txt`, which quotes the manual once.

1 document against 1 source:

```text
$ quotelint review.md sources/manual.html
review.md:3: verbatim in sources/manual.html: "Run the pump for no more than 20 minutes at a time."
review.md:5: differs from sources/manual.html (1 word edit)
  quote:  Let the motor cool for 15 minutes before the next run.
  source: Let the motor cool for 10 minutes before the next run.
  change: "15" -> "10"
review.md:7: differs from sources/manual.html (2 word edits)
  quote:  the filter must be rinsed after every 5 runs
  source: The filter should be rinsed after every 5 runs.
  change: "the" -> "The"
  change: "must" -> "should"
review.md:9: not found in any source: "Quiet enough for a library"
summary: 1 doc, 4 quotes, 1 verbatim, 2 differs, 1 not found
$ echo $?
1
```

The 1st word and the last word of a quote may leave out the punctuation that sits next to them in the source, as "runs" does against "runs." above. Inside a quote, every character counts.

A folder of documents against a folder of sources:

```text
$ quotelint -s sources drafts
drafts/email.md:3: verbatim in sources/warranty.txt: "covers parts and labour for 2 years from the date of purchas..."
drafts/email.md:5: differs from sources/warranty.txt (1 word edit)
  quote:  does not cover damage from running the pump while dry
  source: does not cover damage from running the pump dry.
  change: "while" -> (nothing)
drafts/faq.txt:2: verbatim in sources/manual.html: "Run the pump for no more than 20 minutes at a time."
summary: 2 docs, 3 quotes, 2 verbatim, 1 differs, 0 not found
$ echo $?
1
```

The same document and source as the 1st run, with only the quotes of 11 or more words, as JSON:

```text
$ quotelint --json --min-words 11 review.md sources/manual.html
{
  "docs": [
    "review.md"
  ],
  "match": false,
  "quotes": [
    {
      "changes": [],
      "distance": 0,
      "doc": "review.md",
      "line": 3,
      "passage": "Run the pump for no more than 20 minutes at a time.",
      "quote": "Run the pump for no more than 20 minutes at a time.",
      "source": "sources/manual.html",
      "status": "verbatim"
    },
    {
      "changes": [
        {
          "quote": "15",
          "source": "10"
        }
      ],
      "distance": 1,
      "doc": "review.md",
      "line": 5,
      "passage": "Let the motor cool for 10 minutes before the next run.",
      "quote": "Let the motor cool for 15 minutes before the next run.",
      "source": "sources/manual.html",
      "status": "differs"
    }
  ],
  "sources": [
    "sources/manual.html"
  ],
  "summary": {
    "differs": 1,
    "docs": 1,
    "not_found": 0,
    "quotes": 2,
    "verbatim": 1
  }
}
$ echo $?
1
```

`distance` is the number of word edits, and it is 0 for a quote that is verbatim or not found.

A run on `note.md` (a made-up service note) with single and nested quotes:

```text
$ cat note.md
# Service note

The owner wrote “the manual says ‘Let the motor cool for 10 minutes before the next run.’ and I have followed that rule every day since I bought the pump in May”.

Don't skip the rule that says 'the filter should be rinsed after every 3 runs'.
$ quotelint note.md sources/manual.html
note.md:3: not found in any source: "the manual says ‘Let the motor cool for 10 minutes before th..."
note.md:3: verbatim in sources/manual.html: "Let the motor cool for 10 minutes before the next run."
note.md:5: differs from sources/manual.html (2 word edits)
  quote:  the filter should be rinsed after every 3 runs
  source: The filter should be rinsed after every 5 runs.
  change: "the" -> "The"
  change: "3" -> "5"
summary: 1 doc, 3 quotes, 1 verbatim, 1 differs, 1 not found
$ echo $?
1
```

The outer quote on line 3 holds the owner's own words. It is therefore not in the manual. The inner quote is checked on its own and is verbatim. The apostrophe in `Don't` does not open a quote.

## Licence

MIT Licence. See [LICENSE](LICENSE).
