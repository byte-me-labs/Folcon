//go:build windows

package winicon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSourceICO(t *testing.T) {
	ico := makeICO(2, []byte{1, 2, 3})
	path := filepath.Join(t.TempDir(), "test.ico")
	if err := os.WriteFile(path, ico, 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := LoadSource(path)
	if err != nil {
		t.Fatalf("LoadSource() error: %v", err)
	}
	if !src.IsICO || len(src.Groups) != 1 || src.Groups[0].MaxWidth != 2 {
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
		if src.Groups[i].Err == nil && src.Groups[i].MaxWidth > 0 {
			usable++
		} else {
			t.Logf("group %d: width=%d err=%v",
				src.Groups[i].Ordinal, src.Groups[i].MaxWidth, src.Groups[i].Err)
		}
	}
	if usable == 0 {
		t.Fatal("every group failed to load")
	}
}
