//go:build windows

package winpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckFolderAcceptsTempDir(t *testing.T) {
	dir := t.TempDir()
	if err := CheckFolder(dir); err != nil {
		t.Fatalf("CheckFolder(%q) error: %v", dir, err)
	}
}

func TestCheckFolderRejectsRoot(t *testing.T) {
	root := filepath.VolumeName(os.Getenv("SystemRoot")) + `\`
	if err := CheckFolder(root); err == nil {
		t.Fatalf("CheckFolder(%q) accepted a drive root", root)
	}
}

func TestCheckFolderRejectsNetworkAndLongPaths(t *testing.T) {
	if err := CheckFolder(`\\server\share\folder`); err == nil {
		t.Fatal("CheckFolder() accepted a UNC path")
	}
	long := `C:\` + strings.Repeat("a", 250)
	if err := CheckFolder(long); err == nil {
		t.Fatal("CheckFolder() accepted an over-long path")
	}
}

func TestCheckFolderRejectsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckFolder(file); err == nil {
		t.Fatal("CheckFolder() accepted a file")
	}
}

func TestPreflight(t *testing.T) {
	dir := t.TempDir()
	if err := Preflight(dir); err != nil {
		t.Fatalf("Preflight() error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("Preflight() left files behind: %v", entries)
	}
}

func TestCheckSource(t *testing.T) {
	folder := t.TempDir()
	app := filepath.Join(folder, "bin", "app.exe")
	if err := os.MkdirAll(filepath.Dir(app), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := CheckSource(folder, app)
	if err != nil {
		t.Fatalf("CheckSource() error: %v", err)
	}
	if got != `.\bin\app.exe` {
		t.Fatalf("CheckSource() = %q", got)
	}

	txt := filepath.Join(folder, "readme.txt")
	if err := os.WriteFile(txt, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckSource(folder, txt); err == nil {
		t.Fatal("CheckSource() accepted a .txt file")
	}

	outside := filepath.Join(filepath.Dir(folder), "outside.exe")
	if err := os.WriteFile(outside, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckSource(folder, outside); err == nil {
		t.Fatal("CheckSource() accepted a file outside the folder")
	}
}
