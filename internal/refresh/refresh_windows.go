//go:build windows

// Package refresh asks Explorer to redraw a folder after its icon changed.
package refresh

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	shcneUpdateDir = 0x00001000
	shcnfPathW     = 0x0005

	// ie4uinit is expensive to start and refreshes icons globally, so a batch
	// of folders triggers it at most once every couple of seconds.
	ie4uinitInterval = 2 * time.Second
)

var (
	shell32        = syscall.NewLazyDLL("shell32.dll")
	shChangeNotify = shell32.NewProc("SHChangeNotify")

	mu       sync.Mutex
	lastInit time.Time
)

// Folder tells Explorer that folder changed.
func Folder(folder string) {
	notify(folder)
	scheduleIE4UInit()
}

func notify(folder string) {
	p, err := syscall.UTF16PtrFromString(folder)
	if err != nil {
		return
	}
	_, _, _ = shChangeNotify.Call(
		uintptr(shcneUpdateDir),
		uintptr(shcnfPathW),
		uintptr(unsafe.Pointer(p)),
		0,
	)
}

func scheduleIE4UInit() {
	mu.Lock()
	if time.Since(lastInit) < ie4uinitInterval {
		mu.Unlock()
		return
	}
	lastInit = time.Now()
	mu.Unlock()
	go runIE4UInit()
}

func runIE4UInit() {
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "ie4uinit.exe")
	if _, err := os.Stat(exe); err != nil {
		return
	}
	cmd := exec.Command(exe, "-show")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}
