package doublestar

import (
  "path"
  "os"
  "strings"
  "unicode/utf8"
)

var ErrBadPattern = path.ErrBadPattern

func SplitPathOnSeparator(path string, separator rune) []string {
  // if the separator is '\\', then we can just split...
  if separator == '\\' { return strings.Split(path, string(separator)) }

  // otherwise, we need to be careful of situations where the separator was escaped
  cnt := strings.Count(path, string(separator))
  if cnt == 0 { return []string{path} }
  ret := make([]string, cnt + 1)
  pathlen := len(path)
  separatorLen := utf8.RuneLen(separator)
  idx := 0
  for start := 0; start < pathlen; {
    end := indexRuneWithEscaping(path[start:], separator)
    if end == -1 {
      end = pathlen
    } else {
      end += start
    }
    ret[idx] = path[start:end]
    start = end + separatorLen
    idx++
  }
  return ret[:idx]
}

func indexRuneWithEscaping(s string, r rune) int {
  end := strings.IndexRune(s, r)
  if end == -1 { return -1 }
  if end > 0 && s[end - 1] == '\\' {
    start := end + utf8.RuneLen(r)
    end = indexRuneWithEscaping(s[start:], r)
    if end != -1 { end += start }
  }
  return end
}

// Match returns true if name matches the shell file name pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-path-separators
//              '**'        matches any sequence of characters, including
//                          path separators.
//		'?'         matches any single non-path-separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		'{' { term } [ ',' { term } ... ] '}'
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Match requires pattern to match all of name, not just a substring.
// The path-separator defaults to the '/' character. The only possible
// returned error is ErrBadPattern, when pattern is malformed.
//
func Match(pattern, name string) (bool, error) {
  return matchWithSeparator(pattern, name, '/')
}

// PathMatch is like Match except that it uses your system's path separator.
// For most systems, this will be '/'. However, for Windows, it would be '\\'.
// Note that for systems where the path separator is '\\', escaping is
// disabled.
//
func PathMatch(pattern, name string) (bool, error) {
  return matchWithSeparator(pattern, name, os.PathSeparator)
}

