package doublestar

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

// Windows junctions are reported as ModeIrregular by ReadDir since Go 1.23,
// while Stat follows them and reports the target's mode.
type irregularEntryFS struct {
	fstest.MapFS
	statErr error
}

type irregularEntry struct{ fs.DirEntry }
type irregularInfo struct{ fs.FileInfo }

func (irregularEntry) Type() fs.FileMode { return fs.ModeIrregular }
func (irregularEntry) IsDir() bool       { return false }
func (irregularInfo) Mode() fs.FileMode  { return fs.ModeIrregular | 0666 }
func (irregularInfo) IsDir() bool        { return false }

func (e irregularEntry) Info() (fs.FileInfo, error) {
	info, err := e.DirEntry.Info()
	if err != nil {
		return nil, err
	}
	return irregularInfo{info}, nil
}

func (f irregularEntryFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := f.MapFS.ReadDir(name)
	if name == "." {
		for i, entry := range entries {
			if entry.Name() == "junction" || entry.Name() == "irregular-file" {
				entries[i] = irregularEntry{entry}
			}
		}
	}
	return entries, err
}

func (f irregularEntryFS) Stat(name string) (fs.FileInfo, error) {
	if f.statErr != nil && (name == "junction" || name == "irregular-file") {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: f.statErr}
	}
	return f.MapFS.Stat(name)
}

func TestGlobIrregularEntries(t *testing.T) {
	fsys := irregularEntryFS{MapFS: fstest.MapFS{
		"junction/Documents/file.txt": {},
		"ordinary/Documents/file.txt": {},
		"irregular-file":              {},
		"plain.txt":                   {},
	}}
	tests := []struct {
		name    string
		pattern string
		opts    []GlobOption
		statErr error
		want    []string
		wantErr error
	}{
		{"directory", "*/Documents", nil, nil, []string{"junction/Documents", "ordinary/Documents"}, nil},
		{"recursive", "**/file.txt", nil, nil, []string{"junction/Documents/file.txt", "ordinary/Documents/file.txt"}, nil},
		{"files only", "*", []GlobOption{WithFilesOnly()}, nil, []string{"irregular-file", "plain.txt"}, nil},
		{"no follow", "*/Documents", []GlobOption{WithNoFollow()}, nil, []string{"ordinary/Documents"}, nil},
		{"no follow recursive", "**/file.txt", []GlobOption{WithNoFollow()}, nil, []string{"ordinary/Documents/file.txt"}, nil},
		{"no follow files only", "*", []GlobOption{WithNoFollow(), WithFilesOnly()}, nil, []string{"irregular-file", "junction", "plain.txt"}, nil},
		{"stat error ignored", "*/Documents", nil, fs.ErrPermission, []string{"ordinary/Documents"}, nil},
		{"stat error returned", "*/Documents", []GlobOption{WithFailOnIOErrors()}, fs.ErrPermission, nil, fs.ErrPermission},
		{"no follow avoids stat", "*/Documents", []GlobOption{WithNoFollow(), WithFailOnIOErrors()}, fs.ErrPermission, []string{"ordinary/Documents"}, nil},
		{"dangling", "*/Documents", []GlobOption{WithFailOnIOErrors()}, fs.ErrNotExist, []string{"ordinary/Documents"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys.statErr = tt.statErr
			t.Run("Glob", func(t *testing.T) {
				got, err := Glob(fsys, tt.pattern, tt.opts...)
				if !errors.Is(err, tt.wantErr) || !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Glob() = %v, %v; want %v, %v", got, err, tt.want, tt.wantErr)
				}
			})
			t.Run("GlobWalk", func(t *testing.T) {
				var got []string
				err := GlobWalk(fsys, tt.pattern, func(p string, _ fs.DirEntry) error {
					got = append(got, p)
					return nil
				}, tt.opts...)
				if !errors.Is(err, tt.wantErr) || !reflect.DeepEqual(got, tt.want) {
					t.Errorf("GlobWalk() = %v, %v; want %v, %v", got, err, tt.want, tt.wantErr)
				}
			})
		})
	}
}
