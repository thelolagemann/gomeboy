package themes

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

type Default struct{}

var (
	primaryA0   = color.NRGBA{0xff, 0x78, 0x00, 0xff}
	primaryA20  = color.NRGBA{0xff, 0x88, 0x2e, 0xff}
	primaryA40  = color.NRGBA{0xff, 0x97, 0x4b, 0xff}
	primaryA60  = color.NRGBA{0xff, 0xa6, 0x65, 0xff}
	primaryA80  = color.NRGBA{0xff, 0xb5, 0x7e, 0xff}
	primaryA100 = color.NRGBA{0xff, 0xc4, 0x97, 0xff}

	surfaceA0   = color.NRGBA{0x24, 0x1f, 0x31, 0xff}
	surfaceA20  = color.NRGBA{0x39, 0x34, 0x45, 0xff}
	surfaceA40  = color.NRGBA{0x4f, 0x4a, 0x5a, 0xff}
	surfaceA60  = color.NRGBA{0x66, 0x61, 0x66, 0xff}
	surfaceA80  = color.NRGBA{0x7d, 0x79, 0x86, 0xff}
	surfaceA100 = color.NRGBA{0x96, 0x93, 0x9d, 0xff}

	disabledText = color.NRGBA{156, 156, 156, 255}
	disabled     = color.NRGBA{35, 35, 35, 255}

	boolColor       = color.NRGBA{0, 255, 255, 255}
	boolColorNumber = color.NRGBA{255, 96, 255, 255}
)

const (
	ColorNameSecondary              fyne.ThemeColorName = "secondary"
	ColorNameDisabledText           fyne.ThemeColorName = "disabled-text"
	ColorNameBool                   fyne.ThemeColorName = "bool"
	ColorNameBoolNumber             fyne.ThemeColorName = "bool-number"
	ColorNameBackgroundOnBackground fyne.ThemeColorName = "background-on-background"
)

var colorMap = map[fyne.ThemeColorName]color.Color{
	ColorNameBackgroundOnBackground: surfaceA20,
	ColorNameSecondary:              primaryA60,
	ColorNameDisabledText:           disabledText,
	ColorNameBool:                   boolColor,
	ColorNameBoolNumber:             boolColorNumber,
	theme.ColorNamePrimary:          primaryA20,
	theme.ColorNameBackground:       surfaceA0,
	theme.ColorNameMenuBackground:   surfaceA40,
	theme.ColorNameDisabled:         disabled,
	theme.ColorNameButton:           surfaceA40,
	theme.ColorNameInputBackground:  surfaceA40,
	theme.ColorNameFocus:            surfaceA20,
	theme.ColorNameHover:            surfaceA60,
}

func (d Default) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if c, ok := colorMap[name]; ok {
		return c
	}
	return theme.DefaultTheme().Color(name, fyne.CurrentApp().Settings().ThemeVariant())
}

func (d Default) Font(style fyne.TextStyle) fyne.Resource    { return theme.DefaultTheme().Font(style) }
func (d Default) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(name) }
func (d Default) Size(name fyne.ThemeSizeName) float32       { return theme.DefaultTheme().Size(name) }
