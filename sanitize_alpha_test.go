package namemachine

import (
	"testing"
)

/**
 * isAlnumASCII reports whether the input contains only ascii digits or letters
 * digits zero to nine and letters a to z or A to Z
 * @param s string input token
 * @return bool true when token is ascii alphanumeric only
 */
func isAlnumASCII(s string) bool {
	// scan byte by byte and reject on first non alnum
	for i := 0; i < len(s); i++ {
		b := s[i]
		if !((b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')) {
			return false
		}
	}
	return true
}

/**
 * TestAllWords_AreAlnumOnly enforces that every embedded word is ascii alphanumeric
 * catches spaces punctuation emoji and any other non alnum content
 * logs up to twenty offenders for quick triage then fails hard
 * @param t *testing.T test harness
 * @return void
 */
func TestAllWords_AreAlnumOnly(t *testing.T) {
	files, err := loadAllFiles()
	if err != nil {
		t.Fatalf("loadAllFiles: %v", err)
	}

	// track offenders with file and line for clear action
	type offender struct {
		file string
		line int
		word string
	}
	var bad []offender

	// walk all tokens and collect non alnum items
	for file, words := range files {
		for i, w := range words {
			if !isAlnumASCII(w) {
				bad = append(bad, offender{file: file, line: i + 1, word: w})
			}
		}
	}

	// log a small sample then fail with the total count
	if len(bad) > 0 {
		limit := len(bad)
		if limit > 20 {
			limit = 20
		}
		for i := 0; i < limit; i++ {
			o := bad[i]
			t.Logf("non-alnum: %s:%d %q", o.file, o.line, o.word)
		}
		if len(bad) > limit {
			t.Logf("...and %d more", len(bad)-limit)
		}
		t.Fatalf("found %d non-alphanumeric words in embedded lists", len(bad))
	}
}

/**
 * TestAllWords_LowercaseOnly optionally enforces lowercase only vocabulary
 * skipped by default; enable when you want strict lowercase only lists
 * scans for any ascii uppercase letter and reports a short sample
 * @param t *testing.T test harness
 * @return void
 */
func TestAllWords_LowercaseOnly(t *testing.T) {
	t.Skip("Unskip to enforce lowercase-only vocabulary")

	files, err := loadAllFiles()
	if err != nil {
		t.Fatalf("loadAllFiles: %v", err)
	}

	// track offenders with file and line for clear action
	type offender struct {
		file string
		line int
		word string
	}
	var bad []offender

	// walk all tokens and flag any uppercase ascii
	for file, words := range files {
		for i, w := range words {
			upper := false
			for j := 0; j < len(w); j++ {
				if w[j] >= 'A' && w[j] <= 'Z' {
					upper = true
					break
				}
			}
			if upper {
				bad = append(bad, offender{file: file, line: i + 1, word: w})
			}
		}
	}

	// log a small sample then fail with the total count
	if len(bad) > 0 {
		limit := min(len(bad), 20)
		for i := range limit {
			o := bad[i]
			t.Logf("uppercase: %s:%d %q", o.file, o.line, o.word)
		}
		if len(bad) > limit {
			t.Logf("...and %d more", len(bad)-limit)
		}
		t.Fatalf("found %d uppercase words", len(bad))
	}
}
