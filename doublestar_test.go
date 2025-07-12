package doublestar

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type MatchTest struct {
	pattern, testPath     string // a pattern and path to test the pattern on
	shouldMatch           bool   // true if the pattern should match the path
	shouldMatchGlob       bool   // true if glob should match the path
	expectedErr           error  // an expected error
	expectIOErr           bool   // whether or not to expect an io error
	expectPatternNotExist bool   // whether or not to expect ErrPatternNotExist
	isStandard            bool   // pattern doesn't use any doublestar features (e.g. '**', '{a,b}')
	testOnDisk            bool   // true: test pattern against files in "test" directory
	numResults            int    // number of glob results if testing on disk
	winNumResults         int    // number of glob results on Windows
}

// Tests which contain escapes and symlinks will not work on Windows
var onWindows = runtime.GOOS == "windows"

var matchTests = []MatchTest{
	{"", "", true, false, nil, true, false, true, true, 0, 0},
	{"*", "", true, true, nil, false, false, true, false, 0, 0},
	{"*", "/", false, false, nil, false, false, true, false, 0, 0},
	{"/*", "/", true, true, nil, false, false, true, false, 0, 0},
	{"/*", "/debug/", false, false, nil, false, false, true, false, 0, 0},
	{"/*", "//", false, false, nil, false, false, true, false, 0, 0},
	{"abc", "abc", true, true, nil, false, false, true, true, 1, 1},
	{"*", "abc", true, true, nil, false, false, true, true, 24, 18},
	{"*c", "abc", true, true, nil, false, false, true, true, 2, 2},
	{"*/", "a/", true, true, nil, false, false, true, false, 0, 0},
	{"a*", "a", true, true, nil, false, false, true, true, 9, 9},
	{"a*", "abc", true, true, nil, false, false, true, true, 9, 9},
	{"a*", "ab/c", false, false, nil, false, false, true, true, 9, 9},
	{"a*/b", "abc/b", true, true, nil, false, false, true, true, 2, 2},
	{"a*/b", "a/c/b", false, false, nil, false, false, true, true, 2, 2},
	{"a*/c/", "a/b", false, false, nil, false, false, false, true, 1, 1},
	{"a*b*c*d*e*", "axbxcxdxe", true, true, nil, false, false, true, true, 3, 3},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, true, nil, false, false, true, true, 2, 2},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, true, nil, false, false, true, true, 2, 2},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, false, nil, false, false, true, true, 2, 2},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, false, nil, false, false, true, true, 2, 2},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, true, nil, false, false, true, true, 2, 2},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, false, nil, false, false, true, true, 2, 2},
	{"ab[c]", "abc", true, true, nil, false, false, true, true, 1, 1},
	{"ab[b-d]", "abc", true, true, nil, false, false, true, true, 1, 1},
	{"ab[e-g]", "abc", false, false, nil, false, false, true, true, 0, 0},
	{"ab[^c]", "abc", false, false, nil, false, false, true, true, 0, 0},
	{"ab[^b-d]", "abc", false, false, nil, false, false, true, true, 0, 0},
	{"ab[^e-g]", "abc", true, true, nil, false, false, true, true, 1, 1},
	{"a\\*b", "ab", false, false, nil, false, true, true, !onWindows, 0, 0},
	{"a?b", "a☺b", true, true, nil, false, false, true, true, 1, 1},
	{"a[^a]b", "a☺b", true, true, nil, false, false, true, true, 1, 1},
	{"a[!a]b", "a☺b", true, true, nil, false, false, false, true, 1, 1},
	{"a???b", "a☺b", false, false, nil, false, false, true, true, 0, 0},
	{"a[^a][^a][^a]b", "a☺b", false, false, nil, false, false, true, true, 0, 0},
	{"[a-ζ]*", "α", true, true, nil, false, false, true, true, 21, 17},
	{"*[a-ζ]", "A", false, false, nil, false, false, true, true, 21, 17},
	{"a?b", "a/b", false, false, nil, false, false, true, true, 1, 1},
	{"a*b", "a/b", false, false, nil, false, false, true, true, 1, 1},
	{"[\\]a]", "]", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"[\\-]", "-", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"[x\\-]", "x", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"[x\\-]", "-", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"[x\\-]", "z", false, false, nil, false, false, true, !onWindows, 2, 2},
	{"[\\-x]", "x", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"[\\-x]", "-", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"[\\-x]", "a", false, false, nil, false, false, true, !onWindows, 2, 2},
	{"[]a]", "]", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	// doublestar, like bash, allows these when path.Match() does not
	{"[-]", "-", true, true, nil, false, false, false, !onWindows, 1, 0},
	{"[x-]", "x", true, true, nil, false, false, false, true, 2, 1},
	{"[x-]", "-", true, true, nil, false, false, false, !onWindows, 2, 1},
	{"[x-]", "z", false, false, nil, false, false, false, true, 2, 1},
	{"[-x]", "x", true, true, nil, false, false, false, true, 2, 1},
	{"[-x]", "-", true, true, nil, false, false, false, !onWindows, 2, 1},
	{"[-x]", "a", false, false, nil, false, false, false, true, 2, 1},
	{"[a-b-d]", "a", true, true, nil, false, false, false, true, 3, 2},
	{"[a-b-d]", "b", true, true, nil, false, false, false, true, 3, 2},
	{"[a-b-d]", "-", true, true, nil, false, false, false, !onWindows, 3, 2},
	{"[a-b-d]", "c", false, false, nil, false, false, false, true, 3, 2},
	{"[a-b-x]", "x", true, true, nil, false, false, false, true, 4, 3},
	{"\\", "a", false, false, ErrBadPattern, false, false, true, !onWindows, 0, 0},
	{"[", "a", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"[^", "a", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"[^bc", "a", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"a[", "a", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"a[", "ab", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"ad[", "ab", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"*x", "xxx", true, true, nil, false, false, true, true, 4, 4},
	{"[abc]", "b", true, true, nil, false, false, true, true, 3, 3},
	{"[abc123]", "1", true, true, nil, false, false, true, true, 4, 4},
	{"[a-z0-9]", "1", true, true, nil, false, false, true, true, 7, 7},
	{"**", "", true, true, nil, false, false, false, false, 38, 38},
	{"a/**", "a", true, true, nil, false, false, false, true, 7, 7},
	{"a/**/", "a", true, true, nil, false, false, false, true, 4, 4},
	{"a/**", "a/", true, true, nil, false, false, false, false, 7, 7},
	{"a/**/", "a/", true, true, nil, false, false, false, false, 4, 4},
	{"a/**", "a/b", true, true, nil, false, false, false, true, 7, 7},
	{"a/**", "a/b/c", true, true, nil, false, false, false, true, 7, 7},
	{"**/c", "c", true, true, nil, !onWindows, false, false, true, 5, 4},
	{"**/c", "b/c", true, true, nil, !onWindows, false, false, true, 5, 4},
	{"**/c", "a/b/c", true, true, nil, !onWindows, false, false, true, 5, 4},
	{"**/c", "a/b", false, false, nil, !onWindows, false, false, true, 5, 4},
	{"**/c", "abcd", false, false, nil, !onWindows, false, false, true, 5, 4},
	{"**/c", "a/abc", false, false, nil, !onWindows, false, false, true, 5, 4},
	{"a/**/b", "a/b", true, true, nil, false, false, false, true, 2, 2},
	{"a/**/c", "a/b/c", true, true, nil, false, false, false, true, 2, 2},
	{"a/**/d", "a/b/c/d", true, true, nil, false, false, false, true, 1, 1},
	{"a/\\**", "a/b/c", false, false, nil, false, false, false, !onWindows, 0, 0},
	{"a/\\[*\\]", "a/bc", false, false, nil, false, false, true, !onWindows, 0, 0},
	// this fails the FilepathGlob test on Windows
	{"a/b/c", "a/b//c", false, false, nil, false, false, true, !onWindows, 1, 1},
	// odd: Glob + filepath.Glob return results
	{"a/", "a", false, false, nil, false, false, true, false, 0, 0},
	{"ab{c,d}", "abc", true, true, nil, false, true, false, true, 1, 1},
	{"ab{c,d,*}", "abcde", true, true, nil, false, true, false, true, 5, 5},
	{"ab{c,d}[", "abcd", false, false, ErrBadPattern, false, false, false, true, 0, 0},
	{"a{,bc}", "a", true, true, nil, false, false, false, true, 2, 2},
	{"a{,bc}", "abc", true, true, nil, false, false, false, true, 2, 2},
	{"a/{b/c,c/b}", "a/b/c", true, true, nil, false, false, false, true, 2, 2},
	{"a/{b/c,c/b}", "a/c/b", true, true, nil, false, false, false, true, 2, 2},
	{"a/a*{b,c}", "a/abc", true, true, nil, false, false, false, true, 1, 1},
	{"{a/{b,c},abc}", "a/b", true, true, nil, false, false, false, true, 3, 3},
	{"{a/{b,c},abc}", "a/c", true, true, nil, false, false, false, true, 3, 3},
	{"{a/{b,c},abc}", "abc", true, true, nil, false, false, false, true, 3, 3},
	{"{a/{b,c},abc}", "a/b/c", false, false, nil, false, false, false, true, 3, 3},
	{"{a/ab*}", "a/abc", true, true, nil, false, false, false, true, 1, 1},
	{"{a/*}", "a/b", true, true, nil, false, false, false, true, 3, 3},
	{"{a/abc}", "a/abc", true, true, nil, false, false, false, true, 1, 1},
	{"{a/b,a/c}", "a/c", true, true, nil, false, false, false, true, 2, 2},
	{"abc/**", "abc/b", true, true, nil, false, false, false, true, 3, 3},
	{"**/abc", "abc", true, true, nil, !onWindows, false, false, true, 2, 2},
	{"abc**", "abc/b", false, false, nil, false, false, false, true, 3, 3},
	{"**/*.txt", "abc/【test】.txt", true, true, nil, !onWindows, false, false, true, 1, 1},
	{"**/【*", "abc/【test】.txt", true, true, nil, !onWindows, false, false, true, 1, 1},
	{"**/{a,b}", "a/b", true, true, nil, !onWindows, false, false, true, 5, 5},
	{"a/*/*/d", "a/b/c/d", true, true, nil, false, false, true, true, 1, 1},
	// unfortunately, io/fs can't handle this, so neither can Glob =(
	{"broken-symlink", "broken-symlink", true, true, nil, false, false, true, false, 1, 1},
	{"broken-symlink/*", "a", false, false, nil, false, true, true, true, 0, 0},
	{"broken*/*", "a", false, false, nil, false, false, true, true, 0, 0},
	{"working-symlink/c/*", "working-symlink/c/d", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"working-sym*/*", "working-symlink/c", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"b/**/f", "b/symlink-dir/f", true, true, nil, false, false, false, !onWindows, 2, 2},
	{"*/symlink-dir/*", "b/symlink-dir/f", true, true, nil, !onWindows, false, true, !onWindows, 2, 2},
	{"e/\\[x\\]/*", "e/[x]/[y]", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/\\[x\\]/*/z", "e/[x]/[y]/z", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/**", "e/**", true, true, nil, false, false, false, !onWindows, 14, 9},
	{"e/**", "e/*", true, true, nil, false, false, false, !onWindows, 14, 9},
	{"e/**", "e/?", true, true, nil, false, false, false, !onWindows, 14, 9},
	{"e/**", "e/[", true, true, nil, false, false, false, true, 14, 9},
	{"e/**", "e/]", true, true, nil, false, false, false, true, 14, 9},
	{"e/**", "e/[]", true, true, nil, false, false, false, true, 14, 9},
	{"e/**", "e/{", true, true, nil, false, false, false, true, 14, 9},
	{"e/**", "e/}", true, true, nil, false, false, false, true, 14, 9},
	{"e/**", "e/\\", true, true, nil, false, false, false, !onWindows, 14, 6},
	{"e/*", "e/*", true, true, nil, false, false, true, !onWindows, 11, 5},
	{"e/?", "e/?", true, true, nil, false, false, true, !onWindows, 7, 4},
	{"e/?", "e/*", true, true, nil, false, false, true, !onWindows, 7, 4},
	{"e/?", "e/[", true, true, nil, false, false, true, true, 7, 4},
	{"e/?", "e/]", true, true, nil, false, false, true, true, 7, 4},
	{"e/?", "e/{", true, true, nil, false, false, true, true, 7, 4},
	{"e/?", "e/}", true, true, nil, false, false, true, true, 7, 4},
	{"e/\\[", "e/[", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/[", "e/[", false, false, ErrBadPattern, false, false, true, true, 0, 0},
	{"e/]", "e/]", true, true, nil, false, false, true, true, 1, 1},
	{"e/\\]", "e/]", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/\\{", "e/{", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/\\}", "e/}", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/[\\*\\?]", "e/*", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"e/[\\*\\?]", "e/?", true, true, nil, false, false, true, !onWindows, 2, 2},
	{"e/[\\*\\?]", "e/**", false, false, nil, false, false, true, !onWindows, 2, 2},
	{"e/[\\*\\?]?", "e/**", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/{\\*,\\?}", "e/*", true, true, nil, false, false, false, !onWindows, 2, 2},
	{"e/{\\*,\\?}", "e/?", true, true, nil, false, false, false, !onWindows, 2, 2},
	{"e/\\*", "e/*", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/\\?", "e/?", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"e/\\?", "e/**", false, false, nil, false, false, true, !onWindows, 1, 1},
	{"*\\}", "}", true, true, nil, false, false, true, !onWindows, 1, 1},
	{"nonexistent-path", "a", false, false, nil, false, true, true, true, 0, 0},
	{"nonexistent-path/", "a", false, false, nil, false, true, true, true, 0, 0},
	{"nonexistent-path/file", "a", false, false, nil, false, true, true, true, 0, 0},
	{"nonexistent-path/*", "a", false, false, nil, false, true, true, true, 0, 0},
	{"nonexistent-path/**", "a", false, false, nil, false, true, true, true, 0, 0},
	{"nopermission/*", "nopermission/file", true, false, nil, true, false, true, !onWindows, 0, 0},
	{"nopermission/dir/", "nopermission/dir", false, false, nil, true, false, true, !onWindows, 0, 0},
	{"nopermission/file", "nopermission/file", true, false, nil, true, false, true, !onWindows, 0, 0},
	// this pattern is technically "standard", but path.Glob will fail to match
	{"*/dir/file", "noreaddirpermission/dir/file", true, true, nil, true, false, false, !onWindows, 1, 1},
}

// Calculate the number of results that we expect
// WithFilesOnly at runtime and memoize them here
var numResultsFilesOnly []int

// Calculate the number of results that we expect
// WithNoFollow at runtime and memoize them here
var numResultsNoFollow []int

// Calculate the number of results that we expect with all
// of the options enabled at runtime and memoize them here
var numResultsAllOpts []int

func TestValidatePattern(t *testing.T) {
	for idx, tt := range matchTests {
		testValidatePatternWith(t, idx, tt)
	}
}

func testValidatePatternWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Validate(%#q) panicked: %#v", idx, tt.pattern, r)
		}
	}()

	result := ValidatePattern(tt.pattern)
	if result != (tt.expectedErr == nil) {
		t.Errorf("#%v. ValidatePattern(%#q) = %v want %v", idx, tt.pattern, result, !result)
	}
}

