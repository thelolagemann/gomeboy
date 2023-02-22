package display

import (
	"fyne.io/fyne/v2"
	"time"
)

type EventType int

const (
	// EventTypeQuit is the event type for when the user quits the application
	EventTypeQuit EventType = iota
	// EventTypeFrame is the event type for when a frame should be drawn
	EventTypeFrame
	// EventTypeKeyDown is the event type for when a key is pressed
	EventTypeKeyDown
	// EventTypeKeyUp is the event type for when a key is released
	EventTypeKeyUp
)

type Event struct {
	// Type is the type of event
	Type EventType
	// Data is the data associated with the event
	Data interface{}
}

// View defines the interface contract for a view.
type View interface {
	// Run runs the view and blocks until the view is closed,
	// or an error occurs. The event channel is used to send events
	// to the view.
	Run(events <-chan Event) error
	// Setup is called to setup the view using the given window,
	// before the view is run.
	Setup(window fyne.Window) error
}

type Window interface {
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

type Application struct {
	app fyne.App
	// Windows is a map of windows
	Windows []*baseWindow
}

// NewApplication creates a new application
func NewApplication(a fyne.App) *Application {
	return &Application{
		app:     a,
		Windows: make([]*baseWindow, 0),
	}
}

// NewWindow creates a new window with the given name and provided
// view.
func (a *Application) NewWindow(name string, view View) fyne.Window {
	w := a.app.NewWindow(name)
	b := &baseWindow{
		Window: w,
		view:   view,
		events: make(chan Event, 144),
	}
	a.Windows = append(a.Windows, b)
	return b
}

// Run runs the application and blocks until the application is closed,
// or an error occurs.
func (a *Application) Run(frameTime time.Duration) error {
	// run each window in a goroutine
	for _, win := range a.Windows {
		// setup the view
		if err := win.View().Setup(win.FyneWindow()); err != nil {
			return err
		}
		win.FyneWindow().Show()
		go func(w *baseWindow) {
			// run the view
			if err := w.View().Run(w.events); err != nil {
				panic(err)
			}
		}(win)
	}

	// create a dispatcher
	events := make(chan Event, 144)
	go func() {
		for {
			e := <-events
			for _, w := range a.Windows {
				w.events <- e
			}
		}
	}()

	// frame event ticker
	t := time.NewTicker(frameTime)
	go func() {
		for {
			<-t.C
			events <- Event{
				Type: EventTypeFrame,
			}
		}
	}()

	// run the application
	a.app.Run()

	return nil
}

// TODO
// - add a way to close windows
// - implement Resettable interface for remaining components (apu, cpu, interrupts, joypad, mmu, ppu, timer, types)
// - implement a way to save state
// - implement a way to load state
