package doublestar

import "testing"

func TestBraceAlternationPreservesSegmentStart(t *testing.T) {
	for _, tc := range []struct {
		pattern, expanded, name string
	}{
		{"x{**/a,b}", "x**/a", "xy/z/a"},
		{"x{b,**/a}", "x**/a", "xy/z/a"},
		{"x/{**/a,b}", "x/**/a", "x/y/z/a"},
	} {
		want, err := Match(tc.expanded, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Match(tc.pattern, tc.name)
		if err != nil || got != want {
			t.Errorf("Match(%q, %q) = %v, %v; expanded pattern gives %v", tc.pattern, tc.name, got, err, want)
		}
		if got := MatchUnvalidated(tc.pattern, tc.name); got != want {
			t.Errorf("MatchUnvalidated(%q, %q) = %v, want %v", tc.pattern, tc.name, got, want)
		}
	}
}
