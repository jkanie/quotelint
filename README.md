# quotelint

A command-line tool that checks every quoted passage in your documents against their source files. It lists each quote that does not match its source word for word, with the nearest passage in the sources and the words that differ.

## What it does

- Finds every passage between double quotation marks in a Markdown or text file, straight (`" "`) or curly (`“ ”`), with the line number of its opening mark.
- Checks 1 document or many. A document can be a file or a folder of `.md`, `.markdown`, `.txt`, `.text` files.
- Takes 1 source or many. A source can be a file or a folder of text and HTML files (`.html`, `.htm` added to the document endings).
- Reads folders recursively, in sorted order, and skips every name that starts with a dot (such as `.git`).
- Never checks a document against itself when the document also sits in a source folder.
- Stops with an error when a folder holds no file to read, so that a typo in a path never passes as "nothing to check".
- Skips quotation marks inside fenced code blocks and inline code spans.
- Ends an unclosed quote at the next blank line so that 1 stray mark never swallows the rest of the document.
- Turns an HTML source into its visible text first:
  - Inline tags (such as `em`, `span`, `a`) leave no gap, so `LAG<em>n</em>` reads as `LAGn`, the way a browser shows it.
  - Other tags leave a space.
  - Comments and the content of `script`, `style`, `noscript`, `template` are dropped.
  - Character references such as `&amp;` and `&nbsp;` are decoded.
- Compares quote and source after it collapses each run of white space (no-break spaces included) to 1 space. No other character changes unless `--fold-quotes` is given.
- Marks a quote as verbatim when its text occurs in a source with no letter or digit glued to either end.
- Otherwise, finds the nearest passage in any source (the run of source words with the fewest word edits) and lists the changed words. A quote that needs more edits than half its words counts as not found.
- Prints a JSON object instead of text with `--json`. Keys come out in sorted order so that 2 runs on the same files print the same bytes.
- Exits with 0 when every quote is verbatim, 1 when any quote differs or is not found, 2 on an error.

## Who it is for

Anyone who quotes sources in their writing (reports, reviews, letters, documentation) and needs every quote to match its source exactly. It answers 1 question fast -- does each quote say what the source says?

## Built with

- Go, built and tested with Go 1.24.7. The module needs Go 1.24 or later.
- The Go standard library only, with no third-party module.

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

## Options

```text
$ quotelint --help
Usage: quotelint [OPTIONS] DOC SOURCE...
       quotelint [OPTIONS] -s SOURCE [-s SOURCE]... DOC...

Check every quoted passage in each DOC against the SOURCE files.
A DOC is a Markdown or text file, or a folder of them (.md, .markdown,
.txt, .text). A SOURCE is a text file, an HTML file (.html, .htm) that
is turned into text first, or a folder of such files. Folders are read
recursively, skipping names that start with a dot.

With no -s, the 1st path is the DOC and the rest are SOURCEs.

Options:
  -s, --source PATH  Add a SOURCE file or folder (repeatable). Every
                     other path is then a DOC.
  --min-words N      Check only quotes of N or more words (default 3).
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

## Licence

MIT Licence. See [LICENSE](LICENSE).
