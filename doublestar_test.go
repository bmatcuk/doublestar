// This file is mostly copied from Go's path/match_test.go

package doublestar

import (
  "testing"
  "path"
)

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
  {"[]a]", "]", false, path.ErrBadPattern},
  {"[-]", "-", false, path.ErrBadPattern},
  {"[x-]", "x", false, path.ErrBadPattern},
  {"[x-]", "-", false, path.ErrBadPattern},
  {"[x-]", "z", false, path.ErrBadPattern},
  {"[-x]", "x", false, path.ErrBadPattern},
  {"[-x]", "-", false, path.ErrBadPattern},
  {"[-x]", "a", false, path.ErrBadPattern},
  {"\\", "a", false, path.ErrBadPattern},
  {"[a-b-c]", "a", false, path.ErrBadPattern},
  {"[", "a", false, path.ErrBadPattern},
  {"[^", "a", false, path.ErrBadPattern},
  {"[^bc", "a", false, path.ErrBadPattern},
  {"a[", "a", false, nil},
  {"a[", "ab", false, path.ErrBadPattern},
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
}

func TestMatch(t *testing.T) {
  for idx, tt := range matchTests {
    ok, err := Match(tt.pattern, tt.s)
    if ok != tt.match || err != tt.err {
      t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, tt.pattern, tt.s, ok, err, tt.match, tt.err)
    }
  }
}

