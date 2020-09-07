// This file is mostly copied from Go's path/match_test.go

package doublestar

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type MatchTest struct {
	pattern, testPath string // a pattern and path to test the pattern on
	shouldMatch       bool   // true if the pattern should match the path
	expectedErr       error  // an expected error
	isStandard        bool   // pattern doesn't use any doublestar features
	testOnDisk        bool   // true: test pattern against files in "test" directory
}

// Tests which contain escapes and symlinks will not work on Windows
var onWindows = runtime.GOOS == "windows"

var matchTests = []MatchTest{
	{"*", "", true, nil, true, false},
	{"*", "/", false, nil, true, false},
	{"/*", "/", true, nil, true, false},
	{"/*", "/debug/", false, nil, true, false},
	{"/*", "//", false, nil, true, false},
	{"abc", "abc", true, nil, true, true},
	{"*", "abc", true, nil, true, true},
	{"*c", "abc", true, nil, true, true},
	{"*/", "a/", true, nil, true, false},
	{"a*", "a", true, nil, true, true},
	{"a*", "abc", true, nil, true, true},
	{"a*", "ab/c", false, nil, true, true},
	{"a*/b", "abc/b", true, nil, true, true},
	{"a*/b", "a/c/b", false, nil, true, true},
	{"a*b*c*d*e*", "axbxcxdxe", true, nil, true, true},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil, true, true},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil, true, true},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil, true, true},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil, true, true},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil, true, true},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil, true, true},
	{"ab[c]", "abc", true, nil, true, true},
	{"ab[b-d]", "abc", true, nil, true, true},
	{"ab[e-g]", "abc", false, nil, true, true},
	{"ab[^c]", "abc", false, nil, true, true},
	{"ab[^b-d]", "abc", false, nil, true, true},
	{"ab[^e-g]", "abc", true, nil, true, true},
	{"a\\*b", "ab", false, nil, true, true},
	{"a?b", "a☺b", true, nil, true, true},
	{"a[^a]b", "a☺b", true, nil, true, true},
	{"a???b", "a☺b", false, nil, true, true},
	{"a[^a][^a][^a]b", "a☺b", false, nil, true, true},
	{"[a-ζ]*", "α", true, nil, true, true},
	{"*[a-ζ]", "A", false, nil, true, true},
	{"a?b", "a/b", false, nil, true, true},
	{"a*b", "a/b", false, nil, true, true},
	{"[\\]a]", "]", true, nil, true, !onWindows},
	{"[\\-]", "-", true, nil, true, !onWindows},
	{"[x\\-]", "x", true, nil, true, !onWindows},
	{"[x\\-]", "-", true, nil, true, !onWindows},
	{"[x\\-]", "z", false, nil, true, !onWindows},
	{"[\\-x]", "x", true, nil, true, !onWindows},
	{"[\\-x]", "-", true, nil, true, !onWindows},
	{"[\\-x]", "a", false, nil, true, !onWindows},
	{"[]a]", "]", false, ErrBadPattern, true, true},
	{"[-]", "-", false, ErrBadPattern, true, true},
	{"[x-]", "x", false, ErrBadPattern, true, true},
	{"[x-]", "-", false, ErrBadPattern, true, true},
	{"[x-]", "z", false, ErrBadPattern, true, true},
	{"[-x]", "x", false, ErrBadPattern, true, true},
	{"[-x]", "-", false, ErrBadPattern, true, true},
	{"[-x]", "a", false, ErrBadPattern, true, true},
	{"\\", "a", false, ErrBadPattern, true, !onWindows},
	{"[a-b-c]", "a", false, ErrBadPattern, true, true},
	{"[", "a", false, ErrBadPattern, true, true},
	{"[^", "a", false, ErrBadPattern, true, true},
	{"[^bc", "a", false, ErrBadPattern, true, true},
	{"a[", "a", false, nil, true, false},
	{"a[", "ab", false, ErrBadPattern, true, true},
	{"*x", "xxx", true, nil, true, true},
	{"[abc]", "b", true, nil, true, true},
	{"**", "", true, nil, false, false},
	{"a/**", "a", false, nil, false, true},
	{"a/**", "a/b", true, nil, false, true},
	{"a/**", "a/b/c", true, nil, false, true},
	{"**/c", "c", true, nil, false, true},
	{"**/c", "b/c", true, nil, false, true},
	{"**/c", "a/b/c", true, nil, false, true},
	{"**/c", "a/b", false, nil, false, true},
	{"**/c", "abcd", false, nil, false, true},
	{"**/c", "a/abc", false, nil, false, true},
	{"a/**/b", "a/b", true, nil, false, true},
	{"a/**/c", "a/b/c", true, nil, false, true},
	{"a/**/d", "a/b/c/d", true, nil, false, true},
	{"a/\\**", "a/b/c", false, nil, false, !onWindows},
	// this is an odd case: filepath.Glob() will return results
	{"a//b/c", "a/b/c", false, nil, true, false},
	{"a/b/c", "a/b//c", false, nil, true, true},
	// also odd: Glob + filepath.Glob return results
	{"a/", "a", false, nil, true, false},
	{"ab{c,d}", "abc", true, nil, false, true},
	{"ab{c,d,*}", "abcde", true, nil, false, true},
	{"ab{c,d}[", "abcd", false, ErrBadPattern, false, true},
	{"a{,bc}", "a", true, nil, false, true},
	{"a{,bc}", "abc", true, nil, false, true},
	{"a/{b/c,c/b}", "a/b/c", true, nil, false, true},
	{"a/{b/c,c/b}", "a/c/b", true, nil, false, true},
	{"{a/{b,c},abc}", "a/b", true, nil, false, true},
	{"{a/{b,c},abc}", "a/c", true, nil, false, true},
	{"{a/{b,c},abc}", "abc", true, nil, false, true},
	{"{a/{b,c},abc}", "a/b/c", false, nil, false, true},
	{"{a/ab*}", "a/abc", true, nil, false, true},
	{"{a/*}", "a/b", true, nil, false, true},
	{"{a/abc}", "a/abc", true, nil, false, true},
	{"{a/b,a/c}", "a/c", true, nil, false, true},
	{"abc/**", "abc/b", true, nil, false, true},
	{"**/abc", "abc", true, nil, false, true},
	{"abc**", "abc/b", false, nil, false, true},
	{"**/*.txt", "abc/【test】.txt", true, nil, false, true},
	{"**/【*", "abc/【test】.txt", true, nil, false, true},
	{"broken-symlink", "broken-symlink", true, nil, true, !onWindows},
	{"working-symlink/c/*", "working-symlink/c/d", true, nil, true, !onWindows},
	{"working-sym*/*", "working-symlink/c", true, nil, true, !onWindows},
	{"b/**/f", "b/symlink-dir/f", true, nil, false, !onWindows},
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
		t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, pattern, testPath, ok, err, tt.shouldMatch, tt.expectedErr)
	}

	if tt.isStandard {
		stdOk, stdErr := filepath.Match(pattern, testPath)
		if ok != stdOk || !compareErrors(err, stdErr) {
			t.Errorf("#%v. PathMatch(%#q, %#q) != filepath.Match(...). Got %v, %v want %v, %v", idx, pattern, testPath, ok, err, stdOk, stdErr)
		}
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
	abspath, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current working directory: %v", err)
		return
	}

	abspath = filepath.Join(abspath, "test")

	abspathWithoutVolume := ""
	volumeName := filepath.VolumeName(abspath)
	if volumeName != "" && !strings.HasPrefix(volumeName, `\\`) {
		abspathWithoutVolume = strings.TrimPrefix(abspath, volumeName)
	}

	for idx, tt := range matchTests {
		if tt.testOnDisk {
			// test both relative paths and absolute paths
			testGlobWith(t, idx, tt, "test")
			testGlobWith(t, idx, tt, abspath)
			if abspathWithoutVolume != "" {
				testGlobWith(t, idx, tt, abspathWithoutVolume)
			}
		}
	}
}

