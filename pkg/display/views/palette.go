package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image/color"
	"reflect"
)

type Palette struct {
	PPU *ppu.PPU
}

func (p Palette) Title() string {
	return "Palette"
}

func (p Palette) Run(window fyne.Window, events <-chan display.Event) error {
	// set non-resizable
	window.SetFixedSize(true)

	// create the main container
	mainContainer := container.NewVBox()

	// create a box for the palettes
	paletteBox := container.NewHBox()
	cgbBGPaletteBox := container.NewVBox()
	cgbOBJPaletteBox := container.NewVBox()

	// add titles to the paletteView boxes
	cgbBGPaletteBox.Add(widget.NewLabelWithStyle("Background", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	cgbOBJPaletteBox.Add(widget.NewLabelWithStyle("Objects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	// create a box for the selected paletteView
	selectedPaletteBox := container.New(selectedPaletteLayout{})

	// create a rectangle for the selected paletteView (larger than the others)
	selectedPalette := canvas.NewRectangle(color.White)
	selectedPaletteBox.Add(selectedPalette)

	selectedPaletteColour := color.RGBA{0, 0, 0, 255}

	// create RGB values for the selected paletteView
	selectedPaletteInfoBox := container.New(selectedPaletteInfoLayout{})
	selectedPaletteRedLabel := widget.NewLabelWithStyle("0", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	selectedPaletteGreenLabel := widget.NewLabelWithStyle("0", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	selectedPaletteBlueLabel := widget.NewLabelWithStyle("0", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})

	// add the RGB values to the info box
	selectedPaletteInfoBox.Add(container.NewHBox(widget.NewLabelWithStyle("R:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), selectedPaletteRedLabel))
	selectedPaletteInfoBox.Add(container.NewHBox(widget.NewLabelWithStyle("G:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), selectedPaletteGreenLabel))
	selectedPaletteInfoBox.Add(container.NewHBox(widget.NewLabelWithStyle("B:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), selectedPaletteBlueLabel))

	// create action box
	selectedPaletteActionBox := container.NewVBox()
	selectedPaletteActionBox.Add(widget.NewButton("Copy", func() {

	}))
	selectedPaletteActionBox.Add(widget.NewButton("Save", func() {

	}))

	// add the selected paletteView to the paletteView box
	selectedPaletteBox.Add(selectedPaletteInfoBox)
	selectedPaletteBox.Add(selectedPaletteActionBox)

	// TODO determine DMG or CGB (CGB has 16 palettes, DMG has 3)
	for i := 0; i < 8; i++ {
		// create a container for the paletteView
		bgPaletteContainer := container.New(&paletteView{})

		// add the paletteView to the paletteView box
		cgbBGPaletteBox.Add(bgPaletteContainer)

		// create a container for the paletteView
		objPaletteContainer := container.New(&paletteView{})

		// add the paletteView to the paletteView box
		cgbOBJPaletteBox.Add(objPaletteContainer)
	}

	// create colored rectangles for the palettes
	for i := 0; i < 8; i++ {
		// copy i to a new variable to prevent it from being overwritten
		newI := i
		for j := 0; j < 4; j++ {
			// copy j to a new variable to prevent it from being overwritten
			newJ := j
			// create a rectangle for obj and bg paletteView
			bgRect := newTappableRectangle(func() {
				// set the color of the selected paletteView
				selectedPalette.FillColor = toRGB(p.PPU.ColourPalette.GetColour(uint8(newI), uint8(newJ)))
				selectedPalette.Refresh()
				// set the color of the selected paletteView info
				selectedPaletteColour = toRGB(p.PPU.ColourPalette.GetColour(uint8(newI), uint8(newJ)))
				selectedPaletteRedLabel.SetText(fmt.Sprintf("0x%02X", selectedPaletteColour.R))
				selectedPaletteGreenLabel.SetText(fmt.Sprintf("0x%02X", selectedPaletteColour.G))
				selectedPaletteBlueLabel.SetText(fmt.Sprintf("0x%02X", selectedPaletteColour.B))
			})
			bgRect.rec.SetMinSize(fyne.NewSize(24, 24))
			objRect := newTappableRectangle(func() {

			})
			objRect.rec.SetMinSize(fyne.NewSize(24, 24))

			// add the rectangle to the paletteView
			cgbBGPaletteBox.Objects[i+1].(*fyne.Container).Add(bgRect)
			cgbOBJPaletteBox.Objects[i+1].(*fyne.Container).Add(objRect)

			// create tap handlers

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

	// set the main container as the content of the window
	window.SetContent(mainContainer)

	// empty event loop
	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case display.EventTypeQuit:
					return
				case display.EventTypeFrame:
					for i := uint8(0); i < 8; i++ {
						for j := uint8(0); j < 4; j++ {
							// get the color from the paletteView
							bgColor := toRGB(p.PPU.ColourPalette.GetColour(i, j))
							objColor := toRGB(p.PPU.ColourSpritePalette.GetColour(i, j))

							// get the rectangle
							bgRect := cgbBGPaletteBox.Objects[i+1].(*fyne.Container).Objects[j].(*tappableRectangle).rec
							objRect := cgbOBJPaletteBox.Objects[i+1].(*fyne.Container).Objects[j].(*tappableRectangle).rec

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
			}
		}
	}()

	return nil
}

type tappableRectangle struct {
	rec *canvas.Rectangle
	widget.BaseWidget
	tapHandler func()
}

func (t *tappableRectangle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.rec)
}

func (t *tappableRectangle) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *tappableRectangle) Tapped(*fyne.PointEvent) {
	if t.tapHandler != nil {
		t.tapHandler()
	}
}

func (t *tappableRectangle) TappedSecondary(*fyne.PointEvent) {
	// do nothing
}

func newTappableRectangle(onTap func()) *tappableRectangle {
	t := &tappableRectangle{rec: canvas.NewRectangle(color.White), tapHandler: onTap}
	t.ExtendBaseWidget(t)
	return t
}

type selectedPaletteLayout struct { // TODO account for right padding
}

func (s selectedPaletteLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 3 {
		return
	}
	for i, obj := range objects {
		fmt.Println("selectedPaletteLayout: type of object", i, reflect.TypeOf(obj))
	}
	// set the size of the selected paletteView
	objects[0].(*canvas.Rectangle).Resize(fyne.NewSize(78, 78))
	// set the position of the selected paletteView
	objects[0].(*canvas.Rectangle).Move(fyne.NewPos(8, 0))
	// set the size of the selected paletteView info
	objects[1].Resize(fyne.NewSize(48, 78))
	// set the position of the selected paletteView info
	objects[1].Move(fyne.NewPos(88, 4))
	// set the size of the selected paletteView action buttons
	objects[2].Resize(fyne.NewSize(48, 48))
	// set the position of the selected paletteView action buttons
	objects[2].Move(fyne.NewPos(222, 0))
}

func (s selectedPaletteLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(96, 88)
}

type selectedPaletteInfoLayout struct {
}

func (s selectedPaletteInfoLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 3 {
		fmt.Println("selectedPaletteInfoLayout: invalid number of objects", len(objects))
		return
	} else {
		fmt.Println("selectedPaletteInfoLayout: valid number of objects", len(objects))
	}

	objects[0].Resize(fyne.NewSize(48, 16))
	objects[0].Move(fyne.NewPos(0, -8))
	objects[1].Resize(fyne.NewSize(48, 16))
	objects[1].Move(fyne.NewPos(0, 16))
	objects[2].Resize(fyne.NewSize(48, 16))
	objects[2].Move(fyne.NewPos(0, 40))
}

func (s selectedPaletteInfoLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(96, 80)
}
