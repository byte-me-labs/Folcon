package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/byte-me-labs/folcon/internal/winicon"
)

func testSource() *winicon.Source {
	return &winicon.Source{
		Path: `C:\Temp\demo\app.exe`,
		Groups: []winicon.Group{
			{
				Ordinal: 0,
				ID:      1,
				Frames:  []winicon.Frame{{Width: 32, Height: 32, BPP: 32}},
			},
			{
				Ordinal:  1,
				ID:       7,
				Name:     "MSH_SECURITY",
				IsString: true,
				Frames:   []winicon.Frame{{Width: 256, Height: 256, BPP: 32}},
			},
		},
	}
}

func TestWelcomeView(t *testing.T) {
	a := New()
	tester := ui.NewTester(a.view, welcomeWidth, welcomeHeight)

	if !tester.HasText("Folcon") {
		t.Fatalf("welcome view is missing the title; texts: %q", tester.Texts())
	}
	if !tester.HasText("把文件夹拖到这里") {
		t.Fatalf("welcome view is missing the drop zone; texts: %q", tester.Texts())
	}
	if !tester.HasText("选择文件夹") {
		t.Fatalf("welcome view is missing the folder button; texts: %q", tester.Texts())
	}
}

func TestWelcomeShowsNotice(t *testing.T) {
	a := New()
	a.notice = "已完成 3 个文件夹。"
	tester := ui.NewTester(a.view, welcomeWidth, welcomeHeight)

	if !tester.HasText("已完成 3 个文件夹。") {
		t.Fatalf("notice missing; texts: %q", tester.Texts())
	}
}

func TestWelcomeShowsProgress(t *testing.T) {
	a := New()
	a.running = true
	a.folders = []string{`C:\Temp\demo`, `C:\Temp\other`}
	a.index = 1
	a.applied = 1
	a.status = "正在读取图标…"
	tester := ui.NewTester(a.view, welcomeWidth, welcomeHeight)

	if !tester.HasText("正在读取图标…") {
		t.Fatalf("status missing; texts: %q", tester.Texts())
	}
	if !tester.HasText("第 2 个，共 2 个 · 已应用 1 个") {
		t.Fatalf("progress missing; texts: %q", tester.Texts())
	}
}

func assertFits(t *testing.T, a *App, texts ...string) {
	t.Helper()
	tester := ui.NewTester(a.view, welcomeWidth, welcomeHeight)
	for _, text := range texts {
		r, ok := tester.Find(text)
		if !ok {
			t.Fatalf("%q is not shown at %d×%d; texts: %q",
				text, welcomeWidth, welcomeHeight, tester.Texts())
		}
		if r.X < 0 || r.Y < 0 ||
			r.X+r.W > float32(welcomeWidth) || r.Y+r.H > float32(welcomeHeight) {
			t.Fatalf("%q at %+v does not fit in %d×%d", text, r, welcomeWidth, welcomeHeight)
		}
	}
}

func TestWelcomeFitsCompactWindow(t *testing.T) {
	running := New()
	running.running = true
	running.folders = []string{`C:\Temp\demo`, `C:\Temp\other`}
	running.index = 1
	running.applied = 1
	running.status = "正在读取图标…"
	assertFits(t, running,
		"Folcon",
		"把应用图标设为所在文件夹的图标。",
		"正在读取图标…",
		"第 2 个，共 2 个 · 已应用 1 个",
		"把文件夹拖到这里",
		"选择文件夹…",
	)

	idle := New()
	idle.notice = "已结束向导，已应用的图标不会撤销。"
	assertFits(t, idle,
		"Folcon",
		"把应用图标设为所在文件夹的图标。",
		"把文件夹拖到这里",
		"选择文件夹…",
		"已结束向导，已应用的图标不会撤销。",
	)
}

func TestUsableGroupIndexes(t *testing.T) {
	src := testSource()
	src.Groups = append(src.Groups,
		winicon.Group{Ordinal: 2, Err: os.ErrPermission},
		winicon.Group{Ordinal: 3},
	)
	got := usableGroupIndexes(src)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("usableGroupIndexes = %v, want [0 1]", got)
	}
	if got := usableGroupIndexes(nil); got != nil {
		t.Fatalf("usableGroupIndexes(nil) = %v", got)
	}
	if got := usableGroupIndexes(&winicon.Source{}); len(got) != 0 {
		t.Fatalf("usableGroupIndexes(empty) = %v", got)
	}
}

func TestDefaultGroupPrefersLargest(t *testing.T) {
	src := testSource()
	if got := defaultGroup(src, usableGroupIndexes(src)); got != 1 {
		t.Fatalf("defaultGroup = %d, want the 256 group (1)", got)
	}

	// Equal largest sizes keep the earlier group.
	src.Groups[0].Frames = []winicon.Frame{{Width: 256, Height: 256, BPP: 32}}
	if got := defaultGroup(src, usableGroupIndexes(src)); got != 0 {
		t.Fatalf("defaultGroup = %d, want the earlier group (0) on a tie", got)
	}

	// A larger first group wins over a smaller later one.
	src = testSource()
	src.Groups[1].Frames = []winicon.Frame{{Width: 16, Height: 16, BPP: 32}}
	if got := defaultGroup(src, usableGroupIndexes(src)); got != 0 {
		t.Fatalf("defaultGroup = %d, want the 32px group (0)", got)
	}
}

func TestCleanFolders(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "note.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := cleanFolders([]string{
		"",
		"   ",
		root,
		sub,
		strings.ToUpper(root),
		file,
		filepath.Join(root, "missing"),
	})
	if len(got) != 2 {
		t.Fatalf("cleanFolders returned %d folders (%q), want 2", len(got), got)
	}
	if got[0] != root && !strings.EqualFold(got[0], root) {
		t.Fatalf("first folder = %q, want %q", got[0], root)
	}
	if got[1] != sub && !strings.EqualFold(got[1], sub) {
		t.Fatalf("second folder = %q, want %q", got[1], sub)
	}
}

func TestFinishWizardReportsCount(t *testing.T) {
	a := New()
	a.running = true
	a.applied = 2
	a.finishWizard()
	if a.running {
		t.Fatal("still running after finishWizard")
	}
	if !strings.Contains(a.notice, "2") {
		t.Fatalf("notice = %q, want the applied count", a.notice)
	}
}
