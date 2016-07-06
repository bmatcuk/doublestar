// This file is mostly copied from Go's path/match_test.go

package doublestar

import (
  "testing"
  "path/filepath"
  "runtime"
)

type MatchTest struct {
  pattern, s string
  match      bool
  err        error
  testGlob   bool
}

var matchTests = []MatchTest{
  {"abc", "abc", true, nil, true},
  {"*", "abc", true, nil, true},
  {"*c", "abc", true, nil, true},
  {"a*", "a", true, nil, true},
  {"a*", "abc", true, nil, true},
  {"a*", "ab/c", false, nil, true},
  {"a*/b", "abc/b", true, nil, true},
  {"a*/b", "a/c/b", false, nil, true},
  {"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil, true},
  {"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil, true},
  {"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil, true},
  {"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil, true},
  {"a*b?c*x", "abxbbxdbxebxczzx", true, nil, true},
  {"a*b?c*x", "abxbbxdbxebxczzy", false, nil, true},
  {"ab[c]", "abc", true, nil, true},
  {"ab[b-d]", "abc", true, nil, true},
  {"ab[e-g]", "abc", false, nil, true},
  {"ab[^c]", "abc", false, nil, true},
  {"ab[^b-d]", "abc", false, nil, true},
  {"ab[^e-g]", "abc", true, nil, true},
  {"a\\*b", "ab", false, nil, true},
  {"a?b", "a☺b", true, nil, true},
  {"a[^a]b", "a☺b", true, nil, true},
  {"a???b", "a☺b", false, nil, true},
  {"a[^a][^a][^a]b", "a☺b", false, nil, true},
  {"[a-ζ]*", "α", true, nil, true},
  {"*[a-ζ]", "A", false, nil, true},
  {"a?b", "a/b", false, nil, true},
  {"a*b", "a/b", false, nil, true},
  {"[\\]a]", "]", true, nil, runtime.GOOS!="windows"},
  {"[\\-]", "-", true, nil, runtime.GOOS!="windows"},
  {"[x\\-]", "x", true, nil, runtime.GOOS!="windows"},
  {"[x\\-]", "-", true, nil, runtime.GOOS!="windows"},
  {"[x\\-]", "z", false, nil, runtime.GOOS!="windows"},
  {"[\\-x]", "x", true, nil, runtime.GOOS!="windows"},
  {"[\\-x]", "-", true, nil, runtime.GOOS!="windows"},
  {"[\\-x]", "a", false, nil, runtime.GOOS!="windows"},
  {"[]a]", "]", false, ErrBadPattern, true},
  {"[-]", "-", false, ErrBadPattern, true},
  {"[x-]", "x", false, ErrBadPattern, true},
  {"[x-]", "-", false, ErrBadPattern, true},
  {"[x-]", "z", false, ErrBadPattern, true},
  {"[-x]", "x", false, ErrBadPattern, true},
  {"[-x]", "-", false, ErrBadPattern, true},
  {"[-x]", "a", false, ErrBadPattern, true},
  {"\\", "a", false, ErrBadPattern, runtime.GOOS!="windows"},
  {"[a-b-c]", "a", false, ErrBadPattern, true},
  {"[", "a", false, ErrBadPattern, true},
  {"[^", "a", false, ErrBadPattern, true},
  {"[^bc", "a", false, ErrBadPattern, true},
  {"a[", "a", false, nil, false},
  {"a[", "ab", false, ErrBadPattern, true},
  {"*x", "xxx", true, nil, true},
  {"a/**", "a", false, nil, true},
  {"a/**", "a/b", true, nil, true},
  {"a/**", "a/b/c", true, nil, true},
  {"**/c", "c", true, nil, true},
  {"**/c", "b/c", true, nil, true},
  {"**/c", "a/b/c", true, nil, true},
  {"**/c", "a/b", false, nil, true},
  {"**/c", "abcd", false, nil, true},
  {"**/c", "a/abc", false, nil, true},
  {"a/**/b", "a/b", true, nil, true},
  {"a/**/c", "a/b/c", true, nil, true},
  {"a/**/d", "a/b/c/d", true, nil, true},
  {"a/\\**", "a/b/c", false, nil, runtime.GOOS!="windows"},
  {"ab{c,d}", "abc", true, nil, true},
  {"ab{c,d,*}", "abcde", true, nil, true},
  {"ab{c,d}[", "abcd", false, ErrBadPattern, true},
  {"img/**/*.jpg", "img/blank.jpg", true, nil, true},
  {"img/**/*.jpg", "img/wallpapers/wall3.jpg", true, nil, true},
  {"img/**/*.jpg", "img/wallpapers/big/bigwall.jpg", true, nil, true},
  {"img/**/*.jpg", "img/wallpapers/wall2.png", false, nil, true},
  {"img/**/*.jpg", "img/README.md", false, nil, true},
}

func TestMatch(t *testing.T) {
  for idx, tt := range matchTests {
    testMatchWith(t, idx, tt)
  }
}

func testMatchWith(t *testing.T, idx int, tt MatchTest) {
  defer func() {
    if r := recover(); r != nil {
      t.Errorf("#%v. Match(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.s, r)
    }
  }()

  ok, err := Match(tt.pattern, tt.s)
  if ok != tt.match || err != tt.err {
    t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, tt.pattern, tt.s, ok, err, tt.match, tt.err)
  }
}

func TestGlob(t *testing.T) {
  for idx, tt := range matchTests {
    if tt.testGlob {
      testGlobWith(t, idx, tt)
    }
  }
}

func testGlobWith(t *testing.T, idx int, tt MatchTest) {
  defer func() {
    if r := recover(); r != nil {
      t.Errorf("#%v. Glob(%#q) panicked: %#v", idx, tt.pattern, r)
    }
  }()

  matches, err := Glob(filepath.Join("test", tt.pattern))
  if inSlice("test/" + tt.s, matches) != tt.match {
    if tt.match {
      t.Errorf("#%v. Glob(%#q) = %#v - doesn't contain %v, but should", idx, tt.pattern, matches, tt.s)
    } else {
      t.Errorf("#%v. Glob(%#q) = %#v - contains %v, but shouldn't", idx, tt.pattern, matches, tt.s)
    }
  }
  if err != tt.err {
    t.Errorf("#%v. Glob(%#q) has error %v, but should be %v", idx, tt.pattern, err, tt.err)
  }
}

func TestGlobWindows(t *testing.T) {
  if runtime.GOOS != "windows" {
    t.Skip("Skip on non-Windows")
  }
  // make path absolute
  abs := func(in string) string {
    abs, _ := filepath.Abs(in)
    return abs
  }
  matchTests := []MatchTest{
    {"test\\a\\**", "test\\a\\b\\c\\d", true, nil, true},
    {"test\\a\\**", "test\\a\\abc", true, nil, true},
    {abs("test\\a\\**"), abs("test\\a\\b\\c\\d"), true, nil, true},
    {abs("test\\a\\**"), abs("test\\a\\abc"), true, nil, true},
    {"test\\img\\**\\*.jpg", "test\\img\\wallpapers\\big\\bigwall.jpg", true, nil, true},
    {"test\\img\\**\\*.jpg", "test\\img\\wallpapers\\wall1.jpg", true, nil, true},
    {"test\\img\\**\\*.jpg", "test\\img\\blank.jpg", true, nil, true},
    {abs("test\\img\\**\\*.jpg"), abs("test\\img\\wallpapers\\big\\bigwall.jpg"), true, nil, true},
    {abs("test\\img\\**\\*.jpg"), abs("test\\img\\wallpapers\\wall1.jpg"), true, nil, true},
    {abs("test\\img\\**\\*.jpg"), abs("test\\img\\blank.jpg"), true, nil, true},
    {abs("test\\img\\**\\*.jpg"), abs("test\\img\\wallpapers\\wall2.png"), false, nil, true},
  }
  for idx, tt := range matchTests {
    matches, err := Glob(tt.pattern)
    if inSlice(tt.s, matches) != tt.match {
      if tt.match {
        t.Errorf("#%v. Glob(%#q) = %#v - doesn't contain %v, but should", idx, tt.pattern, matches, tt.s)
      } else {
        t.Errorf("#%v. Glob(%#q) = %#v - contains %v, but shouldn't", idx, tt.pattern, matches, tt.s)
      }
    }
    if err != tt.err {
      t.Errorf("#%v. Glob(%#q) has error %v, but should be %v", idx, tt.pattern, err, tt.err)
    }
  }
}

func inSlice(s string, a []string) bool {
  for _, i := range a {
    if filepath.FromSlash(i) == filepath.FromSlash(s) { return true }
  }
  return false
}
