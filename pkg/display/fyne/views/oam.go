package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
)

var (
	_ fyne.Layout = &sprite{}
)

type OAM struct {
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

	// create the sprites
	for i := 0; i < 40; i++ {
		// create the sprite
		s := container.New(&sprite{})

		// add the sprite to the grid
		grid.Add(s)
	}

	// handle event
	go func() {
		for {
			select {
			case <-events:
				//TODO handle event
			}
		}
	}()

	return nil
}