func TestMatch(t *testing.T) {
	for idx, tt := range matchTests {
		// Since Match() always uses "/" as the separator, we
		// don't need to worry about the tt.testOnDisk flag
		testMatchWith(t, idx, tt)
	}
}

func testMatchWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Match(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.testPath, r)
		}
	}()

	// Match() always uses "/" as the separator
	ok, err := Match(tt.pattern, tt.testPath)
	if ok != tt.shouldMatch || err != tt.expectedErr {
		t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, tt.pattern, tt.testPath, ok, err, tt.shouldMatch, tt.expectedErr)
	}

	if tt.isStandard {
		stdOk, stdErr := path.Match(tt.pattern, tt.testPath)
		if ok != stdOk || !compareErrors(err, stdErr) {
			t.Errorf("#%v. Match(%#q, %#q) != path.Match(...). Got %v, %v want %v, %v", idx, tt.pattern, tt.testPath, ok, err, stdOk, stdErr)
		}
	}
}

func BenchmarkMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard {
				Match(tt.pattern, tt.testPath)
			}
		}
	}
}

func BenchmarkGoMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard {
				path.Match(tt.pattern, tt.testPath)
			}
		}
	}
}

func TestMatchUnvalidated(t *testing.T) {
	for idx, tt := range matchTests {
		testMatchUnvalidatedWith(t, idx, tt)
	}
}

func testMatchUnvalidatedWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. MatchUnvalidated(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.testPath, r)
		}
	}()

	// MatchUnvalidated() always uses "/" as the separator
	ok := MatchUnvalidated(tt.pattern, tt.testPath)
	if ok != tt.shouldMatch {
		t.Errorf("#%v. MatchUnvalidated(%#q, %#q) = %v want %v", idx, tt.pattern, tt.testPath, ok, tt.shouldMatch)
	}

	if tt.isStandard {
		stdOk, _ := path.Match(tt.pattern, tt.testPath)
		if ok != stdOk {
			t.Errorf("#%v. MatchUnvalidated(%#q, %#q) != path.Match(...). Got %v want %v", idx, tt.pattern, tt.testPath, ok, stdOk)
		}
	}
}

func TestPathMatch(t *testing.T) {
	for idx, tt := range matchTests {
		// Even though we aren't actually matching paths on disk, we are using
		// PathMatch() which will use the system's separator. As a result, any
		// patterns that might cause problems on-disk need to also be avoided
		// here in this test.
		if tt.testOnDisk {
			testPathMatchWith(t, idx, tt)
		}
	}
}

func testPathMatchWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Match(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.testPath, r)
		}
	}()

	pattern := filepath.FromSlash(tt.pattern)
	testPath := filepath.FromSlash(tt.testPath)
	ok, err := PathMatch(pattern, testPath)
	if ok != tt.shouldMatch || err != tt.expectedErr {
		t.Errorf("#%v. PathMatch(%#q, %#q) = %v, %v want %v, %v", idx, pattern, testPath, ok, err, tt.shouldMatch, tt.expectedErr)
	}

	if tt.isStandard {
		stdOk, stdErr := filepath.Match(pattern, testPath)
		if ok != stdOk || !compareErrors(err, stdErr) {
			t.Errorf("#%v. PathMatch(%#q, %#q) != filepath.Match(...). Got %v, %v want %v, %v", idx, pattern, testPath, ok, err, stdOk, stdErr)
		}
	}
}

