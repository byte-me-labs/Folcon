//go:build windows

package winicon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSourceICO(t *testing.T) {
	ico := makeICO(2, 2, 32, twoByTwoDIB())
	path := filepath.Join(t.TempDir(), "test.ico")
	if err := os.WriteFile(path, ico, 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := LoadSource(path)
	if err != nil {
		t.Fatalf("LoadSource() error: %v", err)
	}
	if !src.IsICO || len(src.Groups) != 1 || len(src.Groups[0].Frames) != 1 {
		t.Fatalf("source = %+v", src)
	}
}

func TestLoadSourceEXE(t *testing.T) {
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("notepad.exe not available")
	}
	src, err := LoadSource(exe)
	if err != nil {
		t.Fatalf("LoadSource(%q) error: %v", exe, err)
	}
	if len(src.Groups) == 0 {
		t.Fatal("no icon groups")
	}
	usable := 0
	for i := range src.Groups {
		if len(src.Groups[i].Frames) > 0 {
			usable++
		} else {
			t.Logf("group %d (%q, id=%d, string=%v): %v",
				src.Groups[i].Ordinal, src.Groups[i].Name, src.Groups[i].ID,
				src.Groups[i].IsString, src.Groups[i].Err)
		}
	}
	if usable == 0 {
		t.Fatal("every group failed to load")
	}
}
