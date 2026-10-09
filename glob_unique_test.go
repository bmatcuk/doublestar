package doublestar

import (
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestGlobAndGlobWalkDoNotRepeatPaths(t *testing.T) {
	fsys := fstest.MapFS{
		"a/b/c": {},
		"a/d":   {},
		"f":     {},
	}

	for _, tc := range []struct {
		pattern       string
		want          []string
		wantWalkCalls int
	}{
		{
			pattern:       "a/**",
			want:          []string{"a", "a/b", "a/b/c", "a/d"},
			wantWalkCalls: 4,
		},
		{
			pattern:       "a/**/**",
			want:          []string{"a", "a/b", "a/b/c", "a/d"},
			wantWalkCalls: 4,
		},
		{
			pattern:       "a/**/**/**",
			want:          []string{"a", "a/b", "a/b/c", "a/d"},
			wantWalkCalls: 4,
		},
		{
			pattern:       "**/**",
			want:          []string{".", "a", "a/b", "a/b/c", "a/d", "f"},
			wantWalkCalls: 6,
		},
		{
			pattern:       "{f,?}",
			want:          []string{"a", "f"},
			wantWalkCalls: 2,
		},
		{
			pattern:       "**/?/**",
			want:          []string{"a", "a/b", "a/b/c", "a/d"},
			wantWalkCalls: 6,
		},
		{
			pattern:       "{a/**,a/**/**}",
			want:          []string{"a", "a/b", "a/b/c", "a/d"},
			wantWalkCalls: 4,
		},
		{
			pattern:       "{a/**/**,a/b}",
			want:          []string{"a", "a/b", "a/b/c", "a/d"},
			wantWalkCalls: 4,
		},
	} {
		tc := tc
		t.Run(tc.pattern, func(t *testing.T) {
			got, err := Glob(fsys, tc.pattern)
			if err != nil {
				t.Fatalf("Glob(%q) unexpected error: %v", tc.pattern, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Glob(%q) = %q; want %q", tc.pattern, got, tc.want)
			}

			walkCalls := 0
			var walked []string
			err = GlobWalk(fsys, tc.pattern, func(p string, _ fs.DirEntry) error {
				walkCalls++
				walked = append(walked, p)
				return nil
			})
			if err != nil {
				t.Fatalf("GlobWalk(%q) unexpected error: %v", tc.pattern, err)
			}
			if walkCalls != tc.wantWalkCalls {
				t.Fatalf("GlobWalk(%q) callbacks = %d; want %d", tc.pattern, walkCalls, tc.wantWalkCalls)
			}
		})
	}
}

func TestGlobWalkDuplicateDirectoryStillHonorsSkipDir(t *testing.T) {
	fsys := fstest.MapFS{
		"a/b/c": {},
		"a/d":   {},
	}

	var got []string
	err := GlobWalk(fsys, "a/**/**", func(p string, d fs.DirEntry) error {
		got = append(got, p)
		if p == "a/b" {
			return SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("GlobWalk unexpected error: %v", err)
	}

	want := []string{"a", "a/b", "a/d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GlobWalk after SkipDir = %q; want %q", got, want)
	}
}

func TestSimplifyDoubleStars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a/**/**", "a/**"},
		{"a/**/**/**", "a/**"},
		{"**/**", "**"},
		{"**/**/**", "**"},
		{"/**/**", "/**"},
		{"/**/**/b", "/**/b"},
		{"**/**/b", "**/b"},
		{"a/**/**/b", "a/**/b"},
		{"a/**/**/", "a/**/"},
		{"{a/**/**,b}", "{a/**,b}"},
		{"{**/**}", "{**}"},
		{"x**/**", "x**/**"},
		{"**/**x", "**/**x"},
		{"\\**/**", "\\**/**"},
		{"**/\\**", "**/\\**"},
		{"a/*", "a/*"},
		{"a/**", "a/**"},
		{"a/**/b", "a/**/b"},
	}

	for _, tt := range tests {
		got := simplifyDoubleStars(tt.input)
		if got != tt.want {
			t.Errorf("simplifyDoubleStars(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}
