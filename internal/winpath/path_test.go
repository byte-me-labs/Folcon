package winpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRelativePath(t *testing.T) {
	folder := t.TempDir()
	inside := filepath.Join(folder, "sub", "app.exe")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := RelativePath(folder, inside)
	if err != nil {
		t.Fatalf("RelativePath() error: %v", err)
	}
	if got != `.\sub\app.exe` {
		t.Fatalf("RelativePath() = %q, want %q", got, `.\sub\app.exe`)
	}

	outside := filepath.Join(filepath.Dir(folder), "other.exe")
	if _, err := RelativePath(folder, outside); err == nil {
		t.Fatal("RelativePath() accepted a file outside the folder")
	}
}

func TestRelativePathRejectsComma(t *testing.T) {
	folder := t.TempDir()
	bad := filepath.Join(folder, "a,b.exe")
	if _, err := RelativePath(folder, bad); err == nil {
		t.Fatal("RelativePath() accepted a comma")
	}
}

func TestUTF16LenAndLongPath(t *testing.T) {
	if got := UTF16Len("中文abc"); got != 5 {
		t.Fatalf("UTF16Len() = %d, want 5", got)
	}
	long := `C:\` + strings.Repeat("a", 250)
	if !LongPath(long) {
		t.Fatal("LongPath() did not flag an over-long path")
	}
	if LongPath(`C:\short`) {
		t.Fatal("LongPath() flagged a short path")
	}
}

func TestIsOneDrive(t *testing.T) {
	t.Setenv("OneDrive", `C:\Users\Example\OneDrive`)
	t.Setenv("OneDriveConsumer", "")
	t.Setenv("OneDriveCommercial", "")

	if _, ok := IsOneDrive(`C:\Users\Example\OneDrive\Apps\Tool`); !ok {
		t.Fatal("IsOneDrive() missed a OneDrive path")
	}
	if _, ok := IsOneDrive(`C:\Users\Example\OneDriveBackup`); ok {
		t.Fatal("IsOneDrive() matched a sibling with the same prefix")
	}
	if _, ok := IsOneDrive(`D:\Apps`); ok {
		t.Fatal("IsOneDrive() matched a local path")
	}
}

func TestHasSourceFiles(t *testing.T) {
	dir := t.TempDir()
	if HasSourceFiles(dir) {
		t.Fatal("HasSourceFiles() found something in an empty folder")
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if HasSourceFiles(dir) {
		t.Fatal("HasSourceFiles() accepted a text file")
	}
	sub := filepath.Join(dir, "bin")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "app.EXE"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasSourceFiles(dir) {
		t.Fatal("HasSourceFiles() missed an .exe in a subdirectory")
	}
}
