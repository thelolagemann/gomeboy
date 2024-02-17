package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"image"
)

var (
	_ fyne.Layout = &sprite{}
)

type OAM struct {
	PPU *ppu.PPU
}

func (o *OAM) Title() string {
	return "OAM"
}

type sprite struct {
}

func (s sprite) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(8, 8))
		o.Move(fyne.NewPos(0, 0))
	}
}

func (s sprite) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(8*4, 8*4)
}

func (o *OAM) Run(window fyne.Window, events <-chan event.Event) error {
	// create the grid (40 sprites, 10 sprites per row)
	grid := container.NewGridWithRows(4)

	// set the content of the window
	window.SetContent(grid)

	var spriteImgs []*tappableImage

	// create the sprites
	for i := 0; i < 40; i++ {
		// create image for sprite
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		t := canvas.NewRasterFromImage(img)
		t.ScaleMode = canvas.ImageScalePixels
		t.SetMinSize(fyne.NewSize(float32(8), float32(8)))

		o.PPU.DrawSprite(img, o.PPU.Sprites[i])

		newI := i
		tapImage := newTappableImage(img, t, func(_ *fyne.PointEvent) {
			fmt.Println("you clicked a sprite")
			fmt.Println(o.PPU.Sprites[newI])
		})
		spriteImgs = append(spriteImgs, tapImage)

		grid.Add(tapImage)
	}

	// handle event
	go func() {
		for {
			select {
			case <-events:
				for i, img := range spriteImgs {
					o.PPU.DrawSprite(img.img, o.PPU.Sprites[i])
					img.c.Refresh()
				}
			}
		}
	}()

	return nil
}
