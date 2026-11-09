// Package winicon reads the icon groups out of Windows executables and .ico files.
package winicon

// Group is one RT_GROUP_ICON resource, or the single group of an .ico file.
type Group struct {
	// Ordinal is the zero-based position in the file's icon resources. It is
	// the value Explorer expects after IconResource's comma.
	Ordinal int
	// MaxWidth is the width of the widest frame. Zero means the group has no
	// usable frame.
	MaxWidth int
	// Err records why this group's frames could not be read. Groups with an
	// error are skipped when choosing an icon.
	Err error
}

// Source is a file the user chose.
type Source struct {
	Path  string
	IsICO bool
	// Groups holds the icon groups of an executable, or one group for an .ico
	// file.
	Groups []Group
}
