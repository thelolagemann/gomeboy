package views

import (
	"fyne.io/fyne/v2"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
)

type Gameboy struct {
	GB *gameboy.GameBoy
}

func (g *Gameboy) Run(window fyne.Window) error {

	return nil
}