// Match returns true if name matches the shell file name pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-path-separators
//              '**'        matches any sequence of characters, including
//                          path separators.
//		'?'         matches any single non-path-separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		'{' { term } [ ',' { term } ... ] '}'
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c, unless separator is '\\'
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Match requires pattern to match all of name, not just a substring.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
//
func matchWithSeparator(pattern, name string, separator rune) (bool, error) {
  patternComponents := SplitPathOnSeparator(pattern, separator)
  nameComponents := SplitPathOnSeparator(name, separator)
  return doMatching(patternComponents, nameComponents)
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of pattern is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
//
// Your system path separator is automatically used. This means on
// systems where the separator is '\\' (Windows), escaping will be
// disabled.
//
func Glob(pattern string) (matches []string, err error) {
  patternComponents := SplitPathOnSeparator(pattern, os.PathSeparator)
  patternLen := len(patternComponents)
  if patternLen == 0 { return false, nil }
  patIdx := 0
  for ; patIdx < patternLen; {
  }
}

func doMatching(patternComponents, nameComponents []string) (matched bool, err error) {
  patternLen, nameLen := len(patternComponents), len(nameComponents)
  if patternLen == 0 && nameLen == 0 { return true, nil }
  if patternLen == 0 || nameLen == 0 { return false, nil }
  patIdx, nameIdx := 0, 0
  for ; patIdx < patternLen && nameIdx < nameLen; {
    if patternComponents[patIdx] == "**" {
      if patIdx++; patIdx >= patternLen { return true, nil }
      for ; nameIdx < nameLen; nameIdx++ {
	if m, _ := doMatching(patternComponents[patIdx:], nameComponents[nameIdx:]); m {
	  return true, nil
	}
      }
      return false, nil
    } else {
      matched, err = matchComponent(patternComponents[patIdx], nameComponents[nameIdx])
      if !matched || err != nil { return }
    }
    patIdx++
    nameIdx++
  }
  return patIdx >= patternLen && nameIdx >= nameLen, nil
}

func matchComponent(pattern, name string) (bool, error) {
  patternLen, nameLen := len(pattern), len(name)
  if patternLen == 0 && nameLen == 0 { return true, nil }
  if patternLen == 0 { return false, nil }
  if nameLen == 0 && pattern != "*" { return false, nil }
  patIdx, nameIdx := 0, 0
  for ; patIdx < patternLen && nameIdx < nameLen; {
    patRune, patAdj := utf8.DecodeRuneInString(pattern[patIdx:])
    nameRune, nameAdj := utf8.DecodeRuneInString(name[nameIdx:])
    if patRune == '\\' {
      patIdx += patAdj
      patRune, patAdj = utf8.DecodeRuneInString(pattern[patIdx:])
      if patRune == utf8.RuneError {
	return false, ErrBadPattern
      } else if patRune == nameRune {
	patIdx += patAdj
	nameIdx += nameAdj
      } else {
	return false, nil
      }
    } else if patRune == '*' {
      if patIdx += patAdj; patIdx >= patternLen { return true, nil }
      for ; nameIdx < nameLen; nameIdx += nameAdj {
	if m, _ := matchComponent(pattern[patIdx:], name[nameIdx:]); m {
	  return true, nil
	}
      }
      return false, nil
    } else if patRune == '[' {
      patIdx += patAdj
      endClass := indexRuneWithEscaping(pattern[patIdx:], ']')
      if endClass == -1 { return false, ErrBadPattern }
      endClass += patIdx
      classRunes := []rune(pattern[patIdx:endClass])
      classRunesLen := len(classRunes)
      if classRunesLen > 0 {
	classIdx := 0
	matchClass := false
	if classRunes[0] == '^' { classIdx++ }
	for classIdx < classRunesLen {
	  low := classRunes[classIdx]
	  if low == '-' { return false, ErrBadPattern }
	  classIdx++
	  if low == '\\' {
	    if classIdx < classRunesLen {
	      low = classRunes[classIdx]
	      classIdx++
	    } else {
	      return false, ErrBadPattern
	    }
	  }
	  high := low
	  if classIdx < classRunesLen && classRunes[classIdx] == '-' {
	    if classIdx++; classIdx >= classRunesLen { return false, ErrBadPattern }
	    high = classRunes[classIdx]
	    if high == '-' { return false, ErrBadPattern }
	    classIdx++
	    if high == '\\' {
	      if classIdx < classRunesLen {
		high = classRunes[classIdx]
		classIdx++
	      } else {
		return false, ErrBadPattern
	      }
	    }
	  }
	  if low <= nameRune && nameRune <= high { matchClass = true }
	}
	if matchClass == (classRunes[0] == '^') { return false, nil }
      } else {
	return false, ErrBadPattern
      }
      patIdx = endClass + 1
      nameIdx += nameAdj
    } else if patRune == '{' {
      patIdx += patAdj
      endOptions := indexRuneWithEscaping(pattern[patIdx:], '}')
      if endOptions == -1 { return false, ErrBadPattern }
      endOptions += patIdx
      options := SplitPathOnSeparator(pattern[patIdx:endOptions], ',')
      patIdx = endOptions + 1
      for _, o := range options {
	m, e := matchComponent(o + pattern[patIdx:], name[nameIdx:])
	if e != nil { return false, e }
	if m { return true, nil }
      }
      return false, nil
    } else if patRune == '?' || patRune == nameRune {
      patIdx += patAdj
      nameIdx += nameAdj
    } else {
      return false, nil
    }
  }
  if patIdx >= patternLen && nameIdx >= nameLen { return true, nil }
  if nameIdx >= nameLen && pattern[patIdx:] == "*" { return true, nil }
  return false, nil
}

