// This file is mostly copied from Go's path/match_test.go

package doublestar

import "testing"

type MatchTest struct {
	pattern, s string
	match      bool
	err        error
}

var matchTests = []MatchTest{
  {"abc", "abc", true, nil},
  {"*", "abc", true, nil},
  {"*c", "abc", true, nil},
  {"a*", "a", true, nil},
  {"a*", "abc", true, nil},
  {"a*", "ab/c", false, nil},
  {"a*/b", "abc/b", true, nil},
  {"a*/b", "a/c/b", false, nil},
  {"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
  {"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
  {"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
  {"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
  {"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
  {"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
  {"ab[c]", "abc", true, nil},
  {"ab[b-d]", "abc", true, nil},
  {"ab[e-g]", "abc", false, nil},
  {"ab[^c]", "abc", false, nil},
  {"ab[^b-d]", "abc", false, nil},
  {"ab[^e-g]", "abc", true, nil},
  {"a\\*b", "a*b", true, nil},
  {"a\\*b", "ab", false, nil},
  {"a?b", "a☺b", true, nil},
  {"a[^a]b", "a☺b", true, nil},
  {"a???b", "a☺b", false, nil},
  {"a[^a][^a][^a]b", "a☺b", false, nil},
  {"[a-ζ]*", "α", true, nil},
  {"*[a-ζ]", "A", false, nil},
  {"a?b", "a/b", false, nil},
  {"a*b", "a/b", false, nil},
  {"[\\]a]", "]", true, nil},
  {"[\\-]", "-", true, nil},
  {"[x\\-]", "x", true, nil},
  {"[x\\-]", "-", true, nil},
  {"[x\\-]", "z", false, nil},
  {"[\\-x]", "x", true, nil},
  {"[\\-x]", "-", true, nil},
  {"[\\-x]", "a", false, nil},
  {"[]a]", "]", false, ErrBadPattern},
  {"[-]", "-", false, ErrBadPattern},
  {"[x-]", "x", false, ErrBadPattern},
  {"[x-]", "-", false, ErrBadPattern},
  {"[x-]", "z", false, ErrBadPattern},
  {"[-x]", "x", false, ErrBadPattern},
  {"[-x]", "-", false, ErrBadPattern},
  {"[-x]", "a", false, ErrBadPattern},
  {"\\", "a", false, ErrBadPattern},
  {"[a-b-c]", "a", false, ErrBadPattern},
  {"[", "a", false, ErrBadPattern},
  {"[^", "a", false, ErrBadPattern},
  {"[^bc", "a", false, ErrBadPattern},
  {"a[", "a", false, nil},
  {"a[", "ab", false, ErrBadPattern},
  {"*x", "xxx", true, nil},
  {"a/**", "a", false, nil},
  {"a/**", "a/b", true, nil},
  {"a/**", "a/b/c", true, nil},
  {"**/c", "c", true, nil},
  {"**/c", "b/c", true, nil},
  {"**/c", "a/b/c", true, nil},
  {"a/**/b", "a/b", true, nil},
  {"a/**/c", "a/b/c", true, nil},
  {"a/**/d", "a/b/c/d", true, nil},
  {"a/\\**", "a/b/c", false, nil},
  {"a/\\**", "a/*", true, nil},
  {"ab{c,d}", "abc", true, nil},
  {"ab{c,d,*}", "abcde", true, nil},
  {"ab{c,d}[", "abcd", false, ErrBadPattern},
}

func TestMatch(t *testing.T) {
  for idx, tt := range matchTests {
    ok, err := Match(tt.pattern, tt.s)
    if ok != tt.match || err != tt.err {
      t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, tt.pattern, tt.s, ok, err, tt.match, tt.err)
    }
  }
}
