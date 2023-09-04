package views

import (
	"fyne.io/fyne/v2"
	"github.com/thelolagemann/go-gameboy/pkg/display"
)

var _ display.View = (*Serial)(nil)

type Serial struct {
}

func (s *Serial) Title() string {
	return "Serial"
}

func (s *Serial) Run(window fyne.Window, events <-chan display.Event) error {
	// create a serial view

	return nil
}
