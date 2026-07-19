package browser

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/OverStackedLab/wagon/internal/filelist"
)

type iconSet struct {
	wagon  string
	folder string
	cloud  string
	file   string
}

// Emoji render in any modern terminal without a special font. The nerd
// set uses Nerd Font glyphs from the Font Awesome range for terminals
// with a patched font. Select with WAGON_ICONS=emoji|nerd|off.
var (
	emojiIcons = iconSet{
		wagon:  "\U0001F504", // circular sync arrows, echoing the brand icon
		folder: "\U0001F4C1", // folder
		cloud:  "\U0001F310", // globe
		file:   "\U0001F4C4", // page
	}
	nerdIcons = iconSet{
		wagon:  "\uf0ec", // opposing transfer arrows, echoing the brand icon
		folder: "\uf07b", // closed folder
		cloud:  "\uf0c2", // cloud
		file:   "\uf016", // file outline
	}
)

var activeIcons = pickIcons()

func pickIcons() *iconSet {
	switch os.Getenv("WAGON_ICONS") {
	case "off":
		return nil
	case "nerd":
		return &nerdIcons
	default:
		return &emojiIcons
	}
}

func icon(glyph string) string {
	if activeIcons == nil {
		return ""
	}
	return glyph + " "
}

// iconPad matches the width an item icon adds to a row, so column
// headers stay aligned with icon-prefixed names.
func iconPad() string {
	if activeIcons == nil {
		return ""
	}
	return strings.Repeat(" ", lipgloss.Width(activeIcons.folder)+1)
}

func wagonIcon() string {
	if activeIcons == nil {
		return ""
	}
	return icon(activeIcons.wagon)
}

func iconForKind(kind filelist.Kind) string {
	if activeIcons == nil {
		return ""
	}
	if kind == filelist.Remote {
		return icon(activeIcons.cloud)
	}
	return icon(activeIcons.folder)
}

func iconForItem(item filelist.Item) string {
	if activeIcons == nil {
		return ""
	}
	if item.IsDir {
		return icon(activeIcons.folder)
	}
	return icon(activeIcons.file)
}
