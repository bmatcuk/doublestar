package doublestar

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// If returned from GlobWalkFunc, will cause GlobWalk to skip the current
// directory. In other words, if the current path is a directory, GlobWalk will
// not recurse into it. Otherwise, GlobWalk will skip the rest of the current
// directory.
var SkipDir = fs.SkipDir

// Callback function for GlobWalk(). If the function returns an error, GlobWalk
// will end immediately and return the same error.
type GlobWalkFunc func(path string, d fs.DirEntry) error

// GlobWalk calls the callback function `fn` for every file matching pattern.
// The syntax of pattern is the same as in Match() and the behavior is the same
// as Glob(), with regard to limitations (such as patterns containing `/./`,
// `/../`, or starting with `/`). The pattern may describe hierarchical names
// such as usr/*/bin/ed.
//
// GlobWalk may have a small performance benefit over Glob if you do not need a
// slice of matches because it can avoid allocating memory for the matches.
// Additionally, GlobWalk gives you access to the `fs.DirEntry` objects for
// each match, and lets you quit early by returning a non-nil error from your
// callback function. Like `io/fs.WalkDir`, if your callback returns `SkipDir`,
// GlobWalk will skip the current directory. This means that if the current
// path _is_ a directory, GlobWalk will not recurse into it. If the current
// path is not a directory, the rest of the parent directory will be skipped.
//
// GlobWalk ignores file system errors such as I/O errors reading directories.
// GlobWalk may return ErrBadPattern, reporting that the pattern is malformed.
// Additionally, if the callback function `fn` returns an error, GlobWalk will
// exit immediately and return that error.
//
// Like Glob(), this function assumes that your pattern uses `/` as the path
// separator even if that's not correct for your OS (like Windows). If you
// aren't sure if that's the case, you can use filepath.ToSlash() on your
// pattern before calling GlobWalk().
//
// Note: users should _not_ count on the returned error,
// doublestar.ErrBadPattern, being equal to path.ErrBadPattern.
//
func GlobWalk(fsys fs.FS, pattern string, fn GlobWalkFunc) error {
	if !ValidatePattern(pattern) {
		return ErrBadPattern
	}
	return doGlobWalk(fsys, pattern, true, fn)
}

// Actually execute GlobWalk
func doGlobWalk(fsys fs.FS, pattern string, firstSegment bool, fn GlobWalkFunc) error {
	patternStart := indexMeta(pattern)
	if patternStart == -1 {
		// pattern doesn't contain any meta characters - does a file matching the
		// pattern exist?
		// The pattern may contain escaped wildcard characters for an exact path match.
		path := unescapeMeta(pattern)
		info, err := fs.Stat(fsys, path)
		if err == nil {
			err = fn(path, dirEntryFromFileInfo(info))
			if err == SkipDir {
				err = nil
			}
			return err
		} else {
			// ignore IO errors
			return nil
		}
	}

	dir := "."
	splitIdx := lastIndexSlashOrAlt(pattern)
	if splitIdx != -1 {
		if pattern[splitIdx] == '}' {
			openingIdx := indexMatchedOpeningAlt(pattern[:splitIdx])
			if openingIdx == -1 {
				// if there's no matching opening index, technically Match() will treat
				// an unmatched `}` as nothing special, so... we will, too!
				splitIdx = lastIndexSlash(pattern[:splitIdx])
			} else {
				// otherwise, we have to handle the alts:
				return globAltsWalk(fsys, pattern, openingIdx, splitIdx, firstSegment, fn)
			}
		}

		dir = pattern[:splitIdx]
		pattern = pattern[splitIdx+1:]
	}

	// if `splitIdx` is less than `patternStart`, we know `dir` has no meta
	// characters. They would be equal if they are both -1, which means `dir`
	// will be ".", and we know that doesn't have meta characters either.
	if splitIdx <= patternStart {
		return globDirWalk(fsys, dir, pattern, firstSegment, fn)
	}

	return doGlobWalk(fsys, dir, false, func(p string, d fs.DirEntry) error {
		if err := globDirWalk(fsys, p, pattern, firstSegment, fn); err != nil {
			return err
		}
		return nil
	})
}

// handle alts in the glob pattern - `openingIdx` and `closingIdx` are the
// indexes of `{` and `}`, respectively
func globAltsWalk(fsys fs.FS, pattern string, openingIdx, closingIdx int, firstSegment bool, fn GlobWalkFunc) (err error) {
	var matches []DirEntryWithFullPath
	startIdx := 0
	afterIdx := closingIdx + 1
	splitIdx := lastIndexSlashOrAlt(pattern[:openingIdx])
	if splitIdx == -1 || pattern[splitIdx] == '}' {
		// no common prefix
		matches, err = doGlobAltsWalk(fsys, "", pattern, startIdx, openingIdx, closingIdx, afterIdx, firstSegment, matches)
		if err != nil {
			return
		}
	} else {
		// our alts have a common prefix that we can process first
		startIdx = splitIdx + 1
		err = doGlobWalk(fsys, pattern[:splitIdx], false, func(p string, d fs.DirEntry) (e error) {
			matches, e = doGlobAltsWalk(fsys, p, pattern, startIdx, openingIdx, closingIdx, afterIdx, firstSegment, matches)
			return e
		})
		if err != nil {
			return
		}
	}

	skip := ""
	for _, m := range matches {
		if skip != "" {
			// Because matches are sorted, we know that descendants of the skipped
			// item must come immediately after the skipped item. If we find an item
			// that does not have a prefix matching the skipped item, we know we're
			// done skipping. I'm using strings.HasPrefix here because
			// filepath.HasPrefix has been marked deprecated (and just calls
			// strings.HasPrefix anyway). The reason it's deprecated is because it
			// doesn't handle case-insensitive paths, nor does it guarantee that the
			// prefix is actually a parent directory. Neither is an issue here: the
			// paths come from the system so their cases will match, and we guarantee
			// a parent directory by appending a slash to the prefix.
			//
			// NOTE: m.Path will always use slashes as path separators.
			if strings.HasPrefix(m.Path, skip) {
				continue
			}
			skip = ""
		}
		if err = fn(m.Path, m.Entry); err != nil {
			if err == SkipDir {
				if isDir(fsys, "", m.Path, m.Entry) {
					// append a slash to guarantee `skip` will be treated as a parent dir
					skip = m.Path + "/"
				} else {
					// Dir() calls Clean() which calls FromSlash(), so we need to convert
					// back to slashes
					skip = filepath.ToSlash(filepath.Dir(m.Path)) + "/"
				}
				err = nil
				continue
			}
			return
		}
	}

	return
}

