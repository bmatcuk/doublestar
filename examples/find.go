package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// To run:
// $ go run find.go <glob-pattern>
//
// For example:
// $ go run find.go '/usr/bin/*
//
// Make sure to escape the pattern as necessary for your shell, otherwise the
// shell will expand the pattern! Additionally, you should use `/` as the path
// separator even if your OS (like Windows) does not!
//
// Patterns that include `.` or `..` after any meta characters (*, ?, [, or {)
// will not work because io/fs will reject them. If they appear _before_ any
// meta characters _and_ before a `/`, the `splitPattern` function below will
// take care of them correctly.

func main() {
	pattern := os.Args[1]
	fmt.Printf("Searching on disk for pattern: %s\n\n", pattern)

	var basepath string
	basepath, pattern = doublestar.SplitPattern(pattern)
	fsys := os.DirFS(basepath)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	fmt.Printf(strings.Join(matches, "\n"))
	fmt.Print("\n\n")
	fmt.Printf("Found %d items.\n", len(matches))
}
