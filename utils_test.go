package doublestar

import (
	"testing"
)

type FilepathGlobTest struct {
	pattern, result string
}

var filepathGlobTests = []FilepathGlobTest{
	{".", "."},
	{"././.", "."},
	{"..", ".."},
	{"../.", ".."},
	{".././././", ".."},
	{"../..", "../.."},
	{"/", "/"},
	{"./", "."},
	{"/.", "/"},
	{"/././././", "/"},
}

func TestSpecialFilepathGlobCases(t *testing.T) {
	for idx, tt := range filepathGlobTests {
		testSpecialFilepathGlobCasesWith(t, idx, tt)
	}
}

func testSpecialFilepathGlobCasesWith(t *testing.T, idx int, tt FilepathGlobTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. FilepathGlob(%#q) panicked with: %#v", idx, tt.pattern, r)
		}
	}()

	matches, err := FilepathGlob(tt.pattern)
	if err != nil {
		t.Errorf("#%v. FilepathGlob(%#q) has error %v", idx, tt.pattern, err)
	}
	if len(matches) != 1 || matches[0] != tt.result {
		t.Errorf("#%v. FilepathGlob(%#q) = %#v - should be []string{%#v}", idx, tt.pattern, matches, tt.result)
	}
}
