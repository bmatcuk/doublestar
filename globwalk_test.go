package doublestar

import (
	"io/fs"
	"os"
	"path"
	"testing"
)

type SkipTest struct {
	pattern          string // pattern to test
	skipOn           string // a path to skip
	shouldNotContain string // a path that should not match
	numResults       int    // number of expected matches
	winNumResults    int    // number of expected matches on windows
}

var skipTests = []SkipTest{
	{"a", "a", "a", 0, 0},
	{"a/", "a", "a", 1, 1},
	{"*", "b", "b/c", 24, 21},
	{"a/**", "a", "a", 0, 0},
	{"a/**", "a/abc", "a/b", 1, 1},
	{"a/**", "a/b/c", "a/b/c/d", 5, 5},
	{"a/{**,c/*}", "a/b/c", "a/b/c/d", 5, 5},
	{"a/{**,c/*}", "a/abc", "a/b", 1, 1},
}

var symlinkSkipTests = []SkipTest{
	{"symlinks/**", "symlinks/symlink-dir", "symlinks/symlink-dir/f", 3, 3},
}

var symlinkNoFollowSkipTests = []SkipTest{
	{"symlinks/**", "symlinks/symlink-dir", "symlinks/working-symlink", 2, 2},
}

func TestSkipDirInGlobWalk(t *testing.T) {
	fsys := os.DirFS("testdata")
	for idx, tt := range skipTests {
		testSkipDirInGlobWalkWith(t, idx, tt, fsys)
	}
}

func TestSymlinkSkipDirInGlobWalk(t *testing.T) {
	// ensure the symlinks exist - they may not on Windows
	p := path.Join("testdata", "symlinks", "symlink-dir")
	if _, err := os.Lstat(p); err != nil {
		t.Skip("Symlinks do not exist.")
	}

	fsys := os.DirFS("testdata")
	for idx, tt := range symlinkSkipTests {
		testSkipDirInGlobWalkWith(t, idx, tt, fsys)
	}
	for idx, tt := range symlinkNoFollowSkipTests {
		testSkipDirInGlobWalkWith(t, idx, tt, fsys, WithNoFollow())
	}
}

func testSkipDirInGlobWalkWith(t *testing.T, idx int, tt SkipTest, fsys fs.FS, opts ...GlobOption) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. GlobWalk(%#q) panicked: %#v", idx, tt.pattern, r)
		}
	}()

	var matches []string
	hadBadMatch := false
	GlobWalk(fsys, tt.pattern, func(p string, d fs.DirEntry) error {
		if p == tt.skipOn {
			return SkipDir
		}
		if p == tt.shouldNotContain {
			hadBadMatch = true
		}
		matches = append(matches, p)
		return nil
	}, opts...)

	expected := tt.numResults
	if onWindows {
		expected = tt.winNumResults
	}
	if len(matches) != expected {
		t.Errorf("#%v. GlobWalk(%#q) = %#v - should have %#v results, got %#v", idx, tt.pattern, matches, expected, len(matches))
	}
	if hadBadMatch {
		t.Errorf("#%v. GlobWalk(%#q) should not have matched %#q, but did", idx, tt.pattern, tt.shouldNotContain)
	}
}
