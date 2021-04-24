package doublestar

// Validate a pattern. Patterns are validated while they run in Match(),
// PathMatch(), and Glob(), so, you normally wouldn't need to call this.
// However, there are cases where this might be useful: for example, if your
// program allows a user to enter a pattern that you'll run at a later time,
// you might want to validate it.
//
func ValidatePattern(s string) bool {
	altDepth := 0
	l := len(s)
VALIDATE:
	for i := 0; i < l; i++ {
		switch s[i] {
		case '\\':
			// skip the next byte - return false if there is no next byte
			if i++; i >= l {
				return false
			}
			continue

		case '[':
			if i++; i >= l {
				// class didn't end
				return false
			}
			if s[i] == '^' || s[i] == '!' {
				i++
			}
			if i >= l || s[i] == ']' {
				// class didn't end or empty character class
				return false
			}

			for ; i < l; i++ {
				if s[i] == ']' {
					// looks good
					continue VALIDATE
				}
			}

			// class didn't end
			return false

		case '{':
			altDepth++
			continue

		case '}':
			if altDepth == 0 {
				// alt end without a corresponding start
				return false
			}
			altDepth--
			continue
		}
	}

	// valid as long as all alts are closed
	return altDepth == 0
}
