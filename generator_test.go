package namemachine

import (
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
)

/**
 * withCommas returns a decimal string with commas every three digits for readable logs
 * @param n int input value
 * @return string decimal with comma separators
 */
func withCommas(n int) string {
	s := strconv.FormatInt(int64(n), 10)

	// walk from right to left and splice in commas every three digits
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

/**
 * groupByDir collects directory keys from file paths and returns them sorted
 * directories come from path Dir so nested paths collapse to their parent segment
 * @param files fileWords map keyed by relative file path
 * @return []string sorted directory names
 */
func groupByDir(files fileWords) []string {
	dirs := map[string]struct{}{}
	for name := range files {
		dir := path.Dir(name)
		dirs[dir] = struct{}{}
	}
	out := make([]string, 0, len(dirs))
	for d := range dirs {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

/**
 * chooseDirs returns up to n directory buckets in deterministic order
 * useful for tests that need a small stable subset
 * @param files fileWords map of embedded lists
 * @param n int maximum number of directories to return
 * @return []string first n directories or nil if none
 */
func chooseDirs(files fileWords, n int) []string {
	all := groupByDir(files)
	if len(all) == 0 {
		return nil
	}
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

/**
 * TestEmbeddedFilesPresentAndNonEmpty verifies that embedded lists exist and have no empty lines
 * trims whitespace and fails if any blank tokens are found
 * @param t *testing.T test harness
 * @return void
 */
func TestEmbeddedFilesPresentAndNonEmpty(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatalf("loadAllFiles: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no embedded word files found under lists/")
	}
	for fname, words := range files {
		if len(words) == 0 {
			t.Fatalf("file %q parsed but yielded 0 words", fname)
		}
		for i, w := range words {
			if strings.TrimSpace(w) == "" {
				t.Fatalf("empty/whitespace word in %s at line %d", fname, i)
			}
		}
	}
}

/**
 * TestNoDuplicatesWithinEachFile enforces that a single list file has no duplicate tokens
 * cross file duplicates are allowed this test is per file only
 * @param t *testing.T test harness
 * @return void
 */
func TestNoDuplicatesWithinEachFile(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatalf("loadAllFiles: %v", err)
	}
	for fname, arr := range files {
		seen := make(map[string]int, len(arr))
		for i, w := range arr {
			if j, ok := seen[w]; ok {
				t.Fatalf("duplicate within %s: %q at lines %d and %d", fname, w, j, i)
			}
			seen[w] = i
		}
	}
}

/**
 * TestGlobSelectionCounts proves that include and exclude globs change the selected set
 * helps catch embed or walk regressions quickly
 * @param t *testing.T test harness
 * @return void
 */
func TestGlobSelectionCounts(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatalf("loadAllFiles: %v", err)
	}

	allNames := globFilter(files, []string{"**/*.txt"}, nil)
	if len(allNames) == 0 {
		t.Fatal("**/*.txt matched 0 files; embedding likely broken")
	}

	// exclude a known subtree from the source repo and ensure the count drops
	excluded := globFilter(files, []string{"**/*.txt"}, []string{"ipsum/**"})
	if len(excluded) >= len(allNames) {
		t.Fatalf("exclude globs did not reduce selection: all=%d excluded=%d", len(allNames), len(excluded))
	}
}

/**
 * TestAllListsCombinationsReport builds lists by directory and reports combinations for k words
 * uses MergeByDir so test remains stable as files are added under a directory
 * @param t *testing.T test harness
 * @return void
 */
func TestAllListsCombinationsReport(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatalf("loadAllFiles: %v", err)
	}

	// pick a small deterministic subset so logs stay readable
	dirs := chooseDirs(files, 3)
	if len(dirs) == 0 {
		t.Fatal("no directories discovered under lists/")
	}

	// convert directory choices into include globs
	var globs []string
	for _, d := range dirs {
		// handle top level bucket if present
		if d == "." {
			globs = append(globs, "*.txt")
		} else {
			globs = append(globs, d+"/**")
		}
	}

	g, err := New(Options{
		IncludeGlobs: globs,
		Strategy:     MergeByDir, // each chosen directory becomes one list
		Words:        2,
		Delimiter:    '_',
		Seed:         1,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(g.lists) == 0 {
		t.Fatalf("no lists built from globs: %v", globs)
	}

	// log combinations for k up to a small cap
	maxK := min(len(g.lists), 5)
	for k := 1; k <= maxK; k++ {
		total := combinationsForK(g.lists, k)
		if total <= 0 {
			t.Fatalf("expected >0 combinations for k=%d", k)
		}
		t.Logf("k=%d words -> %s combinations", k, withCommas(total))
	}

	// basic guard for k equal two
	if len(g.lists) >= 2 {
		if combinationsForK(g.lists, 2) <= 0 {
			t.Fatal("expected >0 combinations for k=2")
		}
	}
}

/**
 * TestDelimiterAndSlugAndOverride checks delimiter slug length and per call word count override
 * uses a flattened list so assertions are simple and direct
 * @param t *testing.T test harness
 * @return void
 */
func TestDelimiterAndSlugAndOverride(t *testing.T) {
	// build a generator that flattens all files to a single list
	g, err := New(Options{
		IncludeGlobs: []string{"**/*.txt"},
		Strategy:     MergeSingle,
		Words:        2,
		Delimiter:    '-',
		SlugLength:   8,
		Seed:         42,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// generate with override to one word and ensure the slug is present and sized
	name := g.Generate(1)
	parts := strings.Split(name, "-")
	if len(parts) != 2 {
		t.Fatalf("expected 1 delimiter (word-slug), got %q", name)
	}
	if got := len(parts[1]); got != 8 {
		t.Fatalf("expected slug length 8, got %d (%q)", got, parts[1])
	}
	if parts[0] == "" {
		t.Fatal("word part is empty")
	}
}

/**
 * combinationsForK returns the total combinations for exactly k words
 * cycles through lists using modulo to mirror generator behavior
 * guards against integer overflow by clamping to max int
 * @param lists [][]string input lists
 * @param k int target word count
 * @return int total combinations for exactly k words
 */
func combinationsForK(lists [][]string, k int) int {
	if k <= 0 || len(lists) == 0 {
		return 0
	}
	p := 1
	maxInt := int(^uint(0) >> 1)

	// multiply lengths while cycling over lists and bail if any list is empty
	for i := range k {
		size := len(lists[i%len(lists)])
		if size == 0 {
			return 0
		}
		// overflow guard clamp to max int
		if p > maxInt/size {
			return maxInt
		}
		p *= size
	}
	return p
}

/**
 * TestTotalWordsAllFiles logs the total number of tokens across all embedded files
 * a quick view of corpus size for sanity checks
 * @param t *testing.T test harness
 * @return void
 */
func TestTotalWordsAllFiles(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, arr := range files {
		total += len(arr)
	}
	t.Logf("Total words across all files: %d", total)
}

/**
 * TestUniqueWordsAllFiles logs the number of unique tokens across all embedded files
 * counts after exact match de duplication only
 * @param t *testing.T test harness
 * @return void
 */
func TestUniqueWordsAllFiles(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{})
	for _, arr := range files {
		for _, w := range arr {
			seen[w] = struct{}{}
		}
	}
	t.Logf("Unique words across all files: %d", len(seen))
}
