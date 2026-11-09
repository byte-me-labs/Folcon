//go:build windows

package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/byte-me-labs/folcon/internal/foldericon"
	"github.com/byte-me-labs/folcon/internal/winicon"
	"github.com/byte-me-labs/folcon/internal/winpath"
)

// TestApplyPipeline runs the same steps the wizard runs, against a real
// executable copied into a temporary folder.
func TestApplyPipeline(t *testing.T) {
	// notepad.exe keeps its icons in a language MUI file, which does not
	// follow a copy, so use an executable with embedded icons.
	srcExe := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(srcExe); err != nil {
		t.Skipf("powershell.exe not available: %v", err)
	}

	dir := t.TempDir()
	t.Cleanup(func() { resetAttributes(dir) })

	exe := filepath.Join(dir, "powershell.exe")
	copyFile(t, srcExe, exe)

	if err := winpath.Preflight(dir); err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if !winpath.HasSourceFiles(dir) {
		t.Fatal("HasSourceFiles = false after copying an exe")
	}

	rel, err := winpath.CheckSource(dir, exe)
	if err != nil {
		t.Fatalf("CheckSource: %v", err)
	}
	if rel != `.\powershell.exe` {
		t.Fatalf("relative path = %q, want .\\powershell.exe", rel)
	}

	src, err := winicon.LoadSource(exe)
	if err != nil {
		t.Fatalf("LoadSource: %v", err)
	}
	index := defaultGroup(src)
	if index < 0 {
		t.Fatalf("powershell.exe yielded no usable icon groups: %+v", src.Groups)
	}

	// The wizard applies the largest group without asking.
	group := src.Groups[index]
	ordinal := group.Ordinal
	if err := foldericon.Apply(dir, foldericon.IconRef{Path: rel, Index: ordinal}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "desktop.ini"))
	if err != nil {
		t.Fatalf("read desktop.ini: %v", err)
	}
	text := foldericon.Decode(data)
	want := fmt.Sprintf("IconResource=%s,%d", rel, ordinal)
	if !strings.Contains(text, want) {
		t.Fatalf("desktop.ini = %q, want %q", text, want)
	}
	if !strings.Contains(text, "ConfirmFileOp=0") {
		t.Fatalf("desktop.ini = %q, want ConfirmFileOp=0", text)
	}

	dirAttrs, err := windows.GetFileAttributes(windows.StringToUTF16Ptr(dir))
	if err != nil {
		t.Fatal(err)
	}
	if dirAttrs&windows.FILE_ATTRIBUTE_READONLY == 0 {
		t.Fatalf("folder attrs = %#x, want READ-ONLY", dirAttrs)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

// resetAttributes lets t.TempDir delete a folder that Apply marked read-only
// and a desktop.ini that it hid.
func resetAttributes(dir string) {
	ini := filepath.Join(dir, "desktop.ini")
	clear := func(p string, mask uint32) {
		if attrs, err := windows.GetFileAttributes(windows.StringToUTF16Ptr(p)); err == nil {
			_ = windows.SetFileAttributes(windows.StringToUTF16Ptr(p), attrs&^mask)
		}
	}
	clear(ini, windows.FILE_ATTRIBUTE_READONLY|windows.FILE_ATTRIBUTE_HIDDEN|windows.FILE_ATTRIBUTE_SYSTEM)
	clear(dir, windows.FILE_ATTRIBUTE_READONLY)
}