func testGlobWith(t *testing.T, idx int, tt MatchTest, basepath string) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Glob(%#q) panicked: %#v", idx, tt.pattern, r)
		}
	}()

	pattern := joinWithoutClean(basepath, filepath.FromSlash(tt.pattern))
	testPath := joinWithoutClean(basepath, filepath.FromSlash(tt.testPath))
	matches, err := Glob(pattern)
	if inSlice(testPath, matches) != tt.shouldMatch {
		if tt.shouldMatch {
			t.Errorf("#%v. Glob(%#q) = %#v - doesn't contain %v, but should", idx, pattern, matches, tt.testPath)
		} else {
			t.Errorf("#%v. Glob(%#q) = %#v - contains %v, but shouldn't", idx, pattern, matches, tt.testPath)
		}
	}
	if err != tt.expectedErr {
		t.Errorf("#%v. Glob(%#q) has error %v, but should be %v", idx, pattern, err, tt.expectedErr)
	}

	if tt.isStandard {
		stdMatches, stdErr := filepath.Glob(pattern)
		if !compareSlices(matches, stdMatches) || !compareErrors(err, stdErr) {
			t.Errorf("#%v. Glob(%#q) != filepath.Glob(...). Got %#v, %v want %#v, %v", idx, pattern, matches, err, stdMatches, stdErr)
		}
	}
}

func BenchmarkGlob(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				pattern := joinWithoutClean("test", filepath.FromSlash(tt.pattern))
				Glob(pattern)
			}
		}
	}
}

func BenchmarkGoGlob(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tt := range matchTests {
			if tt.isStandard && tt.testOnDisk {
				pattern := joinWithoutClean("test", filepath.FromSlash(tt.pattern))
				filepath.Glob(pattern)
			}
		}
	}
}

func joinWithoutClean(elem ...string) string {
	return strings.Join(elem, string(os.PathSeparator))
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

func TestGlobSorted(t *testing.T) {
	expected := []string{"a", "abc", "abcd", "abcde", "abxbbxdbxebxczzx", "abxbbxdbxebxczzy", "axbxcxdxe", "axbxcxdxexxx", "a☺b"}
	matches, err := Glob(joinWithoutClean("test", "a*"))
	if err != nil {
		t.Errorf("Unexpected error %v", err)
		return
	}

	if len(matches) != len(expected) {
		t.Errorf("Glob returned %#v; expected %#v", matches, expected)
		return
	}
	for idx, match := range matches {
		if match != joinWithoutClean("test", expected[idx]) {
			t.Errorf("Glob returned %#v; expected %#v", matches, expected)
			return
		}
	}
}

func TestMain(m *testing.M) {
	// create the test directory
	mkdirp("test", "a", "b", "c")
	mkdirp("test", "a", "c")
	mkdirp("test", "abc")
	mkdirp("test", "axbxcxdxe", "xxx")
	mkdirp("test", "axbxcxdxexxx")
	mkdirp("test", "b")

	// create test files
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

	if !onWindows {
		// these files/symlinks won't work on Windows
		touch("test", "-")
		touch("test", "]")
		symlink("../axbxcxdxe/", "test/b/symlink-dir")
		symlink("/tmp/nonexistant-file-20160902155705", "test/broken-symlink")
		symlink("a/b", "test/working-symlink")
	}

	os.Exit(m.Run())
}
