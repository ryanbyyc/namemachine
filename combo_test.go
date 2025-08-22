package namemachine

import (
	"math/big"
	"strings"
	"testing"
)

/**
 * withCommasBig formats a big integer with thousands separators for readable logs
 * Resugreat for very large totals in tests and benchmarks
 * @param x *big.Int input value
 * @return string decimal with commas every three digits
 */
func withCommasBig(x *big.Int) string {
	s := x.String()

	// fast path for small values return as is
	if len(s) <= 3 {
		return s
	}

	// pre size the builder to reduce growth and keep it tight
	var b strings.Builder
	b.Grow(len(s) + len(s)/3)

	// compute the head width so the first group aligns before groups of three
	head := len(s) % 3
	if head == 0 {
		head = 3
	}

	// write the head then write groups of three with commas between groups
	b.WriteString(s[:head])
	for i := head; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

/**
 * TestTotalCombinations_AllLists_TwoAndThreeWords reports ordered combination totals across all directory buckets
 * two word count is ordered pairs from distinct lists
 * three word count is ordered triples from distinct lists
 * uses big integers to avoid overflow so we can scale forever
 * @param t *testing.T test harness
 * @return void
 */

func TestTotalCombinations_AllLists_TwoAndThreeWords(t *testing.T) {
	// build lists from all text files grouped by directory buckets
	g, err := New(Options{
		IncludeGlobs: []string{"**/*.txt"}, // pull everything
		Strategy:     MergeByDir,           // one list per directory bucket
		Delimiter:    '_',
		Seed:         1,
	})

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	L := len(g.lists)
	if L == 0 {
		t.Fatal("no lists discovered under lists/**")
	}

	//  power sums s1 s2 s3 using big integers to keep totals exact
	// s1 is sum of list sizes
	// s2 is sum of squares of list sizes
	// s3 is sum of cubes of list sizes
	var S1, S2, S3 big.Int
	for _, lst := range g.lists {
		ai := big.NewInt(int64(len(lst)))
		S1.Add(&S1, ai)

		ai2 := new(big.Int).Mul(ai, ai)
		S2.Add(&S2, ai2)

		ai3 := new(big.Int).Mul(ai2, ai)
		S3.Add(&S3, ai3)
	}

	// two word ordered total with distinct lists equals s1 squared minus s2
	S1S1 := new(big.Int).Mul(&S1, &S1)
	total2 := new(big.Int).Sub(S1S1, &S2)

	// three word ordered total with distinct lists equals s1 cubed minus three times s1 times s2 plus two times s3
	S1S1S1 := new(big.Int).Mul(S1S1, &S1)
	S1S2 := new(big.Int).Mul(&S1, &S2)
	threeS1S2 := new(big.Int).Mul(S1S2, big.NewInt(3))
	twoS3 := new(big.Int).Mul(&S3, big.NewInt(2))
	total3 := new(big.Int).Sub(S1S1S1, threeS1S2)
	total3.Add(total3, twoS3)

	// log a quick breakdown plus final totals with commas for readability
	t.Logf("lists discovered: %d", L)
	for i, lst := range g.lists {
		t.Logf("  list[%d] size = %d", i, len(lst))
	}
	t.Logf("2-word ordered (distinct lists) total = %s", withCommasBig(total2))
	if L >= 3 {
		t.Logf("3-word ordered (distinct lists) total = %s", withCommasBig(total3))
	} else {
		t.Logf("3-word ordered (distinct lists) total = N/A (need >=3 lists, have %d)", L)
	}

	// sanity guards totals must be positive or the setup is wrong
	if total2.Sign() <= 0 {
		t.Fatal("expected >0 total for 2-word combinations across all lists")
	}
	if L >= 3 && total3.Sign() <= 0 {
		t.Fatal("expected >0 total for 3-word combinations across all lists")
	}
}
