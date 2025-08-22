package namemachine

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"time"
)

/**
 * MergeStrategy selects how files are merged into lists
 */
type MergeStrategy int

const (
	MergeByFile MergeStrategy = iota // each file becomes one list
	MergeByDir                       // each directory becomes one list
	MergeSingle                      // all selected files become one list
)

/**
 * Options controls selection normalization and generation behavior
 * fields are optional unless noted and sensible defaults are applied in norm
 */
type Options struct {
	// ListNames allows legacy selection by logical list name
	// example adjectives animals
	ListNames []string

	// Word count behavior
	// Words is exact number of words when greater than zero
	// MinWords and MaxWords define an inclusive range used when Words is zero
	Words    int
	MinWords int
	MaxWords int

	// Delimiter placed between words and before slug when present
	// default underscore (_)
	Delimiter byte

	// Per list include and exclude filters
	// keys are list identifiers values are words to include or exclude
	Include map[string][]string
	Exclude map[string][]string

	// SlugLength controls random slug size
	// zero disables slug
	SlugLength int

	// Seed for deterministic output in tests
	// when zero a secure seed is drawn from crypto rand
	Seed int64

	// Glob selection
	// IncludeGlobs selects files to include
	// ExcludeGlobs removes files from consideration
	IncludeGlobs []string
	ExcludeGlobs []string

	// Merge strategy for building lists
	Strategy MergeStrategy

	// Normalization and filters
	// Lowercase converts tokens to lower case
	// ASCIIOnly drops tokens with non ascii bytes
	// MinLen and MaxLen keep tokens within bounds zero means no bound
	// CrossDedup removes duplicate tokens across lists after they are built
	Lowercase  bool
	ASCIIOnly  bool
	MinLen     int
	MaxLen     int
	CrossDedup bool
}

/**
 * norm applies default values to options in place
 * sets delimiter when empty and seeds the rng when seed is zero
 * @param o *Options options to normalize
 * @return void
 */
func (o *Options) norm() {
	if o.Delimiter == 0 {
		o.Delimiter = '_'
	}
	if o.Seed == 0 {
		var seed [8]byte
		if _, err := cryptoRand.Read(seed[:]); err != nil {
			o.Seed = time.Now().UnixNano()
		} else {
			o.Seed = int64(binary.LittleEndian.Uint64(seed[:]))
		}
	}
}
