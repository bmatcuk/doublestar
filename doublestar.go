package doublestar

import (
  "path"
  "strings"
  "unicode/utf8"
)

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
  return MatchWithSeparator(pattern, name, '/')
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
func MatchWithSeparator(pattern, name string, separator rune) (bool, error) {
  patternComponents := SplitPathOnSeparator(pattern, separator)
  nameComponents := SplitPathOnSeparator(name, separator)
  return doMatching(patternComponents, nameComponents)
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
	return false, path.ErrBadPattern
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
      if endClass == -1 { return false, path.ErrBadPattern }
      endClass += patIdx
      classRunes := []rune(pattern[patIdx:endClass])
      classRunesLen := len(classRunes)
      if classRunesLen > 0 {
	classIdx := 0
	matchClass := false
	if classRunes[0] == '^' { classIdx++ }
	for classIdx < classRunesLen {
	  low := classRunes[classIdx]
	  if low == '-' { return false, path.ErrBadPattern }
	  classIdx++
	  if low == '\\' {
	    if classIdx < classRunesLen {
	      low = classRunes[classIdx]
	      classIdx++
	    } else {
	      return false, path.ErrBadPattern
	    }
	  }
	  high := low
	  if classIdx < classRunesLen && classRunes[classIdx] == '-' {
	    if classIdx++; classIdx >= classRunesLen { return false, path.ErrBadPattern }
	    high = classRunes[classIdx]
	    if high == '-' { return false, path.ErrBadPattern }
	    classIdx++
	    if high == '\\' {
	      if classIdx < classRunesLen {
		high = classRunes[classIdx]
		classIdx++
	      } else {
		return false, path.ErrBadPattern
	      }
	    }
	  }
	  if low <= nameRune && nameRune <= high { matchClass = true }
	}
	if matchClass == (classRunes[0] == '^') { return false, nil }
      } else {
	return false, path.ErrBadPattern
      }
      patIdx = endClass + 1
      nameIdx += nameAdj
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

