package source

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DocExts are the file endings that a document folder is searched for.
var DocExts = []string{".md", ".markdown", ".txt", ".text"}

// SourceExts are the file endings that a source folder is searched for.
var SourceExts = []string{".md", ".markdown", ".txt", ".text", ".html", ".htm"}

// Expand turns each path into a list of files. A file is kept as it is,
// whatever its ending. A folder is walked recursively for files with one
// of exts (any case), skipping every entry whose name starts with a dot
// (such as .git). Each folder's files come out sorted, and a file named
// twice is listed once. A folder that holds no matching file is an error,
// so that a typo in a path never passes as "no quotes to check".
func Expand(paths []string, exts []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		k := key(p)
		if !seen[k] {
			seen[k] = true
			out = append(out, p)
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			add(p)
			continue
		}
		var found []string
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != p && strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() && hasExt(path, exts) {
				found = append(found, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return nil, fmt.Errorf("%s: no file ending in %s", p, strings.Join(exts, ", "))
		}
		sort.Strings(found)
		for _, f := range found {
			add(f)
		}
	}
	return out, nil
}

// Without returns paths minus every path that names the same file as one
// of drop, so that a document is never checked against itself.
func Without(paths, drop []string) []string {
	skip := map[string]bool{}
	for _, d := range drop {
		skip[key(d)] = true
	}
	var out []string
	for _, p := range paths {
		if !skip[key(p)] {
			out = append(out, p)
		}
	}
	return out
}

func hasExt(path string, exts []string) bool {
	e := strings.ToLower(filepath.Ext(path))
	for _, x := range exts {
		if e == x {
			return true
		}
	}
	return false
}

// key gives 1 spelling per file, so that ./a.md and a.md count as 1.
func key(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return filepath.Clean(p)
}