func TestPathMatchFake(t *testing.T) {
	// This test fakes that our path separator is `\\` so we can test what it
	// would be like on Windows - obviously, we don't need to do that if we
	// actually _are_ on Windows, since TestPathMatch will cover it.
	if onWindows {
		return
	}

	for idx, tt := range matchTests {
		// Even though we aren't actually matching paths on disk, we are using
		// PathMatch() which will use the system's separator. As a result, any
		// patterns that might cause problems on-disk need to also be avoided
		// here in this test.
		// On Windows, escaping is disabled. Instead, '\\' is treated as path separator.
		// So it's not possible to match escaped wild characters.
		if tt.testOnDisk && !strings.Contains(tt.pattern, "\\") {
			testPathMatchFakeWith(t, idx, tt)
		}
	}
}

func testPathMatchFakeWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Match(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.testPath, r)
		}
	}()

	pattern := strings.ReplaceAll(tt.pattern, "/", "\\")
	testPath := strings.ReplaceAll(tt.testPath, "/", "\\")
	ok, err := matchWithSeparator(pattern, testPath, '\\', true)
	if ok != tt.shouldMatch || err != tt.expectedErr {
		t.Errorf("#%v. PathMatch(%#q, %#q) = %v, %v want %v, %v", idx, pattern, testPath, ok, err, tt.shouldMatch, tt.expectedErr)
	}
}

func BenchmarkPathMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				pattern := filepath.FromSlash(tt.pattern)
				testPath := filepath.FromSlash(tt.testPath)
				PathMatch(pattern, testPath)
			}
		}
	}
}

func BenchmarkGoPathMatch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				pattern := filepath.FromSlash(tt.pattern)
				testPath := filepath.FromSlash(tt.testPath)
				filepath.Match(pattern, testPath)
			}
		}
	}
}

func TestGlob(t *testing.T) {
	doGlobTest(t)
}

func TestGlobWithFailOnIOErrors(t *testing.T) {
	doGlobTest(t, WithFailOnIOErrors())
}

func TestGlobWithFailOnPatternNotExist(t *testing.T) {
	doGlobTest(t, WithFailOnPatternNotExist())
}

func TestGlobWithFilesOnly(t *testing.T) {
	doGlobTest(t, WithFilesOnly())
}

func TestGlobWithNoFollow(t *testing.T) {
	doGlobTest(t, WithNoFollow())
}

func TestGlobWithAllOptions(t *testing.T) {
	doGlobTest(t, WithFailOnIOErrors(), WithFailOnPatternNotExist(), WithFilesOnly(), WithNoFollow())
}

func doGlobTest(t *testing.T, opts ...GlobOption) {
	glob := newGlob(opts...)
	fsys := os.DirFS("test")
	for idx, tt := range matchTests {
		if tt.testOnDisk {
			testGlobWith(t, idx, tt, glob, opts, fsys)
		}
	}
}

func testGlobWith(t *testing.T, idx int, tt MatchTest, g *glob, opts []GlobOption, fsys fs.FS) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Glob(%#q, %#v) panicked: %#v", idx, tt.pattern, g, r)
		}
	}()

	matches, err := Glob(fsys, tt.pattern, opts...)
	verifyGlobResults(t, idx, "Glob", tt, g, fsys, matches, err)
	if len(opts) == 0 {
		testStandardGlob(t, idx, "Glob", tt, fsys, matches, err)
	}
}

