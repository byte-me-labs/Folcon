//go:build windows

package winpath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	fileAttributeReadOnly = 0x00000001
	fileAttributeReparse  = 0x00000400

	driveRemote = 4
)

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	getDriveTypeW = kernel32.NewProc("GetDriveTypeW")
)

// CheckFolder reports whether folder can carry a custom folder icon.
func CheckFolder(folder string) error {
	abs, err := filepath.Abs(folder)
	if err != nil || strings.TrimSpace(folder) == "" {
		return &Error{Reason: ReasonNotExist, Path: folder}
	}
	abs = filepath.Clean(abs)

	if strings.HasPrefix(abs, `\\`) {
		// \\server\share and the \\?\ forms that disable path checks.
		return &Error{Reason: ReasonNetwork, Path: abs}
	}
	vol := filepath.VolumeName(abs)
	if strings.HasPrefix(vol, `\\`) {
		return &Error{Reason: ReasonNetwork, Path: abs}
	}
	if vol == "" {
		return &Error{Reason: ReasonNotExist, Path: abs}
	}
	if root := vol + string(filepath.Separator); strings.EqualFold(abs, root) {
		return &Error{Reason: ReasonRoot, Path: abs}
	}
	if LongPath(abs) {
		return &Error{Reason: ReasonLongPath, Path: abs}
	}
	if root, ok := IsOneDrive(abs); ok {
		return &Error{Reason: ReasonOneDrive, Path: abs, Detail: root}
	}

	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Error{Reason: ReasonNotExist, Path: abs}
		}
		return &Error{Reason: ReasonNotExist, Path: abs, Detail: err.Error()}
	}
	if !info.IsDir() {
		return &Error{Reason: ReasonNotDir, Path: abs}
	}

	attrs, err := getAttributes(abs)
	if err != nil {
		return &Error{Reason: ReasonNotExist, Path: abs, Detail: err.Error()}
	}
	if attrs&fileAttributeReparse != 0 {
		return &Error{Reason: ReasonReparse, Path: abs}
	}

	root := vol + string(filepath.Separator)
	switch driveType(root) {
	case driveRemote:
		return &Error{Reason: ReasonNetwork, Path: abs}
	case 0, 1: // DRIVE_UNKNOWN, DRIVE_NO_ROOT_DIR
		return &Error{Reason: ReasonNotExist, Path: abs}
	}
	return nil
}

// Preflight performs the writes that Apply needs, without leaving changes
// behind, so the user learns about a read-only folder before choosing an
// icon.
func Preflight(folder string) error {
	if err := CheckFolder(folder); err != nil {
		return err
	}
	abs, err := filepath.Abs(folder)
	if err != nil {
		return &Error{Reason: ReasonNotExist, Path: folder}
	}

	f, err := os.CreateTemp(abs, ".folcon-preflight-*")
	if err != nil {
		return &Error{Reason: ReasonNoWrite, Path: abs, Detail: err.Error()}
	}
	tmp := f.Name()
	_, writeErr := f.Write([]byte("ok"))
	closeErr := f.Close()
	removeErr := os.Remove(tmp)
	if writeErr != nil || closeErr != nil || removeErr != nil {
		return &Error{Reason: ReasonNoWrite, Path: abs, Detail: firstError(writeErr, closeErr, removeErr).Error()}
	}

	attrs, err := getAttributes(abs)
	if err != nil {
		return &Error{Reason: ReasonNoPermission, Path: abs, Detail: err.Error()}
	}
	// Setting the current attributes is enough to exercise the permission.
	if err := setAttributes(abs, attrs); err != nil {
		return &Error{Reason: ReasonNoPermission, Path: abs, Detail: err.Error()}
	}

	ini := filepath.Join(abs, "desktop.ini")
	info, err := os.Stat(ini)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return &Error{Reason: ReasonNoPermission, Path: ini, Detail: err.Error()}
	}
	if info.IsDir() {
		return &Error{Reason: ReasonNoPermission, Path: ini, Detail: "desktop.ini 是一个文件夹"}
	}
	iniAttrs, err := getAttributes(ini)
	if err != nil {
		return &Error{Reason: ReasonNoPermission, Path: ini, Detail: err.Error()}
	}
	if err := setAttributes(ini, iniAttrs); err != nil {
		return &Error{Reason: ReasonNoPermission, Path: ini, Detail: err.Error()}
	}
	if iniAttrs&fileAttributeReadOnly == 0 {
		f, err := os.OpenFile(ini, os.O_WRONLY, 0)
		if err != nil {
			return &Error{Reason: ReasonNoPermission, Path: ini, Detail: err.Error()}
		}
		_ = f.Close()
	}
	return nil
}

// CheckSource validates a file chosen from inside folder and returns the
// relative path to write into desktop.ini.
func CheckSource(folder, candidate string) (string, error) {
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", &Error{Reason: ReasonSourceMissing, Path: candidate}
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &Error{Reason: ReasonSourceMissing, Path: abs}
		}
		return "", &Error{Reason: ReasonSourceMissing, Path: abs, Detail: err.Error()}
	}
	if info.IsDir() {
		return "", &Error{Reason: ReasonSourceInvalid, Path: abs, Detail: "这是一个文件夹"}
	}
	if !IsSourceFile(abs) {
		return "", &Error{Reason: ReasonSourceInvalid, Path: abs}
	}

	// Resolve links so a link that points outside the folder cannot smuggle
	// in an icon path that depends on the link.
	folderReal, err := filepath.EvalSymlinks(folder)
	if err != nil {
		return "", &Error{Reason: ReasonSourceMissing, Path: folder, Detail: err.Error()}
	}
	fileReal, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", &Error{Reason: ReasonSourceMissing, Path: abs, Detail: err.Error()}
	}
	return RelativePath(folderReal, fileReal)
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("未知错误")
}

func driveType(root string) uint32 {
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0
	}
	t, _, _ := getDriveTypeW.Call(uintptr(unsafe.Pointer(p)))
	return uint32(t)
}

func getAttributes(path string) (uint32, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return syscall.GetFileAttributes(p)
}

func setAttributes(path string, attrs uint32) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return syscall.SetFileAttributes(p, attrs)
}
