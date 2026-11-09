// Package app implements Folcon's wizard.
package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/byte-me-labs/folcon/internal/foldericon"
	"github.com/byte-me-labs/folcon/internal/refresh"
	"github.com/byte-me-labs/folcon/internal/winicon"
	"github.com/byte-me-labs/folcon/internal/winpath"
)

// The wizard keeps a compact, fixed-size welcome window: every choice happens
// in the system file dialog, and icons are applied without a preview page.
const (
	welcomeWidth  = 360
	welcomeHeight = 220
)

// App holds the wizard's state. The main thread owns every field except the
// atomic flags; background goroutines hand results back with Window.Update.
type App struct {
	win *mygo.Window

	notice string

	// running is true from the moment a batch starts until it ends.
	running bool
	folders []string
	index   int
	folder  string
	applied int

	busy   bool
	status string

	// generation changes whenever a batch starts or ends; a goroutine that
	// finishes late checks it before touching the interface.
	generation int

	dialogOpen     atomic.Bool
	closeConfirmed atomic.Bool
}

// New returns an app on the welcome screen.
func New() *App {
	return &App{}
}

// Start creates the native window. Call it once the MyGo app is ready.
func (a *App) Start() {
	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:           "Folcon",
		Width:           welcomeWidth,
		Height:          welcomeHeight,
		UseContentSize:  true,
		DisableResize:   true,
		DisableMinimize: true,
		DisableMaximize: true,
		Content:         ui.View(a.view),
	})
	a.win.OnFileDrop(func(e *mygo.FileDropEvent) {
		paths := append([]string(nil), e.Paths...)
		a.win.Update(func() { a.startFromDrop(paths) })
	})
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if a.closeConfirmed.Load() || !a.running || a.applied == 0 {
			return
		}
		e.PreventDefault()
		a.confirmExit()
	})
}

func (a *App) startFromDrop(paths []string) {
	if a.running {
		return
	}
	if !a.startWizard(paths) {
		a.notice = "请拖入文件夹，而不是文件。"
	}
}

func (a *App) chooseFolders() {
	if a.running {
		return
	}
	gen := a.generation
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    a.win,
			Title:     "选择文件夹",
			Message:   "可以一次选择多个文件夹，之后会逐个为它们选择图标。",
			Directory: true,
			Multiple:  true,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.win.Update(func() {
			if gen != a.generation || a.running {
				return
			}
			if !a.startWizard(paths) {
				a.notice = "没有可处理的文件夹。"
			}
		})
	}()
}

// startWizard begins a batch. It reports whether any folder was accepted.
func (a *App) startWizard(paths []string) bool {
	if a.running {
		return false
	}
	folders := cleanFolders(paths)
	if len(folders) == 0 {
		return false
	}
	a.generation++
	a.running = true
	a.folders = folders
	a.index = 0
	a.applied = 0
	a.notice = ""
	a.busy = false
	a.status = ""
	a.processCurrent()
	return true
}

// cleanFolders keeps existing directories, absolute and without duplicates.
func cleanFolders(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		key := strings.ToLower(filepath.Clean(abs))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, abs)
	}
	return out
}

// processCurrent preflights the current folder and, when it can be used,
// opens the source picker right away.
func (a *App) processCurrent() {
	if !a.running {
		return
	}
	if a.index >= len(a.folders) {
		a.finishWizard()
		return
	}
	a.folder = a.folders[a.index]
	a.busy = true
	a.status = "正在检查文件夹…"

	gen := a.generation
	folder := a.folder
	go func() {
		preflightErr := winpath.Preflight(folder)
		hasSources := false
		if preflightErr == nil {
			hasSources = winpath.HasSourceFiles(folder)
		}
		a.win.Update(func() {
			if gen != a.generation {
				return
			}
			a.busy = false
			switch {
			case preflightErr != nil:
				a.status = ""
				go a.folderProblem(gen, folder, preflightErr.Error())
			case !hasSources:
				a.status = ""
				go a.emptyFolder(gen, folder)
			default:
				a.status = "正在等待选择应用…"
				go a.promptSource(gen, folder)
			}
		})
	}()
}