func TestGlobWalk(t *testing.T) {
	doGlobWalkTest(t)
}

func TestGlobWalkWithFailOnIOErrors(t *testing.T) {
	doGlobWalkTest(t, WithFailOnIOErrors())
}

func TestGlobWalkWithFailOnPatternNotExist(t *testing.T) {
	doGlobWalkTest(t, WithFailOnPatternNotExist())
}

func TestGlobWalkWithFilesOnly(t *testing.T) {
	doGlobWalkTest(t, WithFilesOnly())
}

func TestGlobWalkWithNoFollow(t *testing.T) {
	doGlobWalkTest(t, WithNoFollow())
}

func TestGlobWalkWithAllOptions(t *testing.T) {
	doGlobWalkTest(t, WithFailOnIOErrors(), WithFailOnPatternNotExist(), WithFilesOnly(), WithNoFollow())
}

func doGlobWalkTest(t *testing.T, opts ...GlobOption) {
	glob := newGlob(opts...)
	fsys := os.DirFS("test")
	for idx, tt := range matchTests {
		if tt.testOnDisk {
			testGlobWalkWith(t, idx, tt, glob, opts, fsys)
		}
	}
}

func testGlobWalkWith(t *testing.T, idx int, tt MatchTest, g *glob, opts []GlobOption, fsys fs.FS) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Glob(%#q, %#v) panicked: %#v", idx, tt.pattern, opts, r)
		}
	}()

	var matches []string
	err := GlobWalk(fsys, tt.pattern, func(p string, d fs.DirEntry) error {
		matches = append(matches, p)
		return nil
	}, opts...)
	verifyGlobResults(t, idx, "GlobWalk", tt, g, fsys, matches, err)
	if len(opts) == 0 {
		testStandardGlob(t, idx, "GlobWalk", tt, fsys, matches, err)
	}
}

func testStandardGlob(t *testing.T, idx int, fn string, tt MatchTest, fsys fs.FS, matches []string, err error) {
	if tt.isStandard {
		stdMatches, stdErr := fs.Glob(fsys, tt.pattern)
		if !compareSlices(matches, stdMatches) || !compareErrors(err, stdErr) {
			t.Errorf("#%v. %v(%#q) != fs.Glob(...). Got %#v, %v want %#v, %v", idx, fn, tt.pattern, matches, err, stdMatches, stdErr)
		}
	}
}

func TestFilepathGlob(t *testing.T) {
	doFilepathGlobTest(t)
}

func TestFilepathGlobWithFailOnIOErrors(t *testing.T) {
	doFilepathGlobTest(t, WithFailOnIOErrors())
}

func TestFilepathGlobWithFailOnPatternNotExist(t *testing.T) {
	doFilepathGlobTest(t, WithFailOnPatternNotExist())
}

func TestFilepathGlobWithFilesOnly(t *testing.T) {
	doFilepathGlobTest(t, WithFilesOnly())
}

func TestFilepathGlobWithNoFollow(t *testing.T) {
	doFilepathGlobTest(t, WithNoFollow())
}

func doFilepathGlobTest(t *testing.T, opts ...GlobOption) {
	glob := newGlob(opts...)
	fsys := os.DirFS("test")

	// The patterns are relative to the "test" sub-directory.
	defer func() {
		os.Chdir("..")
	}()
	os.Chdir("test")

	for idx, tt := range matchTests {
		// Patterns ending with a slash are treated semantically different by
		// FilepathGlob vs Glob because FilepathGlob runs filepath.Clean, which
		// will remove the trailing slash.
		if tt.testOnDisk && !strings.HasSuffix(tt.pattern, "/") {
			ttmod := tt
			ttmod.pattern = filepath.FromSlash(tt.pattern)
			ttmod.testPath = filepath.FromSlash(tt.testPath)
			testFilepathGlobWith(t, idx, ttmod, glob, opts, fsys)
		}
	}
}

