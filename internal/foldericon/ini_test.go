package foldericon

import (
	"strings"
	"testing"
)

func TestMergeCreatesSection(t *testing.T) {
	got := Merge("", `.\app.exe,1`)
	want := "[.ShellClassInfo]\r\nIconResource=.\\app.exe,1\r\nConfirmFileOp=0\r\n"
	if got != want {
		t.Fatalf("Merge() = %q, want %q", got, want)
	}
}

func TestMergePreservesOtherSectionsAndKeys(t *testing.T) {
	in := "[.ShellClassInfo]\n; keep me\nInfoTip=Hello\nIconFile=old.ico\nIconIndex=3\n[Other]\nKey=Value\n"
	got := Merge(in, `.\tool.exe,2`)
	for _, want := range []string{
		"InfoTip=Hello",
		"[Other]",
		"Key=Value",
		"IconResource=.\\tool.exe,2",
		"ConfirmFileOp=0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Merge() does not contain %q:\n%s", want, got)
		}
	}
	for _, gone := range []string{"IconFile=", "IconIndex="} {
		if strings.Contains(got, gone) {
			t.Errorf("Merge() kept %q:\n%s", gone, got)
		}
	}
	if strings.Index(got, "IconResource=") > strings.Index(got, "InfoTip=Hello") {
		t.Errorf("IconResource should be the first key of the section:\n%s", got)
	}
}

func TestMergeReplacesExistingIconAndConfirm(t *testing.T) {
	in := "[.ShellClassInfo]\nConfirmFileOp=1\nIconResource=.\\old.exe,0\nInfoTip=Hi\n"
	got := Merge(in, `.\new.exe,4`)
	if strings.Count(got, "IconResource=") != 1 {
		t.Fatalf("expected one IconResource line:\n%s", got)
	}
	if strings.Count(got, "ConfirmFileOp=") != 1 || !strings.Contains(got, "ConfirmFileOp=0") {
		t.Fatalf("ConfirmFileOp=0 should replace the old value:\n%s", got)
	}
	if !strings.Contains(got, `IconResource=.\new.exe,4`) {
		t.Fatalf("new resource missing:\n%s", got)
	}
}

func TestMergeCaseInsensitiveSection(t *testing.T) {
	in := "[.shellclassinfo]\nInfoTip=Hi\n"
	got := Merge(in, `.\a.exe,0`)
	if strings.Count(got, "[.shellclassinfo]") != 1 {
		t.Fatalf("section duplicated or lost:\n%s", got)
	}
	if !strings.Contains(got, "InfoTip=Hi") {
		t.Fatalf("key lost:\n%s", got)
	}
}

func TestDecodeEncodeRoundTrip(t *testing.T) {
	text := "[.ShellClassInfo]\r\nInfoTip=中文提示\r\nIconResource=.\\应用.exe,0\r\n"
	data := Encode(text)
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xFE {
		t.Fatalf("Encode() missing UTF-16LE BOM: % x", data[:min(4, len(data))])
	}
	if got := Decode(data); got != text {
		t.Fatalf("Decode(Encode()) = %q, want %q", got, text)
	}
}

func TestDecodeUTF16WithoutBOM(t *testing.T) {
	data := Encode("[.ShellClassInfo]\r\nIconResource=.\\a.exe,0\r\n")[2:]
	if got := Decode(data); !strings.Contains(got, "IconResource=.\\a.exe,0") {
		t.Fatalf("Decode() = %q", got)
	}
}

func TestDecodeANSI(t *testing.T) {
	if got := Decode([]byte("[.ShellClassInfo]\r\nInfoTip=Hi\r\n")); !strings.Contains(got, "InfoTip=Hi") {
		t.Fatalf("Decode() = %q", got)
	}
}
