package doublestar

import (
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestGlobAndGlobWalkDoNotRepeatPaths(t *testing.T) {
	fsys := fstest.MapFS{"a/b/c": {}, "a/d": {}, "f": {}}
	for _, tc := range []struct {
		pattern string
		want    []string
	}{
		{"a/**/**", []string{"a", "a/b", "a/b/c", "a/d"}},
		{"{f,?}", []string{"a", "f"}},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			got, err := Glob(fsys, tc.pattern)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Glob(%q) = %q, %v; want %q", tc.pattern, got, err, tc.want)
			}
			var walked []string
			err = GlobWalk(fsys, tc.pattern, func(p string, _ fs.DirEntry) error {
				walked = append(walked, p)
				return nil
			})
			if err != nil || !reflect.DeepEqual(walked, tc.want) {
				t.Fatalf("GlobWalk(%q) = %q, %v; want %q", tc.pattern, walked, err, tc.want)
			}
		})
	}
}

func TestGlobWalkDuplicateDirectoryStillHonorsSkipDir(t *testing.T) {
	fsys := fstest.MapFS{"a/b/c": {}, "a/d": {}}
	var got []string
	err := GlobWalk(fsys, "a/**/**", func(p string, _ fs.DirEntry) error {
		got = append(got, p)
		if p == "a/b" {
			return SkipDir
		}
		return nil
	})
	if err != nil || !reflect.DeepEqual(got, []string{"a", "a/b", "a/d"}) {
		t.Fatalf("GlobWalk after SkipDir = %q, %v; want [a a/b a/d]", got, err)
	}
}
