package views

import (
	"fyne.io/fyne/v2"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
)

type Gameboy struct {
	GB *gameboy.GameBoy
}

func (g *Gameboy) Run(window fyne.Window) error {

	return nil
}
