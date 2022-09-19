package doublestar

// glob is an internal type to store options during globbing.
type glob struct {
	failOnIOErrors bool
}

// Opt represents a setting that can be passed to GlobWalk and Glob.
type Opt func(*glob)

func newGlob(opts ...Opt) *glob {
	g := glob{}
	g.applyOpts(opts)
	return &g
}

func (g *glob) applyOpts(opts []Opt) {
	for _, o := range opts {
		o(g)
	}
}

// WithFailOnIOErrors is an option that can be passed to Glob, GlobWalk and
// FilepathGlob. If passed, it enables aborting and returning the error when an
// I/O error is encountered.
// When this option is not passed, I/O errors cause that files or directories
// are skipped and globbing continues.
func WithFailOnIOErrors() Opt {
	return func(g *glob) {
		g.failOnIOErrors = true
	}
}

// forwardErrIfFailOnIOErrors is used to wrap the return values of I/O
// functions.
// When WithFailOnIOErrors enabled, it return err, otherwise it always returns
// nil.
func (g *glob) forwardErrIfFailOnIOErrors(err error) error {
	if g.failOnIOErrors {
		return err
	}
	return nil
}
