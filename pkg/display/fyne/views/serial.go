package views

import (
	"fyne.io/fyne/v2"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
)

type Serial struct {
}

func (s *Serial) Title() string {
	return "Serial"
}

func (s *Serial) Run(window fyne.Window, events <-chan event.Event) error {
	// create a serial view

	return nil
}