// runs actual matching for alts
func doGlobAltsWalk(fsys fs.FS, d, pattern string, startIdx, openingIdx, closingIdx, afterIdx int, firstSegment bool, m []DirEntryWithFullPath) (matches []DirEntryWithFullPath, err error) {
	matches = m
	matchesLen := len(m)
	patIdx := openingIdx + 1
	for patIdx < closingIdx {
		nextIdx := indexNextAlt(pattern[patIdx:closingIdx], true)
		if nextIdx == -1 {
			nextIdx = closingIdx
		} else {
			nextIdx += patIdx
		}

		alt := buildAlt(d, pattern, startIdx, openingIdx, patIdx, nextIdx, afterIdx)
		err = doGlobWalk(fsys, alt, firstSegment, func(p string, d fs.DirEntry) error {
			// insertion sort, ignoring dups
			insertIdx := matchesLen
			for insertIdx > 0 && matches[insertIdx-1].Path > p {
				insertIdx--
			}
			if insertIdx > 0 && matches[insertIdx-1].Path == p {
				// dup
				return nil
			}

			// append to grow the slice, then insert
			entry := DirEntryWithFullPath{d, p}
			matches = append(matches, entry)
			for i := matchesLen; i > insertIdx; i-- {
				matches[i] = matches[i-1]
			}
			matches[insertIdx] = entry
			matchesLen++

			return nil
		})
		if err != nil {
			return
		}

		patIdx = nextIdx + 1
	}

	return
}

func globDirWalk(fsys fs.FS, dir, pattern string, canMatchFiles bool, fn GlobWalkFunc) (e error) {
	if pattern == "" {
		// pattern can be an empty string if the original pattern ended in a slash,
		// in which case, we should just return dir, but only if it actually exists
		// and it's a directory (or a symlink to a directory)
		info, err := fs.Stat(fsys, dir)
		if err != nil || !info.IsDir() {
			return nil
		}

		e = fn(dir, dirEntryFromFileInfo(info))
		if e == SkipDir {
			e = nil
		}
		return
	}

	if pattern == "**" {
		// `**` can match *this* dir
		info, err := fs.Stat(fsys, dir)
		if err != nil || !info.IsDir() {
			return nil
		}
		if e = fn(dir, dirEntryFromFileInfo(info)); e != nil {
			if e == SkipDir {
				e = nil
			}
			return
		}
		return globDoubleStarWalk(fsys, dir, canMatchFiles, fn)
	}

	dirs, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// ignore IO errors
		return nil
	}

	var matched bool
	for _, info := range dirs {
		name := info.Name()
		if canMatchFiles || isDir(fsys, dir, name, info) {
			matched, e = matchWithSeparator(pattern, name, '/', false)
			if e != nil {
				return
			}
			if matched {
				if e = fn(path.Join(dir, name), info); e != nil {
					if e == SkipDir {
						e = nil
					}
					return
				}
			}
		}
	}

	return
}

// recursively walk files/directories in a directory
func globDoubleStarWalk(fsys fs.FS, dir string, canMatchFiles bool, fn GlobWalkFunc) (e error) {
	dirs, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// ignore IO errors
		return
	}

	// `**` can match *this* dir, so add it
	for _, info := range dirs {
		name := info.Name()
		if isDir(fsys, dir, name, info) {
			p := path.Join(dir, name)
			if e = fn(p, info); e != nil {
				if e == SkipDir {
					e = nil
					continue
				}
				return
			}
			if e = globDoubleStarWalk(fsys, p, canMatchFiles, fn); e != nil {
				return
			}
		} else if canMatchFiles {
			if e = fn(path.Join(dir, name), info); e != nil {
				if e == SkipDir {
					e = nil
				}
				return
			}
		}
	}

	return
}

type DirEntryFromFileInfo struct {
	fi fs.FileInfo
}

func (d *DirEntryFromFileInfo) Name() string {
	return d.fi.Name()
}

func (d *DirEntryFromFileInfo) IsDir() bool {
	return d.fi.IsDir()
}

func (d *DirEntryFromFileInfo) Type() fs.FileMode {
	return d.fi.Mode().Type()
}

func (d *DirEntryFromFileInfo) Info() (fs.FileInfo, error) {
	return d.fi, nil
}

func dirEntryFromFileInfo(fi fs.FileInfo) fs.DirEntry {
	return &DirEntryFromFileInfo{fi}
}

type DirEntryWithFullPath struct {
	Entry fs.DirEntry
	Path  string
}
