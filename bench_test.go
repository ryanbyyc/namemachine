package namemachine

import (
	"path"
	"sort"
	"testing"
)

/**
 * pickDirs collects directory buckets from embedded file paths and returns up to "want" of them
 * Results are sorted for determinism so benchmarks are stable across runs
 * @param files fileWords map keyed by relative file path
 * @param want maximum number of directories to return
 * @return []string sorted directory names
 */
func pickDirs(files fileWords, want int) []string {
	dirs := map[string]struct{}{}
	for name := range files {
		d := path.Dir(name)
		dirs[d] = struct{}{}
	}
	out := make([]string, 0, len(dirs))
	for d := range dirs {
		out = append(out, d)
	}
	sort.Strings(out)
	if want > len(out) {
		want = len(out)
	}
	return out[:want]
}

/**
 * setupTwoListGenerator builds a Generator tailored for benchmarks
 * It discovers lists dynamically, prefers two directory buckets, and falls back gracefully
 * Zero curation, zero surprises, very repeatable!
 * @param tb testing.TB so we can fail fast in setup
 * @return *Generator ready for hot path benchmarks
 */
func setupTwoListGenerator(tb testing.TB) *Generator {
	tb.Helper()

	files, err := loadAllFiles()
	if err != nil {
		tb.Fatalf("loadAllFiles: %v", err)
	}

	// deterministically choose up to two directory buckets
	dirs := pickDirs(files, 2)

	// turn the chosen directories into IncludeGlobs
	var globs []string
	switch len(dirs) {
	case 0:
		// no dirs; take everything!
		globs = []string{"**/*.txt"}
	case 1:
		if dirs[0] == "." {
			globs = []string{"*.txt"}
		} else {
			globs = []string{dirs[0] + "/**"}
		}
	default:
		for _, d := range dirs {
			if d == "." {
				globs = append(globs, "*.txt")
			} else {
				globs = append(globs, d+"/**")
			}
		}
	}

	// prefer two lists via MergeByDir; fallback to MergeSingle if we ended up with < 2
	strategy := MergeByDir
	if len(dirs) < 2 {
		strategy = MergeSingle
	}

	// fixed seed for stable benchmark names
	g, err := New(Options{
		IncludeGlobs: globs,
		Strategy:     strategy,
		Words:        2,
		Delimiter:    '_',
		Seed:         42,
	})
	if err != nil {
		tb.Fatalf("New: %v", err)
	}

	return g
}

/**
 * BenchmarkGenerate2Words measures the convenience API that returns string
 * Expect one allocation for the string copy, and steady nanoseconds per op
 * @param b *testing.B benchmark harness
 */
func BenchmarkGenerate2Words(b *testing.B) {
	g := setupTwoListGenerator(b)

	b.ReportAllocs() // show allocs so regressions pop instantly!
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// returns string (1 alloc)
		_ = g.Generate(0)
	}
}

/**
 * BenchmarkGenerate3WordsWithSlug exercises a heavier path
 * Three words plus an 8 char slug gives a realistic collision resistant setup
 * @param b *testing.B benchmark harness
 */
func BenchmarkGenerate3WordsWithSlug(b *testing.B) {
	g := setupTwoListGenerator(b)
	g.wordsExact = 3 // bump to 3 words
	g.slugLen = 8    // add a short slug
	b.ReportAllocs() // watch the single alloc for string return
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Generate(0) // returns string (1 alloc)
	}
}

/**
 * BenchmarkGenerateInto_ZeroAllocs validates the zero allocation API
 * Caller supplies and reuses the buffer, so we can prove 0 B/op 0 allocs/op
 * @param b *testing.B benchmark harness
 */
func BenchmarkGenerateInto_ZeroAllocs(b *testing.B) {
	g := setupTwoListGenerator(b)
	dst := make([]byte, 0, 64) // reuse buffer across iterations for zero allocs!

	b.ReportAllocs()
	b.ResetTimer() // discount setup

	for i := 0; i < b.N; i++ {
		dst = g.GenerateInto(dst[:0], 0) // zero allocs if cap(dst) is enough
		if len(dst) == 0 {
			b.Fatal("empty") // guard so the compiler can't elide work
		}
	}
}
