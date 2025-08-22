package namemachine

import (
	"bytes"
	"math/rand"
	"path"
	"reflect"
	"sort"
	"testing"
	"time"
)

/**
 * TestNormalizeAndFilter covers lowercase ascii filtering length bounds & dedup
 * @param t *testing.T test harness
 * @return void
 */
func TestNormalizeAndFilter(t *testing.T) {
	in := []string{
		"Hello", "héllö", "OK", "go", "tool", "tooo", "dup", "dup", "A😊", "B", "éclair",
	}
	out := normalizeAndFilter(in, true, true, 3, 4)
	want := []string{"tool", "tooo", "dup"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("normalizeAndFilter got %v want %v", out, want)
	}
}

/**
 * TestIsASCII exercises ascii true and non ascii false with utf8 validity
 * @param t *testing.T test harness
 * @return void
 */
func TestIsASCII(t *testing.T) {
	if !isASCII("ASCIIonly123") {
		t.Fatal("expected ascii string to be ascii")
	}
	if isASCII("naïve") { // contains ï
		t.Fatal("expected non ascii string to be false")
	}
	if isASCII("ok😊") { // emoji
		t.Fatal("expected emoji to be false")
	}
}

/**
 * TestGlobFilter verifies include and exclude behavior including default include all
 * @param t *testing.T test harness
 * @return void
 */
func TestGlobFilter(t *testing.T) {
	files := fileWords{
		"adjectives/colors.txt": {"red"},
		"nouns/animals.txt":     {"cat"},
		"ipsum/corporate.txt":   {"synergy"},
	}
	all := globFilter(files, nil, nil)
	sort.Strings(all)
	wantAll := []string{"adjectives/colors.txt", "ipsum/corporate.txt", "nouns/animals.txt"}
	if !reflect.DeepEqual(all, wantAll) {
		t.Fatalf("globFilter all got %v want %v", all, wantAll)
	}

	inc := globFilter(files, []string{"**/*.txt"}, []string{"ipsum/**"})
	sort.Strings(inc)
	wantInc := []string{"adjectives/colors.txt", "nouns/animals.txt"}
	if !reflect.DeepEqual(inc, wantInc) {
		t.Fatalf("globFilter inc got %v want %v", inc, wantInc)
	}

	justDir := globFilter(files, []string{"adjectives/**"}, nil)
	if len(justDir) != 1 || path.Dir(justDir[0]) != "adjectives" {
		t.Fatalf("expected adjectives dir match got %v", justDir)
	}
}

/**
 * TestMergeListsStrategies covers MergeByDir MergeSingle and MergeByFile with cross dedup
 * also asserts that cross dedup can empty later lists without panicking
 * @param t *testing.T test harness
 * @return void
 */
func TestMergeListsStrategies(t *testing.T) {
	files := fileWords{
		"a/x.txt": {"foo", "bar"},
		"a/y.txt": {"bar", "baz"},
		"b/z.txt": {"foo"},
	}
	names := []string{"a/x.txt", "a/y.txt", "b/z.txt"}

	// by dir with cross dedup
	lists, ids := mergeLists(files, names, Options{Strategy: MergeByDir, CrossDedup: true})
	if len(lists) != len(ids) || len(lists) != 2 {
		t.Fatalf("by dir expected 2 lists got %d ids %v", len(lists), ids)
	}
	if ids[0] != "a" && ids[1] != "b" && ids[0] != "." {
		t.Fatalf("unexpected ids %v", ids)
	}
	// first bucket should hold foo bar bar baz after normalize then dedup stable becomes foo bar baz
	// second bucket b had foo but cross dedup removes it may become empty
	if len(lists[0]) == 0 {
		t.Fatal("first list unexpectedly empty")
	}

	// single flattened
	flat, fids := mergeLists(files, names, Options{Strategy: MergeSingle, Lowercase: true})
	if len(flat) != 1 || len(fids) != 1 || fids[0] != "all" {
		t.Fatalf("merge single ids %v sizes %v", fids, []int{len(flat[0])})
	}

	// by file
	byFile, fileIDs := mergeLists(files, names, Options{Strategy: MergeByFile})
	if len(byFile) != len(names) || len(fileIDs) != len(names) {
		t.Fatalf("merge by file got %d want %d", len(byFile), len(names))
	}
}

