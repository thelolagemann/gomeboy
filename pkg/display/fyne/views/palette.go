package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"image/color"
)

type Palette struct {
	widget.BaseWidget
	PPU               *ppu.PPU
	bgRects, objRects [8][4]*canvas.Rectangle
}

func NewPalette(p *ppu.PPU) *Palette {
	pa := &Palette{PPU: p}
	pa.ExtendBaseWidget(pa)
	return pa
}

func (p *Palette) CreateRenderer() fyne.WidgetRenderer {
	// create the main container
	mainContainer := container.NewVBox()

	// create a box for the palettes
	paletteBox := container.NewHBox()
	cgbBGPaletteBox := container.NewVBox()
	cgbOBJPaletteBox := container.NewVBox()

	// add titles to the paletteView boxes
	cgbBGPaletteBox.Add(widget.NewLabelWithStyle("Background", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	cgbOBJPaletteBox.Add(widget.NewLabelWithStyle("Objects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	// create a rectangle for the selected paletteView (larger than the others)
	selectedPalette := canvas.NewRectangle(color.White)
	selectedPalette.SetMinSize(fyne.NewSize(48, 48))
	selectedPalette.CornerRadius = 5

	selectedPaletteColour := color.RGBA{0, 0, 0, 255}

	// create RGB values for the selected paletteView
	selectedPaletteInfoBox := container.NewVBox()
	selectedPaletteInfo := widget.NewTextGrid()

	selectedPaletteInfoBox.Add(selectedPaletteInfo)

	selectedPaletteBox := container.NewHBox(selectedPalette, selectedPaletteInfoBox)

	// TODO determine DMG or CGB (CGB has 16 palettes, DMG has 3)
	for i := 0; i < 8; i++ {
		cgbBGPaletteBox.Add(container.NewHBox())
		cgbOBJPaletteBox.Add(container.NewHBox())
	}

	// create colored rectangles for the palettes
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			// create a rectangle for obj and bg paletteView
			r := canvas.NewRectangle(color.White)
			r.SetMinSize(fyne.NewSize(24, 24))
			bgRect := newWrappedTappable(func() {
				// set the color of the selected paletteView
				selectedPalette.FillColor = toRGB(p.PPU.ColourPalette[i][j])
				selectedPalette.Refresh()
				// set the color of the selected paletteView info
				selectedPaletteColour = toRGB(p.PPU.ColourPalette[i][j])
				selectedPaletteInfo.SetText(fmt.Sprintf("BG\t%d:%d\n#%02x%02x%02x", i, j, selectedPaletteColour.R, selectedPaletteColour.G, selectedPaletteColour.B))
			}, r)
			r2 := canvas.NewRectangle(color.White)
			r2.SetMinSize(fyne.NewSize(24, 24))
			objRect := newWrappedTappable(func() {
				selectedPalette.FillColor = toRGB(p.PPU.ColourSpritePalette[i][j])
				selectedPalette.Refresh()
				// set the color of the selected paletteView info
				selectedPaletteColour = toRGB(p.PPU.ColourSpritePalette[i][j])
				selectedPaletteInfo.SetText(fmt.Sprintf("OBJ\t%d:%d\n#%02x%02x%02x", i, j, selectedPaletteColour.R, selectedPaletteColour.G, selectedPaletteColour.B))
			}, r2)

			// add the rectangle to the paletteView
			cgbBGPaletteBox.Objects[i+1].(*fyne.Container).Add(bgRect)
			cgbOBJPaletteBox.Objects[i+1].(*fyne.Container).Add(objRect)

			p.bgRects[i][j] = r
			p.objRects[i][j] = r2
		}
	}

	// add the paletteView box to the main container
	paletteBox.Add(cgbBGPaletteBox)
	paletteBox.Add(cgbOBJPaletteBox)
	mainContainer.Add(paletteBox)

	// add a spacer between the palettes and the selected paletteView
	mainContainer.Add(widget.NewSeparator())

	// add the selected paletteView box to the main container
	mainContainer.Add(selectedPaletteBox)

	return widget.NewSimpleRenderer(mainContainer)
}

func (p *Palette) Refresh() {
	for i := uint8(0); i < 8; i++ {
		for j := uint8(0); j < 4; j++ {
			// get the color from the paletteView
			bgColor := toRGB(p.PPU.ColourPalette[i][j])
			objColor := toRGB(p.PPU.ColourSpritePalette[i][j])

			// get the rectangle
			bgRect := p.bgRects[i][j]
			objRect := p.objRects[i][j]

			// if the color is not the same as the rectangle, update the rectangle
			if bgColor != bgRect.FillColor {
				bgRect.FillColor = bgColor
				bgRect.Refresh()
			}
			if objColor != objRect.FillColor {
				objRect.FillColor = objColor
				objRect.Refresh()
			}
		}
	}
}

// toRGB converts a 3 element uint8 array to a color.RGBA
// with an alpha value of 255 (opaque)
func toRGB(rgb [3]uint8) color.RGBA {
	return color.RGBA{rgb[0], rgb[1], rgb[2], 255}
}
