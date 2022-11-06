package doublestar

import (
	"testing"
	"path/filepath"
)

var filepathGlobTests = []string{
	".",
	"././.",
	"..",
	"../.",
	".././././",
	"../..",
	"/",
	"./",
	"/.",
	"/././././",
}

func TestSpecialFilepathGlobCases(t *testing.T) {
	for idx, pattern := range filepathGlobTests {
		testSpecialFilepathGlobCasesWith(t, idx, pattern)
	}
}

func testSpecialFilepathGlobCasesWith(t *testing.T, idx int, pattern string) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. FilepathGlob(%#q) panicked with: %#v", idx, pattern, r)
		}
	}()

	pattern = filepath.FromSlash(pattern)
	matches, err := FilepathGlob(pattern)
	if err != nil {
		t.Errorf("#%v. FilepathGlob(%#q) has error %v", idx, pattern, err)
	}
	if len(matches) != 1 {
		t.Errorf("#%v. FilepathGlob(%#q) should have 1 result but has %v", idx, pattern, len(matches))
	}

	results, err := filepath.Glob(pattern)
	if len(results) != 1 {
		t.Errorf("#%v. filepath.Glob(%#q) should have 1 result but has %v", idx, pattern, len(results))
	}
	if matches[0] != results[0] {
		t.Errorf("#%v. FilepathGlob(%#q) = %#v - should be %#v", idx, pattern, matches, results)
	}
}
