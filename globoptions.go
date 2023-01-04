package doublestar

import "strings"

// glob is an internal type to store options during globbing.
type glob struct {
	failOnIOErrors        bool
	failOnPatternNotExist bool
	filesOnly             bool
}

// GlobOption represents a setting that can be passed to Glob, GlobWalk, and
// FilepathGlob.
type GlobOption func(*glob)

// Construct a new glob object with the given options
func newGlob(opts ...GlobOption) *glob {
	g := &glob{}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// WithFailOnIOErrors is an option that can be passed to Glob, GlobWalk, or
// FilepathGlob. If passed, doublestar will abort and return IO errors when
// encountered. Note that if the glob pattern references a path that does not
// exist (such as `nonexistent/path/*`), this is _not_ considered an IO error:
// it is considered a pattern with no matches.
//
func WithFailOnIOErrors() GlobOption {
	return func(g *glob) {
		g.failOnIOErrors = true
	}
}

// WithFailOnPatternNotExist is an option that can be passed to Glob, GlobWalk,
// or FilepathGlob. If passed, doublestar will abort and return
// ErrPatternNotExist if the pattern references a path that does not exist
// before any meta charcters such as `nonexistent/path/*`. Note that alts (ie,
// `{...}`) are expanded before this check. In other words, a pattern such as
// `{a,b}/*` may fail if either `a` or `b` do not exist but `*/{a,b}` will
// never fail because the star may match nothing.
//
func WithFailOnPatternNotExist() GlobOption {
	return func(g *glob) {
		g.failOnPatternNotExist = true
	}
}

// WithFilesOnly is an option that can be passed to Glob, GlobWalk, or
// FilepathGlob. If passed, doublestar will only return files that match the
// pattern, not directories.
//
func WithFilesOnly() GlobOption {
	return func(g *glob) {
		g.filesOnly = true
	}
}

// forwardErrIfFailOnIOErrors is used to wrap the return values of I/O
// functions. When failOnIOErrors is enabled, it will return err; otherwise, it
// always returns nil.
//
func (g *glob) forwardErrIfFailOnIOErrors(err error) error {
	if g.failOnIOErrors {
		return err
	}
	return nil
}

// handleErrNotExist handles fs.ErrNotExist errors. If
// WithFailOnPatternNotExist has been enabled and canFail is true, this will
// return ErrPatternNotExist. Otherwise, it will return nil.
//
func (g *glob) handlePatternNotExist(canFail bool) error {
	if canFail && g.failOnPatternNotExist {
		return ErrPatternNotExist
	}
	return nil
}

// Format options for debugging/testing purposes
func (g *glob) GoString() string {
	var b strings.Builder
	b.WriteString("opts: ")

	hasOpts := false
	if (g.failOnIOErrors) {
		b.WriteString("WithFailOnIOErrors")
		hasOpts = true
	}
	if (g.failOnPatternNotExist) {
		if hasOpts {
			b.WriteString(", ")
		}
		b.WriteString("WithFailOnPatternNotExist")
		hasOpts = true
	}
	if (g.filesOnly) {
		if hasOpts {
			b.WriteString(", ")
		}
		b.WriteString("WithFilesOnly")
		hasOpts = true
	}

	if !hasOpts {
		b.WriteString("nil")
	}
	return b.String()
}
