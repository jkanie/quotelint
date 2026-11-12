# Roadmap

Planned work after the seed, with an hour estimate per item. The total is 30 hours.

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

## 6. Configuration file, 1.5 hours

- What: a JSON file next to the documents that names their sources, the file endings that a folder search reads, the `--min-words` and `--fold-quotes` settings, so that a plain `quotelint DOC` works.
- Why: a set of documents with 10 sources needs a long command line today, and the same set has to be typed on every run. The file endings of a folder search are fixed today.

## 7. Unicode normalisation option, 1.5 hours

- What: `--nfc` to compare quote and source after Unicode NFC normalisation, using `golang.org/x/text`.
- Why: an accented letter can be stored as 1 code point or as 2. The 2 look the same, but today they count as different words.

## 8. HTML report, 2 hours

- What: `--html FILE` writes a single self-contained page with each quote, its source passage, the changed words marked, plus a filter by status.
- Why: a long text report is hard to read and hard to share. A page opens in any browser and can go to an editor who does not use a terminal.

## 9. Look-ahead for single marks, 2 hours

- What: before a single mark closes or opens a quote, look ahead in the paragraph for a better pairing.
- Why: today, a plural possessive such as `students'` ends a single-quoted passage early. A leading elision that is not on the list of common ones (such as `'cause`) opens a quote (the README section "What it does" lists both cases). Each gives a wrong quote text, which is then reported as differs or not found.

## 10. Release v0.1.0, 2 hours

- What: write a change log, tag v0.1.0, publish binaries for the 3 platforms, make the module installable with `go install github.com/jkanie/quotelint@latest`.
- Why: a tagged release with binaries lets people use the tool without a Go set-up, and it gives later releases a fixed point to compare against.

## 11. Markdown block quotes, 2 hours

- What: read each paragraph of a Markdown block quote (lines that start with `>`) as 1 quote, with the `>` marks and line-breaks taken out, and check it like a quote between marks.
- Why: a long quote is most often set as a block quote, with no quotation marks at all. Today the tool does not see it, so the longest quotes in a document are the ones left unchecked.

## 12. PDF sources, 3 hours

- What: read the text of `.pdf` files as sources with a pure-Go PDF reader, page by page, and report the page of the matched passage.
- Why: papers, standards, manuals are often PDF files. Today a PDF has to be turned into text by hand before the check can use it.

## 13. SARIF output, 2 hours

- What: `--sarif FILE` writes each quote that differs or is not found as a SARIF result with its file and line, and the source passage as the message.
- Why: GitHub code scanning reads SARIF and shows each result on its line of a pull-request, so a docs repo can check its quotes on every change.

## 14. Watch mode, 2 hours

- What: `--watch` checks again each time the document or a source file changes, and prints only the quotes whose status changed.
- Why: while a writer fixes quotes 1 by 1, starting a new run by hand after each save is slow.

## Total

30 hours.
