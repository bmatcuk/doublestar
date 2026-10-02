package doublestar

import (
	"fmt"
	"path/filepath"
)

// Validate a pattern. Patterns are validated while they run in Match(),
// PathMatch(), and Glob(), so, you normally wouldn't need to call this.
// However, there are cases where this might be useful: for example, if your
// program allows a user to enter a pattern that you'll run at a later time,
// you might want to validate it.
//
// ValidatePattern assumes your pattern uses '/' as the path separator.
func ValidatePattern(s string) bool {
	return ParsePattern(s) == nil
}

// Like ValidatePattern, only uses your OS path separator. In other words, use
// ValidatePattern if you would normally use Match() or Glob(). Use
// ValidatePathPattern if you would normally use PathMatch(). Keep in mind,
// Glob() requires '/' separators, even if your OS uses something else.
func ValidatePathPattern(s string) bool {
	return ParsePathPattern(s) == nil
}

// Like ValidatePattern, only instead of a bool, returns nil if the pattern is
// valid, or an error describing the problem (including the byte offset where it
// was found) if it is not. The returned error wraps ErrBadPattern, so
// errors.Is(err, ErrBadPattern) will report true.
//
// ParsePattern assumes your pattern uses '/' as the path separator.
func ParsePattern(s string) error {
	return doValidatePattern(s, '/')
}

// Like ParsePattern, only uses your OS path separator. In other words, use
// ParsePattern if you would normally use Match() or Glob(). Use
// ParsePathPattern if you would normally use PathMatch(). Keep in mind, Glob()
// requires '/' separators, even if your OS uses something else. The returned
// error wraps ErrBadPattern.
func ParsePathPattern(s string) error {
	return doValidatePattern(s, filepath.Separator)
}

func doValidatePattern(s string, separator rune) error {
	// offsets of the '{' characters that are still open
	var altStarts []int
	l := len(s)
VALIDATE:
	for i := 0; i < l; i++ {
		switch s[i] {
		case '\\':
			if separator != '\\' {
				// skip the next byte - error if there is no next byte
				if i++; i >= l {
					return fmt.Errorf("%w: trailing escape character at offset %d", ErrBadPattern, i-1)
				}
			}
			continue

		case '[':
			start := i
			if i++; i >= l {
				// class didn't end
				return fmt.Errorf("%w: unclosed character class starting at offset %d", ErrBadPattern, start)
			}
			if s[i] == '^' || s[i] == '!' {
				i++
			}
			if i >= l {
				// class didn't end
				return fmt.Errorf("%w: unclosed character class starting at offset %d", ErrBadPattern, start)
			}
			if s[i] == ']' {
				return fmt.Errorf("%w: empty character class at offset %d", ErrBadPattern, start)
			}

			for ; i < l; i++ {
				if separator != '\\' && s[i] == '\\' {
					i++
				} else if s[i] == ']' {
					// looks good
					continue VALIDATE
				}
			}

			// class didn't end
			return fmt.Errorf("%w: unclosed character class starting at offset %d", ErrBadPattern, start)

		case '{':
			altStarts = append(altStarts, i)
			continue

		case '}':
			if len(altStarts) == 0 {
				// alt end without a corresponding start
				return fmt.Errorf("%w: unmatched '}' at offset %d", ErrBadPattern, i)
			}
			altStarts = altStarts[:len(altStarts)-1]
			continue
		}
	}

	// valid as long as all alts are closed
	if len(altStarts) > 0 {
		return fmt.Errorf("%w: unclosed '{' at offset %d", ErrBadPattern, altStarts[len(altStarts)-1])
	}
	return nil
}