func testFilepathGlobWith(t *testing.T, idx int, tt MatchTest, g *glob, opts []GlobOption, fsys fs.FS) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. FilepathGlob(%#q, %#v) panicked: %#v", idx, tt.pattern, g, r)
		}
	}()

	matches, err := FilepathGlob(tt.pattern, opts...)
	verifyGlobResults(t, idx, "FilepathGlob", tt, g, fsys, matches, err)

	if tt.isStandard && len(opts) == 0 {
		stdMatches, stdErr := filepath.Glob(tt.pattern)
		if !compareSlices(matches, stdMatches) || !compareErrors(err, stdErr) {
			t.Errorf("#%v. FilepathGlob(%#q, %#v) != filepath.Glob(...). Got %#v, %v want %#v, %v", idx, tt.pattern, g, matches, err, stdMatches, stdErr)
		}
	}
}

func verifyGlobResults(t *testing.T, idx int, fn string, tt MatchTest, g *glob, fsys fs.FS, matches []string, err error) {
	expectedErr := tt.expectedErr
	if g.failOnPatternNotExist && tt.expectPatternNotExist {
		expectedErr = ErrPatternNotExist
	}

	if g.failOnIOErrors {
		if tt.expectIOErr {
			if err == nil {
				t.Errorf("#%v. %v(%#q, %#v) does not have an error, but should", idx, fn, tt.pattern, g)
			}
			return
		} else if err != nil && err != expectedErr {
			t.Errorf("#%v. %v(%#q, %#v) has error %v, but should not", idx, fn, tt.pattern, g, err)
			return
		}
	}

	if !g.failOnPatternNotExist || !tt.expectPatternNotExist {
		numResults := tt.numResults
		if onWindows {
			numResults = tt.winNumResults
		}
		if g.filesOnly {
			if g.noFollow {
				numResults = numResultsAllOpts[idx]
			} else {
				numResults = numResultsFilesOnly[idx]
			}
		} else if g.noFollow {
			numResults = numResultsNoFollow[idx]
		}

		if len(matches) != numResults {
			t.Errorf("#%v. %v(%#q, %#v) = %#v - should have %#v results, got %#v", idx, fn, tt.pattern, g, matches, numResults, len(matches))
		}
		if !g.filesOnly && !g.noFollow && inSlice(tt.testPath, matches) != tt.shouldMatchGlob {
			if tt.shouldMatchGlob {
				t.Errorf("#%v. %v(%#q, %#v) = %#v - doesn't contain %v, but should", idx, fn, tt.pattern, g, matches, tt.testPath)
			} else {
				t.Errorf("#%v. %v(%#q, %#v) = %#v - contains %v, but shouldn't", idx, fn, tt.pattern, g, matches, tt.testPath)
			}
		}
	}
	if err != expectedErr {
		t.Errorf("#%v. %v(%#q, %#v) has error %v, but should be %v", idx, fn, tt.pattern, g, err, expectedErr)
	}
}

func TestGlobSorted(t *testing.T) {
	fsys := os.DirFS("test")
	expected := []string{"a", "abc", "abcd", "abcde", "abxbbxdbxebxczzx", "abxbbxdbxebxczzy", "axbxcxdxe", "axbxcxdxexxx", "a☺b"}
	matches, err := Glob(fsys, "a*")
	if err != nil {
		t.Errorf("Unexpected error %v", err)
		return
	}

	if len(matches) != len(expected) {
		t.Errorf("Glob returned %#v; expected %#v", matches, expected)
		return
	}
	for idx, match := range matches {
		if match != expected[idx] {
			t.Errorf("Glob returned %#v; expected %#v", matches, expected)
			return
		}
	}
}

func BenchmarkGlob(b *testing.B) {
	fsys := os.DirFS("test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				Glob(fsys, tt.pattern)
			}
		}
	}
}

func BenchmarkGlobWalk(b *testing.B) {
	fsys := os.DirFS("test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				GlobWalk(fsys, tt.pattern, func(p string, d fs.DirEntry) error {
					return nil
				})
			}
		}
	}
}

func BenchmarkGoGlob(b *testing.B) {
	fsys := os.DirFS("test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				fs.Glob(fsys, tt.pattern)
			}
		}
	}
}

func compareErrors(a, b error) bool {
	if a == nil {
		return b == nil
	}
	return b != nil
}

func inSlice(s string, a []string) bool {
	for _, i := range a {
		if i == s {
			return true
		}
	}
	return false
}

func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	diff := make(map[string]int, len(a))

	for _, x := range a {
		diff[x]++
	}

	for _, y := range b {
		if _, ok := diff[y]; !ok {
			return false
		}

		diff[y]--
		if diff[y] == 0 {
			delete(diff, y)
		}
	}

	return len(diff) == 0
}