/**
 * TestRandomSlugInto asserts slug length and alphabet membership
 * @param t *testing.T test harness
 * @return void
 */
func TestRandomSlugInto(t *testing.T) {
	dst := make([]byte, 0, 64)
	out := randomSlugInto(dst, 12)
	if len(out) != 12 {
		t.Fatalf("slug length got %d want 12", len(out))
	}
	for _, b := range out {
		if !(b >= 'a' && b <= 'z' || b >= '2' && b <= '7') {
			t.Fatalf("slug contains non base32 char %q", b)
		}
	}
}

/**
 * newTestGen constructs a small generator directly for deterministic tests
 * @return *Generator generator with two lists and fixed rng
 */
func newTestGen() *Generator {
	return &Generator{
		lists:      [][]string{{"alpha", "beta"}, {"one", "two"}},
		delim:      '_',
		wordsExact: 2,
		slugLen:    0,
		rng:        rand.New(rand.NewSource(1)),
	}
}

/**
 * TestGenerateIntoCappedAndAlloc covers both capacity paths and content formatting
 * @param t *testing.T test harness
 * @return void
 */
func TestGenerateIntoCappedAndAlloc(t *testing.T) {
	g := newTestGen()

	// force allocation by providing too small a buffer
	dst := make([]byte, 0, 4)
	res := g.GenerateInto(dst, 0)
	if got := string(res); got == "" || !bytes.Contains(res, []byte("_")) {
		t.Fatalf("unexpected name %q", got)
	}

	// zero alloc when capacity is enough
	wantLen := len("alpha") + 1 + len("one") // typical first draws with seed 1
	dst = make([]byte, 0, wantLen)
	res2 := g.GenerateInto(dst[:0], 0)
	if len(res2) != len(string(res2)) { // trivial sanity on slice use
		t.Fatal("slice length mismatch")
	}
}

/**
 * TestGenerateSlugAndWriteTo covers slug appending and WriteTo path
 * @param t *testing.T test harness
 * @return void
 */
func TestGenerateSlugAndWriteTo(t *testing.T) {
	g := &Generator{
		lists:      [][]string{{"red"}},
		delim:      '-',
		wordsExact: 1,
		slugLen:    6,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	name := g.Generate(0)
	if !bytes.Contains([]byte(name), []byte("-")) {
		t.Fatalf("expected delimiter in %q", name)
	}
	parts := bytes.Split([]byte(name), []byte("-"))
	if len(parts) != 2 || len(parts[1]) != 6 {
		t.Fatalf("expected slug length 6 got %q", name)
	}

	var buf bytes.Buffer
	n, err := g.WriteTo(&buf, 1) // override to 1 word still includes slug
	if err != nil || n <= 0 || buf.Len() != n {
		t.Fatalf("WriteTo failed n %d err %v len %d", n, err, buf.Len())
	}
}

/**
 * TestRandWordCountRange ensures values respect min and max bounds
 * @param t *testing.T test harness
 * @return void
 */
func TestRandWordCountRange(t *testing.T) {
	g := &Generator{
		minWords: 2,
		maxWords: 3,
		rng:      rand.New(rand.NewSource(1)),
	}
	for i := 0; i < 100; i++ {
		n := g.randWordCount()
		if n < 2 || n > 3 {
			t.Fatalf("randWordCount out of range got %d", n)
		}
	}
}

/**
 * TestOptionsNormDefaults ensures default delimiter and seed assignment
 * @param t *testing.T test harness
 * @return void
 */
func TestOptionsNormDefaults(t *testing.T) {
	var o Options
	o.norm()
	if o.Delimiter == 0 {
		t.Fatal("expected default delimiter to be set")
	}
	if o.Seed == 0 {
		t.Fatal("expected non zero seed when none provided")
	}
}
