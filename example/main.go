package main

import (
	"fmt"
	"sync"

	"github.com/ryanbyyc/namemachine"
)

func main() {
	// Scenario 1: Docker-ish, two words from ALL directory buckets, with a short slug
	demo("two-word (all dirs) + slug",
		namemachine.Options{
			IncludeGlobs: []string{"**/*.txt"},   // include everything embedded
			Strategy:     namemachine.MergeByDir, // each directory = one list
			Words:        2,                      // exactly 2 words
			Delimiter:    '_',
			SlugLength:   6,  // optional collision-buster
			Seed:         42, // deterministic for demo
		},
		5, // samples
		0, // no per-call override
	)

	// Scenario 2: Three-word name across all directories, custom delimiter
	demo("three-word (all dirs), '-' delimiter",
		namemachine.Options{
			IncludeGlobs: []string{"**/*.txt"},
			Strategy:     namemachine.MergeByDir,
			Words:        3,
			Delimiter:    '-',
			Seed:         42,
		},
		5, 0,
	)

	// Scenario 3: Single-word generator, flatten everything into one list
	demo("single-word (flatten all files)",
		namemachine.Options{
			IncludeGlobs: []string{"**/*.txt"},
			Strategy:     namemachine.MergeSingle, // all words in one big list
			Words:        1,
			Seed:         42,
		},
		5, 0,
	)

	// Scenario 4: Random 1..3 words, exclude filler (e.g., lorem ipsum folder), no slug
	demo("1..3 words (exclude ipsum/**)",
		namemachine.Options{
			IncludeGlobs: []string{"**/*.txt"},
			ExcludeGlobs: []string{"ipsum/**"}, // omit a whole folder
			Strategy:     namemachine.MergeByDir,
			MinWords:     1,
			MaxWords:     3,
			Delimiter:    '_',
			Seed:         123,
		},
		8, 0,
	)

	// Scenario 5: Zero-alloc API using GenerateInto (caller-managed buffer)
	demoInto("two-word (zero-alloc buffer API)",
		namemachine.Options{
			IncludeGlobs: []string{"**/*.txt"},
			Strategy:     namemachine.MergeByDir,
			Words:        2,
			Delimiter:    '_',
			Seed:         7,
		},
		6, 0,
	)

	// Scenario 6: Thread-safety demo (concurrent generation)
	concurrencyDemo(
		namemachine.Options{
			IncludeGlobs: []string{"**/*.txt"},
			Strategy:     namemachine.MergeByDir,
			Words:        2,
			Delimiter:    '_',
			Seed:         99,
		},
		3, // goroutines
		4, // names per goroutine
	)
}

func demo(title string, opts namemachine.Options, samples, overrideWords int) {
	g, err := namemachine.New(opts)
	if err != nil {
		fmt.Printf("✖ %s: %v\n", title, err)
		return
	}
	fmt.Printf("\n▶ %s\n", title)
	for i := 0; i < samples; i++ {
		// If overrideWords > 0, it overrides the generator's Words/Min/Max for this call
		fmt.Printf("  - %s\n", g.Generate(overrideWords))
	}
}

func demoInto(title string, opts namemachine.Options, samples, overrideWords int) {
	g, err := namemachine.New(opts)
	if err != nil {
		fmt.Printf("✖ %s: %v\n", title, err)
		return
	}
	fmt.Printf("\n▶ %s\n", title)

	// Reuse a buffer to achieve zero allocations in hot paths.
	buf := make([]byte, 0, 64)
	for i := 0; i < samples; i++ {
		buf = g.GenerateInto(buf[:0], overrideWords)
		fmt.Printf("  - %s\n", string(buf))
	}
}

func concurrencyDemo(opts namemachine.Options, goroutines, perGoroutine int) {
	g, err := namemachine.New(opts)
	if err != nil {
		fmt.Printf("✖ concurrency demo: %v\n", err)
		return
	}
	fmt.Printf("\n▶ concurrency demo (%d goroutines × %d names)\n", goroutines, perGoroutine)

	var wg sync.WaitGroup
	for t := 0; t < goroutines; t++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				fmt.Printf("  [w%d] %s\n", id, g.Generate(0))
			}
		}(t + 1)
	}
	wg.Wait()
}
