package winpath

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

const desktopIni = "desktop.ini"

// UTF16Len returns the length of s in UTF-16 code units, which is what
// Windows counts for MAX_PATH.
func UTF16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// LongPath reports whether desktop.ini inside folder would exceed MAX_PATH.
func LongPath(folder string) bool {
	return UTF16Len(filepath.Join(folder, desktopIni)) >= 260
}

// RelativePath returns candidate relative to folder in the form used by
// IconResource, e.g. `.\app.exe`.
func RelativePath(folder, candidate string) (string, error) {
	folderAbs, err := filepath.Abs(folder)
	if err != nil {
		return "", err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(folderAbs, candidateAbs)
	if err != nil {
		return "", &Error{Reason: ReasonSourceOutside, Path: candidate, Detail: err.Error()}
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", &Error{Reason: ReasonSourceOutside, Path: candidate}
	}
	if strings.ContainsAny(rel, ",\r\n") {
		return "", &Error{Reason: ReasonSourceInvalid, Path: candidate, Detail: "路径包含逗号或换行"}
	}
	return ".\\" + rel, nil
}

// IsOneDrive reports whether p is inside a OneDrive sync root.
func IsOneDrive(p string) (string, bool) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", false
	}
	for _, env := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		root := os.Getenv(env)
		if strings.TrimSpace(root) == "" {
			continue
		}
		root = filepath.Clean(root)
		if sameOrInside(abs, root) {
			return root, true
		}
	}
	return "", false
}

func sameOrInside(path, root string) bool {
	if strings.EqualFold(path, root) {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return len(path) > len(prefix) && strings.EqualFold(path[:len(prefix)], prefix)
}

// HasSourceFiles reports whether folder, or a directory below it, contains an
// .exe or .ico file.
func HasSourceFiles(folder string) bool {
	found := false
	_ = filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if found {
			return fs.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		if IsSourceFile(path) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// IsSourceFile reports whether path has an extension Folcon can read icons
// from.
func IsSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".ico":
		return true
	default:
		return false
	}
}
