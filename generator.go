package namemachine

import (
	"fmt"
	"io"
	"math/rand"
	"sync"
)

/**
 * Generator produces names from embedded lists with optional slug and custom delimiter
 * thread safe random source guarded by mutex
 * supports zero allocation generation when caller provides a buffer
 */
type Generator struct {
	lists [][]string // in order user requested
	delim byte

	wordsExact int
	minWords   int
	maxWords   int

	slugLen int

	rngMu sync.Mutex
	rng   *rand.Rand
}

/**
 * New creates a Generator and performs one time loading filtering and merging
 * expensive setup happens once here
 * @param opts Options configuration for list selection normalization and behavior
 * @return *Generator instance or error
 */
func New(opts Options) (*Generator, error) {
	opts.norm()

	files, err := loadAllFiles()
	if err != nil {
		return nil, err
	}

	// select files using include and exclude globs
	selected := globFilter(files, opts.IncludeGlobs, opts.ExcludeGlobs)

	// merge selected files into lists based on strategy
	lists, _ := mergeLists(files, selected, opts)

	// require at least one list to proceed
	if len(lists) == 0 {
		return nil, fmt.Errorf("no lists selected (IncludeGlobs/ExcludeGlobs matched zero files)")
	}

	// seed a private rng for this generator
	r := rand.New(rand.NewSource(opts.Seed))
	return &Generator{
		lists:      lists,
		delim:      opts.Delimiter,
		wordsExact: opts.Words,
		minWords:   opts.MinWords,
		maxWords:   opts.MaxWords,
		slugLen:    opts.SlugLength,
		rng:        r,
	}, nil
}

/**
 * GenerateInto writes a name into dst and returns the used slice
 * zero heap allocations when dst capacity is sufficient
 * if nWords is greater than zero it overrides the generator word count settings
 * @param dst []byte destination buffer provided by the caller
 * @param nWords int optional override for number of words
 * @return []byte slice containing the generated name
 */
func (g *Generator) GenerateInto(dst []byte, nWords int) []byte {
	if len(g.lists) == 0 {
		return dst[:0]
	}

	// decide word count
	count := nWords
	if count <= 0 {
		if g.wordsExact > 0 {
			count = g.wordsExact
		} else {
			count = g.randWordCount()
		}
	}
	if count <= 0 {
		count = 1
	}

	// compute final length to size buffer correctly
	totalLen := 0
	for i := 0; i < count; i++ {
		list := g.lists[i%len(g.lists)]

		// one rng call per word
		g.rngMu.Lock()
		idx := g.rng.Intn(len(list))
		g.rngMu.Unlock()

		totalLen += len(list[idx])
	}
	if count > 1 {
		totalLen += count - 1 // delimiters between words
	}
	if g.slugLen > 0 {
		totalLen += 1 + g.slugLen // one delimiter plus slug bytes
	}

	// ensure capacity without allocating if caller provided enough space
	if cap(dst) < totalLen {
		// fall back to allocation only if caller did not give enough space
		dst = make([]byte, 0, totalLen)
	} else {
		dst = dst[:0]
	}

	// build words into dst
	for i := 0; i < count; i++ {
		if i > 0 {
			dst = append(dst, g.delim)
		}
		list := g.lists[i%len(g.lists)]

		// choose a word using the rng
		g.rngMu.Lock()
		w := list[g.rng.Intn(len(list))]
		g.rngMu.Unlock()

		dst = append(dst, w...)
	}

	// append slug directly into dst no temp slice
	if g.slugLen > 0 {
		dst = append(dst, g.delim)
		dst = randomSlugInto(dst, g.slugLen)
	}
	return dst
}

/**
 * Generate is a convenience wrapper that returns a string
 * this allocates for the byte slice and for the string copy
 * @param nWords int optional override for number of words
 * @return string generated name
 */
func (g *Generator) Generate(nWords int) string {
	b := g.GenerateInto(nil, nWords) // will allocate exactly once for byte slice
	return string(b)                 // second allocation string copy
}

/**
 * randWordCount picks a word count using min and max bounds
 * returns an (old) docker like default of two when bounds are not set
 * @return int chosen word count
 */
func (g *Generator) randWordCount() int {
	if g.minWords <= 0 && g.maxWords <= 0 {
		return 2
	}
	min := g.minWords
	max := g.maxWords
	if min <= 0 {
		min = 1
	}
	if max < min {
		max = min
	}
	g.rngMu.Lock()
	n := g.rng.Intn(max-min+1) + min
	g.rngMu.Unlock()
	return n
}

/**
 * WriteTo writes a generated name to an io Writer
 * uses a small stack buffer and the zero allocation path inside GenerateInto
 * @param w io.Writer destination writer
 * @param nWords int optional override for number of words
 * @return int number of bytes written and error if any
 */
func (g *Generator) WriteTo(w io.Writer, nWords int) (int, error) {
	buf := make([]byte, 0, 64) // small stack buffer
	buf = g.GenerateInto(buf, nWords)
	return w.Write(buf) // writer may allocate but this function does not
}
