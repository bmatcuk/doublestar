![Release](https://img.shields.io/github/release/bmatcuk/doublestar.svg?branch=master)
[![Build Status](https://travis-ci.org/bmatcuk/doublestar.svg?branch=master)](https://travis-ci.org/bmatcuk/doublestar)
[![codecov.io](https://img.shields.io/codecov/c/github/bmatcuk/doublestar.svg?branch=master)](https://codecov.io/github/bmatcuk/doublestar?branch=master)

# doublestar

**doublestar** is a [golang](http://golang.org/) implementation of path pattern matching and globbing with support for "doublestar" (aka globstar: `**`) patterns.

doublestar patterns match files and directories recursively. For example, if you had the following directory structure:

```
grandparent
`-- parent
    |-- child1
    `-- child2
```

You could find the children with patterns such as: `**/child*`, `grandparent/**/child?`, `**/parent/*`, or even just `**` by itself (which will return all files and directories recursively).

## Installation

**doublestar** can be installed via `go get`:

```bash
go get github.com/bmatcuk/doublestar
```

To use it in your code, you must import it:

```go
import "github.com/bmatcuk/doublestar"
```

## Functions

### Match
```go
func Match(pattern, name string) (bool, error)
```

Match returns true if `name` matches the file name `pattern` ([see below](#patterns)). `name` and `pattern` are split on forward slash (`/`) characters and may be relative or absolute.

### PathMatch
```go
func PathMatch(pattern, name string) (bool, error)
```

PathMatch returns true  if `name` matches the file name `pattern` ([see below](#patterns)). The difference between Match and PathMatch is that PathMatch will automatically use your system's path separator to split `name` and `pattern`.

### PathGlob
```go
func Glob(pattern string) ([]string, error)
```

Glob finds all files and directories in the filesystem that match `pattern` ([see below](#patterns)). `pattern` may be relative (to the current working directory), or absolute.

### GlobFrom
```go
func GlobFrom(basedir, pattern string) ([]string, error)
```

GlobFrom finds all files and directories in the filesystem that match `pattern` ([see below](#patterns)).
If pattern is a relative path, it will be searched relative to `basedir`.

### PathGlobFrom
```go
func PathGlob(basedir, pattern string) ([]string, error)
```

PathGlobFrom finds all files and directories in the filesystem that match `pattern` ([see below](#patterns)).
If pattern is a relative path, it will be searched relative to `basedir`.
The difference between GlobFrom and PathGlobFrom is that PathGlobFrom will automatically use your system's path separator to split the `pattern`.

## Patterns

**doublestar** supports the following special terms in the patterns:

Special Terms | Meaning
------------- | -------
`*`           | matches any sequence of non-path-separators
`**`          | matches any sequence of characters, including path separators
`?`           | matches any single non-path-separator character
`[class]`     | matches any single non-path-separator character against a class of characters ([see below](#character-classes))
`{alt1,...}`  | matches a sequence of characters if one of the comma-separated alternatives matches

Any character with a special meaning can be escaped with a backslash (`\`).

### Character Classes

Character classes support the following:

Class      | Meaning
---------- | -------
`[abc]`    | matches any single character within the set
`[a-z]`    | matches any single character in the range
`[^class]` | matches any single character which does *not* match the class

