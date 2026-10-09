package ui

import (
	"image/color"

	"github.com/thomaslaurenson/cellmate/internal/edition"
)

// theme is the window chrome of one edition. The table and cards look the
// same in both; menus and message boxes follow the Windows of the day.
type theme struct {
	face      color.RGBA // menu bar, dialog and button background
	caption   color.RGBA // dialog title bar
	highlight color.RGBA // selected menu item
	disabled  color.RGBA // greyed-out menu text
	dropdown  color.RGBA // open menu background
	// embossed draws disabled text with a white shadow, as Windows 95 did;
	// XP drew it flat.
	embossed bool
}

var (
	theme95 = theme{
		face:      color.RGBA{0xc0, 0xc0, 0xc0, 0xff},
		caption:   color.RGBA{0x00, 0x00, 0x80, 0xff},
		highlight: color.RGBA{0x00, 0x00, 0x80, 0xff},
		disabled:  color.RGBA{0x80, 0x80, 0x80, 0xff},
		dropdown:  color.RGBA{0xc0, 0xc0, 0xc0, 0xff},
		embossed:  true,
	}
	themeXP = theme{
		face:      color.RGBA{0xec, 0xe9, 0xd8, 0xff},
		caption:   color.RGBA{0x00, 0x54, 0xe3, 0xff},
		highlight: color.RGBA{0x31, 0x6a, 0xc5, 0xff},
		disabled:  color.RGBA{0xac, 0xa8, 0x99, 0xff},
		dropdown:  color.RGBA{0xff, 0xff, 0xff, 0xff},
	}
)

func themeFor(e edition.Edition) theme {
	if e == edition.Win95 {
		return theme95
	}
	return themeXP
}