func buildNumResults() {
	testLen := len(matchTests)
	numResultsFilesOnly = make([]int, testLen, testLen)
	numResultsNoFollow = make([]int, testLen, testLen)
	numResultsAllOpts = make([]int, testLen, testLen)

	fsys := os.DirFS("test")
	g := newGlob()
	for idx, tt := range matchTests {
		if tt.testOnDisk {
			filesOnly := 0
			noFollow := 0
			allOpts := 0
			GlobWalk(fsys, tt.pattern, func(p string, d fs.DirEntry) error {
				isDir, _ := g.isDir(fsys, "", p, d)
				if !isDir {
					filesOnly++
				}

				hasNoFollow := (strings.HasPrefix(tt.pattern, "working-symlink") || !strings.Contains(p, "working-symlink/")) && !strings.Contains(p, "/symlink-dir/")
				if hasNoFollow {
					noFollow++
				}

				if hasNoFollow && (!isDir || p == "working-symlink") {
					allOpts++
				}

				return nil
			})

			numResultsFilesOnly[idx] = filesOnly
			numResultsNoFollow[idx] = noFollow
			numResultsAllOpts[idx] = allOpts
		}
	}
}

func mkdirp(parts ...string) {
	dirs := path.Join(parts...)
	err := os.MkdirAll(dirs, 0755)
	if err != nil {
		log.Fatalf("Could not create test directories %v: %v\n", dirs, err)
	}
}

func touch(parts ...string) {
	filename := path.Join(parts...)
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Could not create test file %v: %v\n", filename, err)
	}
	f.Close()
}

func symlink(oldname, newname string) {
	// since this will only run on non-windows, we can assume "/" as path separator
	err := os.Symlink(oldname, newname)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Could not create symlink %v -> %v: %v\n", oldname, newname, err)
	}
}

func exists(parts ...string) bool {
	p := path.Join(parts...)
	_, err := os.Lstat(p)
	return err == nil
}

func TestMain(m *testing.M) {
	// create the test directory
	mkdirp("test", "a", "b", "c")
	mkdirp("test", "a", "c")
	mkdirp("test", "abc")
	mkdirp("test", "axbxcxdxe", "xxx")
	mkdirp("test", "axbxcxdxexxx")
	mkdirp("test", "b")
	mkdirp("test", "e", "[x]", "[y]")

	// create test files
	touch("test", "1")
	touch("test", "a", "abc")
	touch("test", "a", "b", "c", "d")
	touch("test", "a", "c", "b")
	touch("test", "abc", "b")
	touch("test", "abcd")
	touch("test", "abcde")
	touch("test", "abxbbxdbxebxczzx")
	touch("test", "abxbbxdbxebxczzy")
	touch("test", "axbxcxdxe", "f")
	touch("test", "axbxcxdxe", "xxx", "f")
	touch("test", "axbxcxdxexxx", "f")
	touch("test", "axbxcxdxexxx", "fff")
	touch("test", "a☺b")
	touch("test", "b", "c")
	touch("test", "c")
	touch("test", "x")
	touch("test", "xxx")
	touch("test", "z")
	touch("test", "α")
	touch("test", "abc", "【test】.txt")

	touch("test", "e", "[")
	touch("test", "e", "]")
	touch("test", "e", "{")
	touch("test", "e", "}")
	touch("test", "e", "[]")
	touch("test", "e", "[x]", "[y]", "z")

	touch("test", "}")

	if !onWindows {
		// these files/symlinks won't work on Windows
		touch("test", "-")
		touch("test", "]")
		touch("test", "e", "*")
		touch("test", "e", "**")
		touch("test", "e", "****")
		touch("test", "e", "?")
		touch("test", "e", "\\")

		symlink("../axbxcxdxe/", "test/b/symlink-dir")
		symlink("/tmp/nonexistant-file-20160902155705", "test/broken-symlink")
		symlink("a/b", "test/working-symlink")

		// no permissions at all
		if !exists("test", "nopermission") {
			mkdirp("test", "nopermission", "dir")
			touch("test", "nopermission", "file")
			os.Chmod(path.Join("test", "nopermission"), 0)
		}

		// no permission to read dir
		if !exists("test", "noreaddirpermission") {
			mkdirp("test", "noreaddirpermission", "dir")
			touch("test", "noreaddirpermission", "dir", "file")
			os.Chmod(path.Join("test", "noreaddirpermission"), 0o111)
		}
	}

	// initialize numResultsFilesOnly
	buildNumResults()

	os.Exit(m.Run())
}
