# Roadmap

Planned work after the seed, with an hour estimate per item. The total is 30 hours.

## 1. Single quotation marks and quotes inside quotes, 2.5 hours

- What: find passages between single marks (`‘ ’`, and `' '` where a rule can tell them from apostrophes), plus quotes nested inside quotes.
- Why: British style and nested quotes use single marks. Today a quote such as ‘a quote’ is never checked, and the tool says nothing about it.

## 2. Full HTML parser and pre-formatted text, 2.5 hours

- What: replace the hand-written tag stripper with the HTML5 parser from `golang.org/x/net/html`, keep line breaks inside `pre`, and drop hidden elements (`hidden`, `aria-hidden`).
- Why: the stripper handles well-formed pages, but real pages hold unclosed tags, `<` inside attributes, text that only a full parser reads the way a browser does.

## 3. Sources by URL, with a cache, 3 hours

- What: accept `https://` sources, fetch each page once into a local cache keyed by URL and fetch time, then work offline from the cache. A `--refresh` option fetches again.
- Why: today each source page must be saved by hand before a check. A cache also lets a re-run show the same result after the page changes.

## 4. Pair each quote with its own source, 3 hours

- What: read the Markdown link, footnote or "Source:" line that follows a quote, and check that quote only against the source it names. Quotes with no named source still go against all sources.
- Why: a quote that matches the wrong source is a fault, and checking every quote against every source is slow on large documents.

## 5. Ellipses, brackets and [sic], 2.5 hours

- What: treat `...`, `…` and `[...]` inside a quote as a gap of any length, accept text in square brackets as the writer's own insertion, and report `[sic]`.
- Why: shortened quotes are normal practice. Today a quote with an ellipsis can never be verbatim, which makes the tool flag correct quotes.

## 6. Word documents as sources and as the document, 2 hours

- What: read `.docx` files with `archive/zip` and `encoding/xml` from the standard library, as a source and as the document to check.
- Why: letters, contracts, reports written in Word must be copied to text before a check today, and the copy loses the line numbers that the report gives.

## 7. Faster search in large sources, 2.5 hours

- What: index each source's words once, then align a quote only against the places that hold its rarest words, with the full search as a fall-back.
- Why: the search costs the quote's word count times the word count of every source. A folder of book-length sources with 500 quotes needs an index to stay under 1 second.

## 8. Character-level detail inside changed words, 2 hours

- What: show the differing characters inside a changed word (such as `store[s]`), and colour the change on a terminal.
- Why: a 1-character change (a hyphen, an accent, a curly apostrophe) is the hardest kind to see in a 2-line quote and source pair.

## 9. Allow-list for known paraphrases, 1.5 hours

- What: a file of quote texts or line numbers that the check skips, each with a reason, plus a count of skipped quotes in the summary.
- Why: a quoted title or term is not a quote of a source. Without a way to mark it, every run lists the same false hit.

## 10. Configuration file, 1.5 hours

- What: a JSON file next to the documents that names their sources, the file endings that a folder search reads, the `--min-words` and `--fold-quotes` settings, so that a plain `quotelint DOC` works.
- Why: a set of documents with 10 sources needs a long command line today, and the same set has to be typed on every run. The file endings of a folder search are fixed today.

## 11. Unicode normalisation option, 1.5 hours

- What: `--nfc` to compare quote and source after Unicode NFC normalisation, using `golang.org/x/text`.
- Why: an accented letter can be stored as 1 code point or as 2. The 2 look the same, but today they count as different words.

## 12. HTML report, 2 hours

- What: `--html FILE` writes a single self-contained page with each quote, its source passage, the changed words marked, plus a filter by status.
- Why: a long text report is hard to read and hard to share. A page opens in any browser and can go to an editor who does not use a terminal.

## 13. UTF-16 files, 1.5 hours

- What: read text files that start with a UTF-16 byte order mark (little-endian or big-endian) with `unicode/utf16` from the standard library, both as sources and as documents.
- Why: Windows PowerShell 5.1 writes UTF-16 with `>`, and Notepad saves it as "UTF-16 LE". Today, a UTF-16 source stops the run with "not UTF-8 text", and a UTF-16 document is read as bytes, which puts a zero byte between the letters of each quote and reports every quote as not found.

## 14. Release v0.1.0, 2 hours

- What: write a change log, tag v0.1.0, publish binaries for the 3 platforms, make the module installable with `go install github.com/jkanie/quotelint@latest`.
- Why: a tagged release with binaries lets people use the tool without a Go set-up, and it gives later releases a fixed point to compare against.

## Total

30 hours.
