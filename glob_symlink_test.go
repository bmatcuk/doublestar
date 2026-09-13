package doublestar

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFilepathGlobParentSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("parent-dir symlinks are not created reliably on Windows")
	}

	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(sub, "link-to-parent")
	if err := os.Symlink("..", link); err != nil {
		t.Skipf("symlink: %v", err)
	}

	patterns := []string{
		filepath.Join(sub, "**"),
		filepath.Join(link, "**"),
		link,
	}
	for _, pattern := range patterns {
		pattern := pattern
		t.Run(pattern, func(t *testing.T) {
			done := make(chan struct{})
			var matches []string
			var err error
			go func() {
				matches, err = FilepathGlob(pattern)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatalf("FilepathGlob(%q) hung on parent symlink", pattern)
			}
			if err != nil {
				t.Fatalf("FilepathGlob(%q): %v", pattern, err)
			}
			if pattern == link && len(matches) != 1 {
				t.Fatalf("FilepathGlob(%q) = %v, want the symlink path", pattern, matches)
			}
			if pattern != link && len(matches) == 0 {
				t.Fatalf("FilepathGlob(%q) returned no matches", pattern)
			}
		})
	}
}
