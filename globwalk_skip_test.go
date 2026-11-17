package doublestar

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestGlobWalkSkipDirSiblings(t *testing.T) {
	files := fstest.MapFS{
		"a/x":        {},
		"b":          {},
		"c/y":        {},
		"d":          {},
		"nested/a/x": {},
		"nested/b":   {},
		"nested/c/y": {},
		"nested/d":   {},
	}
	dir := t.TempDir()
	for name := range files {
		name = filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	stop := errors.New("stop walking")
	for _, backend := range []struct {
		name string
		fsys fs.FS
	}{
		{"map", files},
		{"disk", os.DirFS(dir)},
	} {
		t.Run(backend.name, func(t *testing.T) {
			for _, tt := range []struct {
				name, pattern, stopOn string
				callbackErr, wantErr  error
				want                  []string
				opts                  []GlobOption
			}{
				{"star directory", "*", "a", SkipDir, nil, []string{"a", "b", "c", "d", "nested"}, nil},
				{"question directory", "?", "a", SkipDir, nil, []string{"a", "b", "c", "d"}, nil},
				{"class directory", "[a-d]", "a", SkipDir, nil, []string{"a", "b", "c", "d"}, nil},
				{"nested star directory", "nested/*", "nested/a", SkipDir, nil, []string{"nested/a", "nested/b", "nested/c", "nested/d"}, nil},
				{"nested question directory", "nested/?", "nested/a", SkipDir, nil, []string{"nested/a", "nested/b", "nested/c", "nested/d"}, nil},
				{"nested class directory", "nested/[a-d]", "nested/a", SkipDir, nil, []string{"nested/a", "nested/b", "nested/c", "nested/d"}, nil},
				{"wildcard parent directory", "*/[a-d]", "nested/a", SkipDir, nil, []string{"nested/a", "nested/b", "nested/c", "nested/d"}, nil},
				{"file", "*", "b", SkipDir, nil, []string{"a", "b"}, nil},
				{"nested file", "nested/*", "nested/b", SkipDir, nil, []string{"nested/a", "nested/b"}, nil},
				{"files only", "*", "b", SkipDir, nil, []string{"b"}, []GlobOption{WithFilesOnly()}},
				{"no skip", "*", "", nil, nil, []string{"a", "b", "c", "d", "nested"}, nil},
				{"callback error", "*", "a", stop, stop, []string{"a"}, nil},
				{"recursive directory", "a/**", "a", SkipDir, nil, []string{"a"}, nil},
			} {
				t.Run(tt.name, func(t *testing.T) {
					var visited []string
					err := GlobWalk(backend.fsys, tt.pattern, func(p string, d fs.DirEntry) error {
						visited = append(visited, p)
						if p == tt.stopOn {
							return tt.callbackErr
						}
						return nil
					}, tt.opts...)
					if err != tt.wantErr {
						t.Errorf("GlobWalk error = %v, want %v", err, tt.wantErr)
					}
					if !reflect.DeepEqual(visited, tt.want) {
						t.Errorf("visited = %q, want %q", visited, tt.want)
					}
				})
			}
		})
	}
}

func TestGlobWalkSkipDirSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "target"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(dir, "a")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	for _, tt := range []struct {
		name string
		opts []GlobOption
		want []string
	}{
		{"follow", nil, []string{"a", "b"}},
		{"no follow", []GlobOption{WithNoFollow()}, []string{"a"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var visited []string
			err := GlobWalk(os.DirFS(dir), "[ab]", func(p string, d fs.DirEntry) error {
				visited = append(visited, p)
				if p == "a" {
					return SkipDir
				}
				return nil
			}, tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(visited, tt.want) {
				t.Errorf("visited = %q, want %q", visited, tt.want)
			}
		})
	}
}
