//go:build windows

package foldericon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	fileAttributeReadOnly = 0x00000001
	fileAttributeHidden   = 0x00000002
	fileAttributeSystem   = 0x00000004
)

// IconRef points at an icon group inside a file.
type IconRef struct {
	// Path is relative to the folder, e.g. `.\app.exe`.
	Path  string
	Index int
}

// Apply makes folder show ref's icon in Explorer.
//
// desktop.ini is written first, then the folder gets the READ-ONLY
// attribute. The order matters: once the folder is READ-ONLY, Explorer is
// told to use the customization, and writing desktop.ini then fails with
// access denied on some systems.
func Apply(folder string, ref IconRef) error {
	if strings.TrimSpace(ref.Path) == "" {
		return errors.New("图标路径为空")
	}
	if strings.ContainsAny(ref.Path, ",\r\n") {
		return fmt.Errorf("图标路径包含不支持的字符: %q", ref.Path)
	}

	iniPath := filepath.Join(folder, "desktop.ini")
	original, err := os.ReadFile(iniPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("读取 desktop.ini 失败: %w", err)
	}

	merged := Merge(Decode(original), ref.Path+","+strconv.Itoa(ref.Index))
	if err := clearReadOnly(iniPath); err != nil {
		return err
	}
	if err := writeFileAtomic(iniPath, Encode(merged)); err != nil {
		return fmt.Errorf("写入 desktop.ini 失败: %w", err)
	}
	if err := addAttributes(iniPath, fileAttributeHidden|fileAttributeSystem); err != nil {
		return fmt.Errorf("设置 desktop.ini 属性失败: %w", err)
	}
	if err := addAttributes(folder, fileAttributeReadOnly); err != nil {
		return fmt.Errorf("设置文件夹只读属性失败: %w", err)
	}
	return nil
}

func clearReadOnly(path string) error {
	attrs, err := getAttributes(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取 desktop.ini 属性失败: %w", err)
	}
	if attrs&fileAttributeReadOnly == 0 {
		return nil
	}
	if err := setAttributes(path, attrs&^fileAttributeReadOnly); err != nil {
		return fmt.Errorf("清除 desktop.ini 只读属性失败: %w", err)
	}
	return nil
}

func addAttributes(path string, add uint32) error {
	attrs, err := getAttributes(path)
	if err != nil {
		return err
	}
	if attrs&add == add {
		return nil
	}
	return setAttributes(path, attrs|add)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".folcon-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	remove := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}
	if _, err := f.Write(data); err != nil {
		remove()
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func getAttributes(path string) (uint32, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return 0, err
	}
	return attrs, nil
}

func setAttributes(path string, attrs uint32) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return syscall.SetFileAttributes(p, attrs)
}
