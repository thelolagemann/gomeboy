package display

import "fyne.io/fyne/v2"

type Window interface {
	// Events returns the events channel
	Events() chan Event
	// SetView sets the view of the window
	SetView(v View)
	// View returns the view of the window
	View() View
	// FyneWindow returns the fyne window
	FyneWindow() fyne.Window
}

type baseWindow struct {
	events chan Event
	fyne.Window
	view View
}

func (b *baseWindow) View() View {
	return b.view
}

func (b *baseWindow) SetView(v View) {
	b.view = v
}

func (b *baseWindow) FyneWindow() fyne.Window {
	return b.Window
}
