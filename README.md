# namemachine

Human-friendly name generator. **Opinionated by design:** names should read well and be easy to say and type. They are **not** intended to be cryptographically unique. If you want to push collision risk even lower, append an **optional base32 slug** at the end.

- Zero-alloc hot path via a caller-provided buffer
- Thread-safe RNG
- Dependency-free (standard library only)
- Flexible list selection and merging strategies
- Glob-based inclusion and exclusion
- Tests to keep the wordlists clean and fast

[![Go Reference](https://pkg.go.dev/badge/github.com/ryanbyyc/namemachine.svg)](https://pkg.go.dev/github.com/ryanbyyc/namemachine)
[![Go Report Card](https://goreportcard.com/badge/github.com/ryanbyyc/namemachine)](https://goreportcard.com/report/github.com/ryanbyyc/namemachine)
[![CodeQL](https://github.com/ryanbyyc/namemachine/actions/workflows/codeql.yml/badge.svg)](https://github.com/ryanbyyc/namemachine/actions/workflows/codeql.yml)
[![CI](https://img.shields.io/github/actions/workflow/status/ryanbyyc/namemachine/ci.yml?branch=main&label=ci)](https://github.com/ryanbyyc/namemachine/actions/workflows/ci.yml)
[![Lint](https://img.shields.io/github/actions/workflow/status/ryanbyyc/namemachine/lint.yml?branch=main&label=lint)](https://github.com/ryanbyyc/namemachine/actions/workflows/lint.yml)
[![Coverage](https://img.shields.io/codecov/c/github/ryanbyyc/namemachine?branch=main&label=coverage)](https://app.codecov.io/gh/ryanbyyc/namemachine)
[![Go Version](https://img.shields.io/github/go-mod/go-version/ryanbyyc/namemachine?label=go)](https://github.com/ryanbyyc/namemachine/blob/main/go.mod)
[![License](https://img.shields.io/github/license/ryanbyyc/namemachine)](./LICENSE)
[![Release](https://img.shields.io/github/v/release/ryanbyyc/namemachine?display_name=tag&sort=semver)](https://github.com/ryanbyyc/namemachine/releases)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](pulls)

---

## What this does

`namemachine` builds names by sampling words from embedded text lists. You choose how lists are combined:

- **By directory**: each folder under `lists/` becomes one list (e.g., adjectives, nouns)
- **By file**: each text file is its own list
- **Single flattened list**: all selected files become one giant list

You control the delimiter (default `_`), word count (exact or range), optional **slug** length, and list selection through include/exclude globs. No env variables, no external deps.

---

## Install

```bash
go get github.com/ryanbyyc/namemachine
```

Minimum Go: any reasonably recent version that supports `go:embed` (Go 1.16+).

---

## Usage

See `example/main.go` for a full runnable tour:

```go
g, _ := namemachine.New(namemachine.Options{
  IncludeGlobs: []string{"**/*.txt"},  // select embedded lists
  Strategy:     namemachine.MergeByDir,
  Words:        2, // or MinWords/MaxWords for a range
  Delimiter:    '_',
  SlugLength:   6, // optional collision-buster
  Seed:         42, // deterministic for tests and demos
})

// Convenience API (returns string, will allocate)
name := g.Generate(0)

// Zero-alloc API (you own the buffer)
buf := make([]byte, 0, 64)
buf = g.GenerateInto(buf[:0], 0)
fmt.Println(string(buf))
```

### Example output

```
▶ two-word (all dirs) + slug
  - ringing_selfservice_m43kbx
  - mild_crossfunctional_laff7u
  - rugged_partnerships_6cxh4c
  - knotted_internalorganic_ok5fqf
  - humongous_sodales_acqme5

▶ three-word (all dirs), '-' delimiter
  - pureed-premier-insulation
  - rosy-yieldfarming-pilot
  - dense-dogecoin-ancestor
  - clever-cursus-wrap
  - weary-tortor-loggia

▶ single-word (flatten all files)
  - radial
  - pita
  - upload
  - pagoda
  - wicker

▶ 1..3 words (exclude ipsum/**)
  - stringy_shopping_dribble
  - pragmatic_soldier
  - watery
  - tolerant_restaurant
  - tapered
  - universal_backlot_proclaim
  - organic_soundboard_amplify
  - crabby_envelope_guard

▶ two-word (zero-alloc buffer API)
  - vaulted_tempor
  - candied_automoderator
  - ringing_ape
  - coffee_tristique
  - chromatic_tellus
  - buttery_microbrew

▶ concurrency demo (3 goroutines × 4 names)
  [w3] chalky_leadership
  [w3] constant_potentialities
  [w3] isometric_outdoorsy
  [w3] blocky_heirloom
  [w2] objective_sint
  [w2] extended_streamline
  [w2] teal_methodsofempowerment
  [w2] ebony_microservices
  [w1] chartreuse_butcher
  [w1] faithful_startup
  [w1] amortized_applications
  [w1] organic_predominate
```

---

## Options overview

```go
type Options struct {
  // Selection and merging
  IncludeGlobs []string // e.g. []{"**/*.txt"}, or "ipsum/**", "crypto/*.txt"
  ExcludeGlobs []string
  Strategy     MergeStrategy // MergeByDir, MergeByFile, MergeSingle

  // Word count controls
  Words    int // exact, if > 0
  MinWords int // inclusive
  MaxWords int // inclusive (used when Words == 0)

  // Formatting and collision control
  Delimiter  byte // default '_'
  SlugLength int  // 0 disables slug

  // Normalization and filters
  Lowercase  bool
  ASCIIOnly  bool
  MinLen     int
  MaxLen     int
  CrossDedup bool // remove dup words across lists after merging

  // Reproducibility
  Seed int64 // if 0, seeded from crypto/rand
}
```

---

## Adding words

Just add plain text files under `lists/`:

```
lists/
  adjectives/
    colors.txt
  ipsum/
    corporate.txt
  crypto/
    crypto.txt
  hipster/
    hipster.txt
```

Rules we enforce in tests:

- One token per line
- No comments except lines starting with `#`
- The corpus is **heavily sanitized** to be alphanumeric friendly and lower friction for naming
- We maintain additional curated lists on top of the excellent upstream source

You can select subsets with globs:

```go
// Use everything except the lorem-ipsum subtree
IncludeGlobs: []string{"**/*.txt"},
ExcludeGlobs: []string{"ipsum/**"},
Strategy:     namemachine.MergeByDir,
```

Or flatten everything:

```go
IncludeGlobs: []string{"**/*.txt"},
Strategy:     namemachine.MergeSingle,
```

---

## Testing

Run all tests:

```bash
go test ./... -v
```

Sample:

```
$ go test ./... -v
=== RUN   TestTotalCombinations_AllLists_TwoAndThreeWords
    combo_test.go:98: lists discovered: 4
    combo_test.go:100:   list[0] size = 972
    combo_test.go:100:   list[1] size = 1188
    combo_test.go:100:   list[2] size = 4810
    combo_test.go:100:   list[3] size = 913
    combo_test.go:102: 2-word ordered (distinct lists) total = 35,815,892
    combo_test.go:104: 3-word ordered (distinct lists) total = 96,565,553,568
--- PASS: TestTotalCombinations_AllLists_TwoAndThreeWords (0.00s)
=== RUN   TestEmbeddedFilesPresentAndNonEmpty
--- PASS: TestEmbeddedFilesPresentAndNonEmpty (0.00s)
=== RUN   TestNoDuplicatesWithinEachFile
--- PASS: TestNoDuplicatesWithinEachFile (0.00s)
=== RUN   TestGlobSelectionCounts
--- PASS: TestGlobSelectionCounts (0.00s)
=== RUN   TestAllListsCombinationsReport
    generator_test.go:186: k=1 words -> 972 combinations
    generator_test.go:186: k=2 words -> 1,154,736 combinations
    generator_test.go:186: k=3 words -> 5,554,280,160 combinations
--- PASS: TestAllListsCombinationsReport (0.00s)
=== RUN   TestDelimiterAndSlugAndOverride
--- PASS: TestDelimiterAndSlugAndOverride (0.00s)
=== RUN   TestTotalWordsAllFiles
    generator_test.go:276: Total words across all files: 10002
--- PASS: TestTotalWordsAllFiles (0.00s)
=== RUN   TestUniqueWordsAllFiles
    generator_test.go:296: Unique words across all files: 7238
--- PASS: TestUniqueWordsAllFiles (0.00s)
=== RUN   TestAllWords_AreAlnumOnly
--- PASS: TestAllWords_AreAlnumOnly (0.00s)
=== RUN   TestAllWords_LowercaseOnly
    sanitize_alpha_test.go:79: Unskip to enforce lowercase-only vocabulary
--- SKIP: TestAllWords_LowercaseOnly (0.00s)
PASS
ok      github.com/ryanbyyc/namemachine 0.022s
```

### Benchmarks

```bash
go test -bench=. -benchmem
```

```
goos: linux
goarch: amd64
pkg: github.com/ryanbyyc/namemachine
cpu: 13th Gen Intel(R) Core(TM) i9-13900KF

BenchmarkGenerate2Words-32              14854232                81.25 ns/op           35 B/op          1 allocs/op
BenchmarkGenerate3WordsWithSlug-32       3218545               369.7 ns/op            98 B/op          2 allocs/op
BenchmarkGenerateInto_ZeroAllocs-32     22164978                53.99 ns/op            0 B/op          0 allocs/op
```

- `GenerateInto` is the **zero-alloc** path when you provide a reusable buffer
- `Generate` is the convenience API that returns a string and allocates

---

## Attribution

This project builds on the excellent upstream wordlists at **imsky/wordlists**. The original files were **heavily sanitized and expanded** for naming use. Upstream license is included under `/lists`.

- Source: https://github.com/imsky/wordlists

---

## License

MIT for this project. See `LICENSE` for details. Upstream license is preserved in `/lists`.
