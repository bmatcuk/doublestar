package doublestar

// glob is an internal type to store options during globbing.
type glob struct {
	failOnIOErrors bool
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
// FilepathGlob. If passed, it enables aborting and returning the error when an
// IO error is encountered.
//
func WithFailOnIOErrors() GlobOption {
	return func(g *glob) {
		g.failOnIOErrors = true
	}
}

// forwardErrIfFailOnIOErrors is used to wrap the return values of I/O
// functions. When failOnIOErrors is enabled, it will return err; otherwise, it
// always returns nil.
func (g *glob) forwardErrIfFailOnIOErrors(err error) error {
	if g.failOnIOErrors {
		return err
	}
	return nil
}

func (g *glob) GoString() string {
	if g.failOnIOErrors {
		return "opts: WithFailOnIOErrors"
	}
	return "opts: nil"
}
