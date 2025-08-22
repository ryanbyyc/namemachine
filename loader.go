package namemachine

import (
	"bufio"
	"bytes"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

/**
 * fileWords maps a unix style relative path to its word slice
 * example adjectives age txt
 */
type fileWords map[string][]string // key: path "adjectives/age.txt"

/**
 * loadAllFiles walks the embedded lists tree and loads every txt file
 * paths are stored with forward slashes for consistent glob matching
 * @return fileWords map of file path to words and error
 */
func loadAllFiles() (fileWords, error) {
	out := make(fileWords)

	// walk the embedded filesystem rooted at ./lists
	err := fs.WalkDir(listsFS, "lists", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		// only process txt files
		if filepath.Ext(p) != ".txt" {
			return nil
		}

		// read file bytes from the embed fs
		b, err := listsFS.ReadFile(p)
		if err != nil {
			return err
		}

		// store with slash separators for matching
		rel := strings.TrimPrefix(p, "lists/")
		rel = filepath.ToSlash(rel)
		out[rel] = parseWordFile(b)
		return nil
	})
	return out, err
}

/**
 * parseWordFile splits a text file into trimmed non empty non comment lines
 * comment lines start with hash
 * @param b []byte file contents
 * @return []string words one per line in file order
 */
func parseWordFile(b []byte) []string {
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var words []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words = append(words, line)
	}
	return words
}

/**
 * globFilter returns file names that match any include glob and are not excluded
 * globs are matched against slash separated paths like adjectives age txt
 * @param files fileWords map of available files
 * @param includes []string include globs
 * @param excludes []string exclude globs
 * @return []string sorted list of kept file names
 */
func globFilter(files fileWords, includes, excludes []string) []string {
	// collect and sort all names for traversal
	var allNames []string
	for name := range files {
		allNames = append(allNames, name)
	}
	sort.Strings(allNames)

	// helper to test exclusion
	isExcluded := func(name string) bool {
		for _, g := range excludes {
			if ok, _ := path.Match(g, name); ok {
				return true
			}
		}
		return false
	}

	// no includes means include everything then subtract excludes
	var kept []string
	if len(includes) == 0 {
		for _, n := range allNames {
			if !isExcluded(n) {
				kept = append(kept, n)
			}
		}
		return kept
	}

	// apply include globs then de-duplicate and sort
	seen := make(map[string]struct{})
	for _, inc := range includes {
		for _, n := range allNames {
			if isExcluded(n) {
				continue
			}
			if ok, _ := path.Match(inc, n); ok {
				if _, dup := seen[n]; !dup {
					seen[n] = struct{}{}
					kept = append(kept, n)
				}
			}
		}
	}
	sort.Strings(kept)
	return kept
}

/**
 * normalizeAndFilter applies lowercasing ascii filtering length bounds and dedup
 * order of first occurrence is preserved
 * @param words []string input tokens
 * @param lowercase bool convert to lower case when true
 * @param asciiOnly bool drop tokens with non ascii bytes when true
 * @param minLen int minimum length to keep zero means no minimum
 * @param maxLen int maximum length to keep zero means no maximum
 * @return []string normalized filtered and deduplicated words
 */
func normalizeAndFilter(words []string, lowercase, asciiOnly bool, minLen, maxLen int) []string {
	dst := words[:0]
	for _, w := range words {
		if lowercase {
			w = strings.ToLower(w)
		}
		if asciiOnly && !isASCII(w) {
			continue
		}
		if minLen > 0 && len(w) < minLen {
			continue
		}
		if maxLen > 0 && len(w) > maxLen {
			continue
		}
		dst = append(dst, w)
	}

	// stable dedup keep first appearance
	seen := make(map[string]struct{}, len(dst))
	out := dst[:0]
	for _, w := range dst {
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
	}
	return out
}

/**
 * isASCII returns true when the string has only ascii bytes
 * also verifies the string is valid utf8
 * @param s string input
 * @return bool true when ascii only and valid utf8
 */
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7F {
			return false
		}
	}
	// ensure valid utf8 anyway
	return utf8.ValidString(s)
}

/**
 * mergeLists builds word lists from selected files using the requested strategy
 * can merge by directory single list or by file then optionally cross deduplicate
 * returns both the lists and their identifiers
 * @param files fileWords map of all loaded files
 * @param names []string selected file names after glob filtering
 * @param opts Options options controlling normalization strategy and dedup
 * @return [][]string merged lists and []string their ids
 */
func mergeLists(files fileWords, names []string, opts Options) (lists [][]string, ids []string) {
	switch opts.Strategy {

	case MergeByDir:
		// group by first directory component for example adjectives or names
		buckets := map[string][]string{}
		for _, n := range names {
			dir := path.Dir(n)
			buckets[dir] = append(buckets[dir], n)
		}

		// sort keys for stable output
		keys := make([]string, 0, len(buckets))
		for k := range buckets {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// accumulate words per bucket & normalize
		for _, k := range keys {
			acc := make([]string, 0, 1024)
			for _, f := range buckets[k] {
				acc = append(acc, files[f]...)
			}
			acc = normalizeAndFilter(acc, opts.Lowercase, opts.ASCIIOnly, opts.MinLen, opts.MaxLen)
			if len(acc) > 0 {
				lists = append(lists, acc)
				ids = append(ids, k)
			}
		}

	case MergeSingle:
		// flatten all selected files into one big list then normalize
		acc := make([]string, 0, 4096)
		for _, n := range names {
			acc = append(acc, files[n]...)
		}
		acc = normalizeAndFilter(acc, opts.Lowercase, opts.ASCIIOnly, opts.MinLen, opts.MaxLen)
		if len(acc) > 0 {
			lists = append(lists, acc)
			ids = append(ids, "all")
		}

	default: // MergeByFile
		// keep one list per file after normalization
		for _, n := range names {
			w := normalizeAndFilter(files[n], opts.Lowercase, opts.ASCIIOnly, opts.MinLen, opts.MaxLen)
			if len(w) > 0 {
				lists = append(lists, w)
				ids = append(ids, n)
			}
		}
	}

	// optional cross list dedup remove tokens seen in earlier lists
	if opts.CrossDedup && len(lists) > 1 {
		globSeen := make(map[string]int)
		for i := range lists {
			dst := lists[i][:0]
			for _, w := range lists[i] {
				if _, ok := globSeen[w]; ok {
					continue
				}
				globSeen[w] = 1
				dst = append(dst, w)
			}
			lists[i] = dst
		}
	}
	return lists, ids
}
