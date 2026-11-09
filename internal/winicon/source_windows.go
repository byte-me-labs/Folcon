//go:build windows

package winicon

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
)

var enumResourceNamesW = windows.NewLazySystemDLL("kernel32.dll").NewProc("EnumResourceNamesW")

// identifier names one icon resource. Windows resources are identified by a
// 16-bit number or by a UTF-16 string.
type identifier struct {
	ID       uint16
	Name     string
	IsString bool
}

// LoadSource reads every icon group from an .exe or a single group from an
// .ico file.
func LoadSource(path string) (*Source, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("无法读取文件：%w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("所选路径是文件夹")
	}

	switch strings.ToLower(filepath.Ext(abs)) {
	case ".ico":
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("读取图标文件失败：%w", err)
		}
		largest, err := ParseICO(data)
		if err != nil {
			return nil, err
		}
		return &Source{
			Path:  abs,
			IsICO: true,
			Groups: []Group{{
				Ordinal:  0,
				MaxWidth: largest,
			}},
		}, nil
	case ".exe":
		return loadEXE(abs)
	default:
		return nil, fmt.Errorf("只能读取 .exe 或 .ico 文件")
	}
}

func loadEXE(path string) (*Source, error) {
	h, err := windows.LoadLibraryEx(path, 0,
		windows.LOAD_LIBRARY_AS_DATAFILE|windows.LOAD_LIBRARY_AS_IMAGE_RESOURCE)
	if err != nil {
		return nil, fmt.Errorf("无法加载程序资源：%w", err)
	}
	defer windows.FreeLibrary(h)

	ids, err := enumerateGroupIDs(h)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("该程序不包含图标资源")
	}

	groups := make([]Group, 0, len(ids))
	for i, ident := range ids {
		g := Group{Ordinal: i}
		data, err := loadResource(h, ident, windows.ResourceID(rtGroupIcon))
		if err != nil {
			g.Err = fmt.Errorf("读取图标组失败：%w", err)
			groups = append(groups, g)
			continue
		}
		g.MaxWidth, err = parseGroupResource(h, data)
		if err != nil {
			g.Err = err
		}
		groups = append(groups, g)
	}
	return &Source{Path: path, Groups: groups}, nil
}

func enumerateGroupIDs(h windows.Handle) ([]identifier, error) {
	var ids []identifier
	cb := syscall.NewCallback(func(_, _, name *uint16, _ uintptr) uintptr {
		p := uintptr(unsafe.Pointer(name))
		if p>>16 == 0 {
			ids = append(ids, identifier{ID: uint16(p)})
		} else {
			ids = append(ids, identifier{Name: windows.UTF16PtrToString(name), IsString: true})
		}
		return 1
	})
	r, _, callErr := enumResourceNamesW.Call(
		uintptr(h),
		uintptr(windows.ResourceID(rtGroupIcon)),
		cb,
		0,
	)
	if r == 0 {
		if errno, ok := callErr.(syscall.Errno); ok && errno != 0 && errno != windows.ERROR_RESOURCE_TYPE_NOT_FOUND {
			return nil, fmt.Errorf("枚举图标资源失败：%v", errno)
		}
		return nil, nil
	}
	return ids, nil
}

// parseGroupResource reads the declared frame widths from an RT_GROUP_ICON
// resource and verifies that every referenced RT_ICON exists. A width byte of
// 0 means 256.
func parseGroupResource(h windows.Handle, data []byte) (int, error) {
	if len(data) < 6 {
		return 0, fmt.Errorf("图标组数据过短")
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return 0, fmt.Errorf("图标组格式无效")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count <= 0 || count > 1024 {
		return 0, fmt.Errorf("图标组包含无效的图像数量（%d）", count)
	}
	const entrySize = 14
	if len(data) < 6+count*entrySize {
		return 0, fmt.Errorf("图标组数据不完整")
	}

	largest := 0
	var firstErr error
	for i := 0; i < count; i++ {
		off := 6 + i*entrySize
		width := int(data[off])
		if width == 0 {
			width = 256
		}
		size := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		iconID := binary.LittleEndian.Uint16(data[off+12 : off+14])
		if size <= 0 || size > maxIconBytes {
			if firstErr == nil {
				firstErr = fmt.Errorf("图标 %d 的大小无效（%d）", i+1, size)
			}
			continue
		}
		if _, err := loadResource(h, identifier{ID: iconID}, windows.ResourceID(rtIcon)); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("图标 %d：%w", i+1, err)
			}
			continue
		}
		if width > largest {
			largest = width
		}
	}
	if largest == 0 {
		if firstErr != nil {
			return 0, firstErr
		}
		return 0, fmt.Errorf("图标组不包含可读取的图像")
	}
	return largest, nil
}

// loadResource returns data owned by the loaded module; it stays valid until
// FreeLibrary.
func loadResource(h windows.Handle, ident identifier, typeID windows.ResourceID) ([]byte, error) {
	var (
		hr  windows.Handle
		err error
	)
	if ident.IsString {
		hr, err = windows.FindResource(h, ident.Name, typeID)
	} else {
		hr, err = windows.FindResource(h, windows.ResourceID(ident.ID), typeID)
	}
	if err != nil {
		return nil, fmt.Errorf("找不到资源：%w", err)
	}
	data, err := windows.LoadResourceData(h, hr)
	if err != nil {
		return nil, fmt.Errorf("加载资源失败：%w", err)
	}
	return data, nil
}
