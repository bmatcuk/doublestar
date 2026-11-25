package doublestar

import (
	"io/fs"
	"os"
	"path"
	"runtime"
	"testing"
)

func setupExecOnlyTree(t *testing.T) string {
	t.Helper()
	root := path.Join("testdata", "execonly-test")
	if err := os.MkdirAll(path.Join(root, "noread", "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(root, "noread", "subdir", "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path.Join(root, "noread"), 0o111); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(path.Join(root, "noread"), 0o755)
		os.RemoveAll(root)
	})
	return root
}

func TestGlobStaticSegmentsThroughExecuteOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission bits do not restrict root")
	}
	setupExecOnlyTree(t)

	fsys := os.DirFS("testdata")
	matches, err := Glob(fsys, "execonly-test/*/subdir/file")
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(matches) != 1 || matches[0] != "execonly-test/noread/subdir/file" {
		t.Fatalf("expected [execonly-test/noread/subdir/file], got %v", matches)
	}
}

func TestGlobWalkStaticSegmentsThroughExecuteOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission bits do not restrict root")
	}
	setupExecOnlyTree(t)

	fsys := os.DirFS("testdata")
	var matches []string
	err := GlobWalk(fsys, "execonly-test/*/subdir/file", func(p string, d fs.DirEntry) error {
		matches = append(matches, p)
		return nil
	})
	if err != nil {
		t.Fatalf("GlobWalk returned error: %v", err)
	}
	if len(matches) != 1 || matches[0] != "execonly-test/noread/subdir/file" {
		t.Fatalf("expected [execonly-test/noread/subdir/file], got %v", matches)
	}
}
