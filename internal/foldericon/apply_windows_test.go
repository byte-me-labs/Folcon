//go:build windows

package foldericon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyWritesIniAndAttributes(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() {
		if attrs, err := getAttributes(dir); err == nil {
			_ = setAttributes(dir, attrs&^fileAttributeReadOnly)
		}
		ini := filepath.Join(dir, "desktop.ini")
		if attrs, err := getAttributes(ini); err == nil {
			_ = setAttributes(ini, attrs&^(fileAttributeReadOnly|fileAttributeHidden|fileAttributeSystem))
		}
	})

	if err := Apply(dir, IconRef{Path: `.\app.exe`, Index: 1}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	ini := filepath.Join(dir, "desktop.ini")
	data, err := os.ReadFile(ini)
	if err != nil {
		t.Fatalf("read desktop.ini: %v", err)
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xFE {
		t.Fatalf("desktop.ini is not UTF-16LE with a BOM: % x", data[:min(4, len(data))])
	}
	text := Decode(data)
	if !strings.Contains(text, `IconResource=.\app.exe,1`) {
		t.Fatalf("desktop.ini content: %q", text)
	}
	if !strings.Contains(text, "ConfirmFileOp=0") {
		t.Fatalf("desktop.ini content: %q", text)
	}

	iniAttrs, err := getAttributes(ini)
	if err != nil {
		t.Fatal(err)
	}
	if iniAttrs&fileAttributeHidden == 0 || iniAttrs&fileAttributeSystem == 0 {
		t.Fatalf("desktop.ini attrs = %#x, want hidden+system", iniAttrs)
	}

	dirAttrs, err := getAttributes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirAttrs&fileAttributeReadOnly == 0 {
		t.Fatalf("folder attrs = %#x, want READ-ONLY", dirAttrs)
	}
}

func TestApplyMergesAndReapplies(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() {
		if attrs, err := getAttributes(dir); err == nil {
			_ = setAttributes(dir, attrs&^fileAttributeReadOnly)
		}
		ini := filepath.Join(dir, "desktop.ini")
		if attrs, err := getAttributes(ini); err == nil {
			_ = setAttributes(ini, attrs&^(fileAttributeReadOnly|fileAttributeHidden|fileAttributeSystem))
		}
	})

	ini := filepath.Join(dir, "desktop.ini")
	if err := os.WriteFile(ini, Encode("[.ShellClassInfo]\r\nInfoTip=保留\r\nIconResource=.\\old.exe,0\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(dir, IconRef{Path: `.\new.exe`, Index: 2}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if err := Apply(dir, IconRef{Path: `.\newer.exe`, Index: 0}); err != nil {
		t.Fatalf("second Apply() error: %v", err)
	}

	data, err := os.ReadFile(ini)
	if err != nil {
		t.Fatal(err)
	}
	text := Decode(data)
	if !strings.Contains(text, "InfoTip=保留") {
		t.Fatalf("InfoTip lost: %q", text)
	}
	if strings.Contains(text, "old.exe") || strings.Contains(text, "new.exe,") {
		t.Fatalf("stale IconResource kept: %q", text)
	}
	if !strings.Contains(text, `IconResource=.\newer.exe,0`) {
		t.Fatalf("IconResource not updated: %q", text)
	}
}
