package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/thelolagemann/go-gameboy/pkg/display"
)

var (
	_ display.View = &OAM{}
	_ fyne.Layout  = &sprite{}
)

type OAM struct {
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

func (o *OAM) Run(window fyne.Window, events <-chan display.Event) error {
	// create the grid (40 sprites, 10 sprites per row)
	grid := container.NewGridWithRows(4)

	// set the content of the window
	window.SetContent(grid)

	// create the sprites
	for i := 0; i < 40; i++ {
		// create the sprite
		s := container.New(&sprite{})

		// add the sprite to the grid
		grid.Add(s)
	}

	// handle events
	go func() {
		for {
			select {
			case <-events:
				//TODO handle events
			}
		}
	}()

	return nil
}
