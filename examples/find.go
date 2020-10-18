package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v2"
)

// To run:
// $ go run find.go <glob-pattern>

// For example:
// $ go run find.go '/usr/bin/*' 			# Make sure to escape as necessary for your shell

func main() {
	pattern := os.Args[1]
	fmt.Printf("Searching on disk for pattern: %s\n\n", pattern)

	matches, err := doublestar.Glob(pattern)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	fmt.Printf(strings.Join(matches, "\n"))
	fmt.Print("\n\n")
	fmt.Printf("Found %d items.\n", len(matches))
}
