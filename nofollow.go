package doublestar

import "io/fs"

// An LstatFS is a file system with an Lstat method.
type LstatFS interface {
	fs.FS
	Lstat(name string) (fs.FileInfo, error)
}

// A lstatToStatFS replaces Stat method calls with Lstat method calls.
type lstatToStatFS struct {
	wrapped LstatFS
}

// Open calls the underlying LstatFS's Open method.
func (s lstatToStatFS) Open(name string) (fs.File, error) {
	return s.wrapped.Open(name)
}

// Stat calls the underlying LstatFS's Lstat method.
func (s lstatToStatFS) Stat(name string) (fs.FileInfo, error) {
	return s.wrapped.Lstat(name)
}

// GlobNoFollow is like Glob but does not follow symlinks.
func GlobNoFollow(fsys LstatFS, pattern string) ([]string, error) {
	return Glob(lstatToStatFS{
		wrapped: fsys,
	}, pattern)
}