// promptSource opens the file dialog for folder. Cancelling it skips the
// folder, or ends the batch when nothing has been applied yet.
func (a *App) promptSource(gen int, folder string) {
	if !a.beginDialog() {
		return
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:      a.win,
		Title:       "为「" + filepath.Base(folder) + "」选择应用",
		Message:     "只能选择此文件夹内的 .exe 或 .ico 文件。\n" + folder,
		DefaultPath: folder,
		Filters: []mygo.FileFilter{{
			Name:       "应用程序或图标 (*.exe; *.ico)",
			Extensions: []string{"exe", "ico"},
		}},
	})
	a.endDialog()
	if err != nil || len(paths) == 0 {
		a.win.Update(func() {
			if gen != a.generation {
				return
			}
			a.busy = false
			a.status = ""
			a.stopBatch(gen, func() { a.skipCurrent() })
		})
		return
	}
	a.win.Update(func() {
		if gen != a.generation || a.folder != folder || !a.running {
			return
		}
		a.loadSource(paths[0])
	})
}

// folderProblem reports a folder that Preflight rejected.
func (a *App) folderProblem(gen int, folder, detail string) {
	if !a.beginDialog() {
		return
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        a.win,
		Type:          mygo.MessageWarning,
		Title:         "无法处理此文件夹",
		Message:       "Folcon 无法为这个文件夹设置图标：",
		Detail:        folder + "\n\n" + detail,
		Buttons:       []string{"跳过此文件夹", "结束向导"},
		DefaultButton: 0,
		CancelButton:  1,
	})
	a.endDialog()
	a.win.Update(func() {
		if gen != a.generation {
			return
		}
		if err != nil || res.Button == 0 {
			a.skipCurrent()
			return
		}
		a.stopBatch(gen, func() { a.skipCurrent() })
	})
}

// emptyFolder reports a folder with no .exe or .ico to choose from.
func (a *App) emptyFolder(gen int, folder string) {
	if !a.beginDialog() {
		return
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        a.win,
		Type:          mygo.MessageInfo,
		Title:         "没有可用的文件",
		Message:       "这个文件夹及其子文件夹中没有 .exe 或 .ico 文件。",
		Detail:        folder,
		Buttons:       []string{"跳过此文件夹", "结束向导"},
		DefaultButton: 0,
		CancelButton:  1,
	})
	a.endDialog()
	a.win.Update(func() {
		if gen != a.generation {
			return
		}
		if err != nil || res.Button == 0 {
			a.skipCurrent()
			return
		}
		a.stopBatch(gen, func() { a.skipCurrent() })
	})
}

func (a *App) skipCurrent() {
	if !a.running {
		return
	}
	a.index++
	a.processCurrent()
}

// loadSource validates candidate, reads its icons, and applies the group with
// the highest resolution without asking.
func (a *App) loadSource(candidate string) {
	if _, err := winpath.CheckSource(a.folder, candidate); err != nil {
		a.sourceProblem(a.generation, a.folder, err)
		return
	}

	gen := a.generation
	folder := a.folder
	a.busy = true
	a.status = "正在读取图标…"
	go func() {
		src, loadErr := winicon.LoadSource(candidate)
		a.win.Update(func() {
			if gen != a.generation || a.folder != folder || !a.running {
				return
			}
			a.busy = false
			if loadErr != nil {
				a.status = ""
				a.sourceProblem(gen, folder, loadErr)
				return
			}
			index := defaultGroup(src)
			if index < 0 {
				a.status = ""
				a.sourceProblem(gen, folder, errors.New("该文件不包含可读取的图标。"))
				return
			}
			a.status = ""
			a.applyGroup(folder, src, index)
		})
	}()
}

// sourceProblem explains why the chosen file cannot be used and lets the
// user choose another one.
func (a *App) sourceProblem(gen int, folder string, err error) {
	go func() {
		if !a.beginDialog() {
			return
		}
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  a.win,
			Type:    mygo.MessageWarning,
			Title:   "无法使用该文件",
			Message: err.Error(),
			Detail:  "请选择此文件夹内的 .exe 或 .ico 文件。",
			Buttons: []string{"重新选择"},
		})
		a.endDialog()
		a.win.Update(func() {
			if gen != a.generation || a.folder != folder || !a.running {
				return
			}
			a.busy = false
			a.status = "正在等待选择应用…"
			go a.promptSource(gen, folder)
		})
	}()
}

