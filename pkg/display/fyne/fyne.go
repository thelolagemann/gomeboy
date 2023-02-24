package fyne

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
	"image/png"
	"os"
)

var keyMap = map[fyne.KeyName]joypad.Button{
	fyne.KeyA:         joypad.ButtonA,
	fyne.KeyB:         joypad.ButtonB,
	fyne.KeyUp:        joypad.ButtonUp,
	fyne.KeyDown:      joypad.ButtonDown,
	fyne.KeyLeft:      joypad.ButtonLeft,
	fyne.KeyRight:     joypad.ButtonRight,
	fyne.KeyReturn:    joypad.ButtonStart,
	fyne.KeyBackspace: joypad.ButtonSelect,
}

var keyHandlers = map[fyne.KeyName]func(*gameboy.GameBoy){
	fyne.KeyF: func(gb *gameboy.GameBoy) {
		img := gb.PPU.DumpTiledata()

		f, err := os.Create("tiledata.png")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
	},
	fyne.KeyP: func(gb *gameboy.GameBoy) {
		gb.TogglePause()
	},
}

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
// TODO move application to pkg/display
func (a *Application) Run() error {
	// set the default theme
	a.app.Settings().SetTheme(&defaultTheme{})

	// run each window in a goroutine
	for _, win := range a.Windows {
		win.FyneWindow().Show()
		if err := win.View().Run(win); err != nil {
			panic(err)
		}

	}

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
			// lock the gameboy
			e := <-events
			a.gb.Lock()
			// is this event for the main window? (e.g. title)
			if e.Type == display.EventTypeTitle {
				mainWindow.SetTitle(e.Data.(string))
			} else {
				// was this a frame event?
				if e.Type == display.EventTypeFrame {
					c.Refresh()
				}

				// send the event to all windows
				for _, w := range a.Windows {
					w.Events() <- e
				}
			}
			// unlock the gameboy
			a.gb.Unlock()
		}
	}()

	pressed, released := make(chan joypad.Button, 10), make(chan joypad.Button, 10)
	if desk, ok := mainWindow.Canvas().(desktop.Canvas); ok {
		desk.SetOnKeyDown(func(e *fyne.KeyEvent) {
			// check if this is a gameboy key
			if k, ok := keyMap[e.Name]; ok {
				pressed <- k
			} else if h, ok := keyHandlers[e.Name]; ok {
				h(a.gb)
			}
		})
		desk.SetOnKeyUp(func(e *fyne.KeyEvent) {
			if k, ok := keyMap[e.Name]; ok {
				released <- k
			}
		})
	}

	frameBuffer := make(chan []byte, 144)

	// run the Game Boy emulator
	go func() {
		a.gb.Start(frameBuffer, events, pressed, released)
	}()

	// run the frame buffer
	go func() {
		for {
			select {
			case frame := <-frameBuffer:
				// lock the gameboy
				a.gb.Lock()
				// update the image
				for i := 0; i < ppu.ScreenHeight*ppu.ScreenWidth; i++ {
					img.Pix[i*4] = frame[i*3]
					img.Pix[i*4+1] = frame[i*3+1]
					img.Pix[i*4+2] = frame[i*3+2]
					img.Pix[i*4+3] = 255
				}

				// refresh the canvas
				c.Refresh()

				// unlock the gameboy
				a.gb.Unlock()
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
