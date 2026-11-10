# Roadmap

Planned work after the seed, with an hour estimate per item. The total is 24.5 hours.

## 1. Pair each quote with its own source, 3 hours

- What: read the Markdown link, footnote or "Source:" line that follows a quote, and check that quote only against the source it names. Quotes with no named source still go against all sources.
- Why: a quote that matches the wrong source is a fault, and checking every quote against every source is slow on large documents.

## 2. Ellipses, brackets and [sic], 2.5 hours

- What: treat `...`, `…` and `[...]` inside a quote as a gap of any length, accept text in square brackets as the writer's own insertion, and report `[sic]`.
- Why: shortened quotes are normal practice. Today a quote with an ellipsis can never be verbatim, which makes the tool flag correct quotes.

## 3. Word documents as sources and as the document, 2 hours

- What: read `.docx` files with `archive/zip` and `encoding/xml` from the standard library, as a source and as the document to check.
- Why: letters, contracts, reports written in Word must be copied to text before a check today, and the copy loses the line numbers that the report gives.

## 4. Faster search in large sources, 2.5 hours

- What: index each source's words once, then align a quote only against the places that hold its rarest words, with the full search as a fall-back.
- Why: the search costs the quote's word count times the word count of every source. A folder of book-length sources with 500 quotes needs an index to stay under 1 second.

## 5. Character-level detail inside changed words, 2 hours

- What: show the differing characters inside a changed word (such as `store[s]`), and colour the change on a terminal.
- Why: a 1-character change (a hyphen, an accent, a curly apostrophe) is the hardest kind to see in a 2-line quote and source pair.

## 6. Allow-list for known paraphrases, 1.5 hours

- What: a file of quote texts or line numbers that the check skips, each with a reason, plus a count of skipped quotes in the summary.
- Why: a quoted title or term is not a quote of a source. Without a way to mark it, every run lists the same false hit.

## 7. Configuration file, 1.5 hours

- What: a JSON file next to the documents that names their sources, the file endings that a folder search reads, the `--min-words` and `--fold-quotes` settings, so that a plain `quotelint DOC` works.
- Why: a set of documents with 10 sources needs a long command line today, and the same set has to be typed on every run. The file endings of a folder search are fixed today.

## 8. Unicode normalisation option, 1.5 hours

- What: `--nfc` to compare quote and source after Unicode NFC normalisation, using `golang.org/x/text`.
- Why: an accented letter can be stored as 1 code point or as 2. The 2 look the same, but today they count as different words.

## 9. HTML report, 2 hours

- What: `--html FILE` writes a single self-contained page with each quote, its source passage, the changed words marked, plus a filter by status.
- Why: a long text report is hard to read and hard to share. A page opens in any browser and can go to an editor who does not use a terminal.

## 10. UTF-16 files, 1.5 hours

- What: read text files that start with a UTF-16 byte order mark (little-endian or big-endian) with `unicode/utf16` from the standard library, both as sources and as documents.
- Why: Windows PowerShell 5.1 writes UTF-16 with `>`, and Notepad saves it as "UTF-16 LE". Today, a UTF-16 source stops the run with "not UTF-8 text", and a UTF-16 document is read as bytes, which puts a zero byte between the letters of each quote and reports every quote as not found.

## 11. Look-ahead for single marks, 2.5 hours

- What: before a single mark closes or opens a quote, look ahead in the paragraph for a better pairing, plus a list of common elisions (`'tis`, `'em`, `'n'`, `'90s`).
- Why: today, a plural possessive such as `students'` ends a single-quoted passage early. A leading elision opens a quote (the README section "What it does" lists both cases). Each gives a wrong quote text, which is then reported as differs or not found.

## 12. Release v0.1.0, 2 hours

- What: write a change log, tag v0.1.0, publish binaries for the 3 platforms, make the module installable with `go install github.com/jkanie/quotelint@latest`.
- Why: a tagged release with binaries lets people use the tool without a Go set-up, and it gives later releases a fixed point to compare against.

## Total

27.5 hours.