// defaultGroup picks the usable group that holds the widest icon, so the
// folder gets the highest-resolution artwork. Ties keep the earlier group.
// It returns -1 when the source has no usable group.
func defaultGroup(src *winicon.Source) int {
	best := -1
	if src == nil {
		return best
	}
	for i := range src.Groups {
		g := &src.Groups[i]
		if g.Err != nil || g.MaxWidth <= 0 {
			continue
		}
		if best < 0 || g.MaxWidth > src.Groups[best].MaxWidth {
			best = i
		}
	}
	return best
}

func (a *App) applyGroup(folder string, src *winicon.Source, index int) {
	if a.busy || src == nil || index < 0 || index >= len(src.Groups) {
		return
	}
	group := src.Groups[index]
	if group.Err != nil || group.MaxWidth <= 0 {
		return
	}

	gen := a.generation
	ordinal := group.Ordinal
	if src.IsICO {
		ordinal = 0
	}
	a.busy = true
	a.status = "正在应用…"

	go func() {
		path, err := winpath.CheckSource(folder, src.Path)
		if err == nil {
			err = foldericon.Apply(folder, foldericon.IconRef{Path: path, Index: ordinal})
		}
		if err == nil {
			refresh.Folder(folder)
		}
		a.win.Update(func() {
			if gen != a.generation {
				return
			}
			a.busy = false
			if err != nil {
				a.status = ""
				a.applyFailed(gen, err)
				return
			}
			a.applied++
			a.status = ""
			a.index++
			a.processCurrent()
		})
	}()
}

func (a *App) applyFailed(gen int, err error) {
	go func() {
		if !a.beginDialog() {
			return
		}
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  a.win,
			Type:    mygo.MessageError,
			Title:   "应用图标失败",
			Message: "未能修改文件夹图标，向导将回到开始页。",
			Detail:  err.Error(),
			Buttons: []string{"返回开始"},
		})
		a.endDialog()
		a.win.Update(func() {
			if gen != a.generation {
				return
			}
			a.resetToWelcome()
		})
	}()
}

func (a *App) finishWizard() {
	done := a.applied
	a.resetToWelcome()
	if done > 0 {
		a.notice = fmt.Sprintf("已完成 %d 个文件夹。", done)
	}
}

// stopBatch ends the batch, asking first when icons were already applied.
// resume runs when the user prefers to keep going instead.
func (a *App) stopBatch(gen int, resume func()) {
	if !a.running {
		return
	}
	if a.applied == 0 {
		a.resetToWelcome()
		return
	}
	applied := a.applied
	go func() {
		if !a.beginDialog() {
			return
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:        a.win,
			Type:          mygo.MessageQuestion,
			Title:         "结束向导？",
			Message:       fmt.Sprintf("已经为 %d 个文件夹应用了图标。", applied),
			Detail:        "剩余的文件夹不会处理。已应用的图标不会撤销。",
			Buttons:       []string{"继续处理", "结束向导"},
			DefaultButton: 0,
			CancelButton:  1,
		})
		a.endDialog()
		if err == nil && res.Button == 1 {
			a.win.Update(func() {
				if gen != a.generation {
					return
				}
				a.resetToWelcome()
				a.notice = "已结束向导，已应用的图标不会撤销。"
			})
			return
		}
		if resume != nil {
			a.win.Update(func() {
				if gen != a.generation {
					return
				}
				resume()
			})
		}
	}()
}

func (a *App) confirmExit() {
	applied := a.applied
	go func() {
		if !a.beginDialog() {
			return
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:        a.win,
			Type:          mygo.MessageQuestion,
			Title:         "退出 Folcon？",
			Message:       fmt.Sprintf("已经为 %d 个文件夹应用了图标。", applied),
			Detail:        "剩余的文件夹不会处理。已应用的图标不会撤销。",
			Buttons:       []string{"继续处理", "退出"},
			DefaultButton: 0,
			CancelButton:  1,
		})
		a.endDialog()
		if err != nil || res.Button != 1 {
			return
		}
		a.closeConfirmed.Store(true)
		a.win.Close()
	}()
}

func (a *App) resetToWelcome() {
	a.generation++
	a.running = false
	a.folders = nil
	a.index = 0
	a.folder = ""
	a.applied = 0
	a.busy = false
	a.status = ""
}

func (a *App) beginDialog() bool {
	return a.dialogOpen.CompareAndSwap(false, true)
}

func (a *App) endDialog() {
	a.dialogOpen.Store(false)
}
