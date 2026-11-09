// Package foldericon writes the desktop.ini that gives a folder its icon.
package foldericon

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ShellClassInfo is the desktop.ini section that carries folder settings.
const ShellClassInfo = ".ShellClassInfo"

// Decode turns the bytes of a desktop.ini into text.
//
// Explorer writes the file as UTF-16LE with a BOM. Older files are plain
// ANSI, so that case is accepted too.
func Decode(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) >= 2 {
		switch {
		case data[0] == 0xFF && data[1] == 0xFE:
			return decodeUTF16LE(data[2:])
		case data[0] == 0xFE && data[1] == 0xFF:
			return decodeUTF16BE(data[2:])
		}
	}
	if looksUTF16LE(data) {
		return decodeUTF16LE(data)
	}
	if utf8.Valid(data) {
		return string(data)
	}
	// Legacy ANSI file. Treat the bytes as Latin-1 so that every byte
	// survives; non-ASCII names are rare in desktop.ini.
	var b strings.Builder
	b.Grow(len(data))
	for _, c := range data {
		b.WriteRune(rune(c))
	}
	return b.String()
}

// Encode turns text into the UTF-16LE bytes Explorer expects, BOM included.
func Encode(text string) []byte {
	units := utf16.Encode([]rune(text))
	out := make([]byte, 2+len(units)*2)
	out[0], out[1] = 0xFF, 0xFE
	for i, u := range units {
		out[2+i*2] = byte(u)
		out[3+i*2] = byte(u >> 8)
	}
	return out
}

// Merge returns text with the icon resource set in [.ShellClassInfo].
//
// Other sections and other keys in [.ShellClassInfo] are preserved. A stale
// IconFile/IconIndex pair is removed, because it would override IconResource.
// ConfirmFileOp=0 is added so that deleting a folder with a custom icon does
// not make Explorer warn about deleting a system folder.
func Merge(text, iconResource string) string {
	lines := splitLines(text)
	sectionStart, sectionEnd := findSection(lines, ShellClassInfo)

	iconLine := "IconResource=" + iconResource
	confirmLine := "ConfirmFileOp=0"

	if sectionStart < 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "["+ShellClassInfo+"]", iconLine, confirmLine)
		return joinLines(lines)
	}

	kept := make([]string, 0, sectionEnd-sectionStart)
	for _, line := range lines[sectionStart+1 : sectionEnd] {
		key := normalizeKey(line)
		switch key {
		case "iconresource", "iconfile", "iconindex", "confirmfileop":
			// Replaced below. IconFile/IconIndex are not written by this
			// app, but a leftover pair would take precedence.
		default:
			kept = append(kept, line)
		}
	}

	out := make([]string, 0, len(lines)+3)
	out = append(out, lines[:sectionStart+1]...)
	out = append(out, iconLine, confirmLine)
	out = append(out, kept...)
	out = append(out, lines[sectionEnd:]...)
	return joinLines(out)
}

// normalizeKey returns the lowercased key of an INI line, or "" when the
// line is a section header, a comment or blank.
func normalizeKey(line string) string {
	s := strings.TrimSpace(line)
	if s == "" || s[0] == ';' || s[0] == '[' {
		return ""
	}
	if i := strings.IndexByte(s, '='); i >= 0 {
		return strings.ToLower(strings.TrimSpace(s[:i]))
	}
	return strings.ToLower(s)
}

func findSection(lines []string, name string) (start, end int) {
	want := "[" + strings.ToLower(name) + "]"
	start = -1
	for i, line := range lines {
		s := strings.TrimSpace(line)
		if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
			continue
		}
		if start < 0 {
			if strings.ToLower(s) == want {
				start = i
			}
			continue
		}
		return start, i
	}
	if start < 0 {
		return -1, -1
	}
	return start, len(lines)
}

func splitLines(text string) []string {
	text = normalizeNewlines(text)
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\r\n") + "\r\n"
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func decodeUTF16LE(b []byte) string {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	return string(utf16.Decode(units))
}

func decodeUTF16BE(b []byte) string {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = uint16(b[i*2])<<8 | uint16(b[i*2+1])
	}
	return string(utf16.Decode(units))
}

// looksUTF16LE reports whether data looks like UTF-16LE text without a BOM.
// It requires a NUL in most odd positions, which plain UTF-8/ANSI text never
// has.
func looksUTF16LE(data []byte) bool {
	if len(data) < 4 || len(data)%2 != 0 {
		return false
	}
	odd := 0
	for i := 1; i < len(data); i += 2 {
		if data[i] == 0 {
			odd++
		}
	}
	return odd*10 >= (len(data)/2)*6
}
