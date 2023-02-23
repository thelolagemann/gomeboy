package views

import (
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
)

type Gameboy struct {
}

func (g *Gameboy) Run(frames <-chan []byte, events <-chan display.Event, img *image.RGBA) error {
	for {
		select {
		case e := <-events:
			switch e.Type {
			case display.EventTypeQuit:
				return nil
			case display.EventTypeFrame:
				// refresh canvas

			}
		case f := <-frames:
			// set the pixels of the image
			for i := 0; i < len(f); i++ {
				img.Pix[i] = f[i]
			}
		}
	}
}
