package fyne

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
)

type fyneWindow struct {
	fyne.Window
	view   display.View
	events chan display.Event
}

func (f fyneWindow) Events() chan display.Event {
	return f.events
}

func (f fyneWindow) SetView(v display.View) {
	f.view = v
}

func (f fyneWindow) View() display.View {
	return f.view
}

func (f fyneWindow) FyneWindow() fyne.Window {
	return f.Window
}

type Application struct {
	app fyne.App
	// Windows is a map of windows
	Windows []display.Window

	gb *gameboy.GameBoy
}

// NewApplication creates a new application
func NewApplication(a fyne.App, gb *gameboy.GameBoy) *Application {
	return &Application{
		app:     a,
		Windows: make([]display.Window, 0),
		gb:      gb,
	}
}

// NewWindow creates a new window with the given name and provided
// view.
func (a *Application) NewWindow(name string, view display.View) fyne.Window {
	w := a.app.NewWindow(name)
	b := fyneWindow{
		Window: w,
		view:   view,
		events: make(chan display.Event, 144),
	}
	a.Windows = append(a.Windows, b)
	return b
}

// Run runs the application and blocks until the application is closed,
// or an error occurs.
func (a *Application) Run() error {
	// run each window in a goroutine
	for _, win := range a.Windows {
		// setup the view
		if err := win.View().Setup(win.FyneWindow()); err != nil {
			return err
		}
		win.FyneWindow().Show()
		go func(w display.Window) {
			// run the view
			if err := w.View().Run(w.Events()); err != nil {
				panic(err)
			}
		}(win)
	}

	// frame channel
	frames := make(chan []byte, 144)

	// create the game boy window
	mainWindow := a.app.NewWindow("GomeBoy")
	mainWindow.SetMaster()
	mainWindow.Resize(fyne.NewSize(160*4, 144*4))
	mainWindow.SetPadded(false)

	// create the base canvas for the Emulator
	img := image.NewRGBA(image.Rect(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))
	c := canvas.NewRasterFromImage(img)
	c.ScaleMode = canvas.ImageScalePixels
	c.SetMinSize(fyne.NewSize(ppu.ScreenWidth, ppu.ScreenHeight))
	mainWindow.SetContent(c)
	mainWindow.Show()

	// create a dispatcher
	events := make(chan display.Event, 144)
	go func() {
		for {
			e := <-events
			// is this event for the main window? (e.g. title)
			if e.Type == display.EventTypeTitle {
				mainWindow.SetTitle(e.Data.(string))
			} else {
				for _, w := range a.Windows {
					w.Events() <- e
				}
			}
		}
	}()

	// run the Game Boy emulator
	go func() {
		a.gb.Start(frames, events)
	}()

	// run the main window
	go func() {
		for {
			// get the next frame
			frame := <-frames

			// update the canvas
			for i := 0; i < len(frame); i++ {
				img.Pix[i] = frame[i]
			}

			c.Refresh()
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
