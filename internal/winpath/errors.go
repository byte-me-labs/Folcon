// Package winpath validates folders and the files Folcon points at.
package winpath

import "fmt"

// Reason identifies why a path cannot be used.
type Reason string

const (
	ReasonNotExist      Reason = "not-exist"
	ReasonNotDir        Reason = "not-dir"
	ReasonRoot          Reason = "root"
	ReasonLongPath      Reason = "long-path"
	ReasonOneDrive      Reason = "onedrive"
	ReasonNetwork       Reason = "network"
	ReasonReparse       Reason = "reparse"
	ReasonNoWrite       Reason = "no-write"
	ReasonNoPermission  Reason = "no-permission"
	ReasonSourceOutside Reason = "source-outside"
	ReasonSourceInvalid Reason = "source-invalid"
	ReasonSourceMissing Reason = "source-missing"
)

// Error explains why a folder or file is rejected, in words the interface can
// show to the user.
type Error struct {
	Reason Reason
	Path   string
	Detail string
}

func (e *Error) Error() string {
	switch e.Reason {
	case ReasonNotExist:
		return fmt.Sprintf("路径不存在：%s", e.Path)
	case ReasonNotDir:
		return fmt.Sprintf("所选路径不是文件夹：%s", e.Path)
	case ReasonRoot:
		return fmt.Sprintf("不能选择磁盘根目录：%s", e.Path)
	case ReasonLongPath:
		return fmt.Sprintf("路径过长，超过 Windows 的 260 个字符限制：%s", e.Path)
	case ReasonOneDrive:
		return fmt.Sprintf("不支持 OneDrive 同步文件夹，desktop.ini 不会被同步：%s", e.Path)
	case ReasonNetwork:
		return fmt.Sprintf("不支持网络位置，请选择本地磁盘上的文件夹：%s", e.Path)
	case ReasonReparse:
		return fmt.Sprintf("不支持符号链接、联接点或 OneDrive 占位符：%s", e.Path)
	case ReasonNoWrite:
		return fmt.Sprintf("没有写入权限：%s", e.Path)
	case ReasonNoPermission:
		if e.Detail != "" {
			return fmt.Sprintf("无法修改文件夹或 desktop.ini 的属性：%s（%s）", e.Path, e.Detail)
		}
		return fmt.Sprintf("无法修改文件夹或 desktop.ini 的属性：%s", e.Path)
	case ReasonSourceOutside:
		return fmt.Sprintf("只能选择该文件夹内的文件：%s", e.Path)
	case ReasonSourceInvalid:
		return fmt.Sprintf("只能选择 .exe 或 .ico 文件：%s", e.Path)
	case ReasonSourceMissing:
		return fmt.Sprintf("文件不存在或已被移动：%s", e.Path)
	default:
		return fmt.Sprintf("不支持的位置：%s", e.Path)
	}
}
