package fyne

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"time"
	"unsafe"
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
	fyne.Key1: func(gb *gameboy.GameBoy) {
		gb.PPU.Debug.BackgroundDisabled = !gb.PPU.Debug.BackgroundDisabled
	},
	fyne.Key2: func(gb *gameboy.GameBoy) {
		gb.PPU.Debug.WindowDisabled = !gb.PPU.Debug.WindowDisabled
	},
	fyne.Key3: func(gb *gameboy.GameBoy) {
		gb.PPU.Debug.SpritesDisabled = !gb.PPU.Debug.SpritesDisabled
	},
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
	fyne.KeyY: func(boy *gameboy.GameBoy) {
		// dump current frame
		img := image.NewRGBA(image.Rect(0, 0, 160, 144))
		for y := 0; y < 144; y++ {
			for x := 0; x < 160; x++ {
				img.Set(x, y, color.RGBA{R: boy.PPU.PreparedFrame[y][x][0], G: boy.PPU.PreparedFrame[y][x][1], B: boy.PPU.PreparedFrame[y][x][2], A: 255})
			}
		}

		f, err := os.Create("frame.png")
		if err != nil {
			panic(err)
		}

		if err := png.Encode(f, img); err != nil {
			panic(err)
		}

		f.Close()
	},
	fyne.KeyT: func(gb *gameboy.GameBoy) {
		// print the size of all the various components of the gameboy struct
		fmt.Printf("CPU: %d\n", unsafe.Sizeof(*gb.CPU))
		fmt.Printf("PPU: %d\n", unsafe.Sizeof(*gb.PPU))
		fmt.Printf("MMU: %d\n", unsafe.Sizeof(*gb.MMU))
		fmt.Printf("APU: %d\n", unsafe.Sizeof(*gb.APU))
		fmt.Printf("Timer: %d\n", unsafe.Sizeof(gb.Timer))
		fmt.Printf("Cartridge: %d\n", unsafe.Sizeof(*gb.MMU.Cart))
		fmt.Printf("Joypad: %d\n", unsafe.Sizeof(*gb.Joypad))
		fmt.Printf("GameBoy: %d\n", unsafe.Sizeof(*gb))

		// print the size of the various types used throughout the gameboy
		fmt.Printf("Palette: %d\n", unsafe.Sizeof(palette.Palette{}))
		fmt.Printf("Color: %d\n", unsafe.Sizeof(palette.CGBPalette{}))
		fmt.Printf("Tile: %d\n", unsafe.Sizeof(ppu.Tile{}))
		fmt.Printf("Sprite: %d\n", unsafe.Sizeof(ppu.Sprite{}))
		fmt.Printf("HardwareRegister: %d\n", unsafe.Sizeof(&types.Address{}))

		// print the size of the various components of the PPU
		fmt.Printf("PPU: %d\n", unsafe.Sizeof(*gb.PPU))
		fmt.Printf("Render Job: %d\n", unsafe.Sizeof(ppu.RenderJob{}))
		fmt.Printf("Render Output: %d\n", unsafe.Sizeof(ppu.RenderOutput{}))
	},
	fyne.KeyS: func(gb *gameboy.GameBoy) {
		st := types.NewState()
		gb.Save(st)
		if err := st.SaveToFile("state.json"); err != nil {
			gb.Logger.Errorf("failed to save state: %v", err)
		} else {
			gb.Logger.Infof("saved state to state.json")
		}
	},
}

type fyneWindow struct {
	fyne.Window
	view   display.View
	events chan display.Event
}

type Application struct {
	app fyne.App
	// Windows is a map of windows
	Windows []*fyneWindow

	gb1 *gameboy.GameBoy
	gb2 *gameboy.GameBoy
}

// NewApplication creates a new application
func NewApplication(a fyne.App, gb *gameboy.GameBoy) *Application {
	return &Application{
		app:     a,
		Windows: make([]*fyneWindow, 0),
		gb1:     gb,
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
	a.Windows = append(a.Windows, &b)
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
		win.Show()
		if err := win.view.Run(win, win.events); err != nil {
			panic(err)
		}
	}

	// create the gameboy1 window
	mainWindow1 := a.app.NewWindow("GomeBoy")
	mainWindow1.SetMaster()

	mainWindow1.Resize(fyne.NewSize(160*2, 144*2))
	mainWindow1.SetPadded(false)

	// create the gameboy2 window (for multiplayer) if it exists
	var mainWindow2 fyne.Window
	if a.gb2 != nil {
		mainWindow2 = a.app.NewWindow("GomeBoy 2")
		mainWindow2.Resize(fyne.NewSize(160*4, 144*4))
		mainWindow2.SetPadded(false)

	}

	// create the image to draw to
	img := image.NewRGBA(image.Rect(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))

	// create the canvas
	c := canvas.NewRasterFromImage(img)
	c.ScaleMode = canvas.ImageScalePixels
	c.SetMinSize(fyne.NewSize(ppu.ScreenWidth, ppu.ScreenHeight))

	// create submenus
	menuItemOpenROM := fyne.NewMenuItem("Open ROM", func() {
		// open a file dialog to select a ROM
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			// read the ROM
			rom, err := io.ReadAll(reader)
			if err != nil {
				return
			}

			// close the current gameboy if it's running
			if a.gb1.IsRunning() {
				a.gb1.Lock()
				// close the current gameboy
				a.gb1.Close <- struct{}{}
				a.gb1.Unlock()
				a.gb1 = nil
			}
			// load the ROM
			a.gb1 = gameboy.NewGameBoy(rom) // TODO retain the options from the previous gameboy

			// start the gameboy
			a.StartGameView(img, c, a.gb1, mainWindow1)
		}, mainWindow1)
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".gb", ".gbc"}))
		lister, err := storage.ListerForURI(storage.NewFileURI("."))
		if err != nil {
			// TODO
		}
		fileDialog.SetLocation(lister)
		fileDialog.Show()
	})
	menuItemSaveState := fyne.NewMenuItem("Save State", func() {
		// TODO
	})
	menuItemLoadState := fyne.NewMenuItem("Load State", func() {
		// TODO
	})

	// add menu items to submenus
	fileMenu := fyne.NewMenu("File", menuItemOpenROM, menuItemSaveState, menuItemLoadState)

	// create main menu
	mainMenu := fyne.NewMainMenu(
		fileMenu,
	)
	mainMenu.Refresh()

	// set the content of the window and show it
	mainWindow1.SetContent(c)
	// mainWindow1.SetMainMenu(mainMenu)
	mainWindow1.Show()

	// start the gameboy if ROM is loaded
	if a.gb1.MMU.Cart.MD5 != "" {
		a.StartGameView(img, c, a.gb1, mainWindow1)
	}

	// run the application
	a.app.Run()

	// close the gameboy on exit
	if a.gb1.IsRunning() {
		a.gb1.Lock()
		// close the current gameboy
		a.gb1.Close <- struct{}{}
		a.gb1.Close <- struct{}{}
		a.gb1.Unlock()
	}
	// wait for the gameboy to close
	for a.gb1.IsRunning() {
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

func (a *Application) StartGameView(img *image.RGBA, c *canvas.Raster, gb *gameboy.GameBoy, window fyne.Window) {
	// setup framebuffer
	fb := make(chan []byte, 144)
	go func() {
		for {
			select {
			case f := <-fb:
				// copy the framebuffer to the image
				for i := 0; i < ppu.ScreenHeight*ppu.ScreenWidth; i++ {
					img.Pix[i*4] = f[i*3]
					img.Pix[i*4+1] = f[i*3+1]
					img.Pix[i*4+2] = f[i*3+2]
					img.Pix[i*4+3] = 255
				}

				// refresh the canvas
				c.Refresh()
			}
		}
	}()

	// create a dispatcher
	events := make(chan display.Event, 144)
	go func() {
		for {
			// lock the gameboy
			e := <-events
			gb.Lock()
			// is this event for the main window? (e.g. title)
			if e.Type == display.EventTypeTitle {
				window.SetTitle(e.Data.(string))
			} else {
				// send the event to all windows
				for _, w := range a.Windows {
					w.events <- e
				}
			}
			// unlock the gameboy
			gb.Unlock()
		}
	}()

	// handle input
	pressed, release := make(chan joypad.Button, 10), make(chan joypad.Button, 10)
	if desk, ok := window.Canvas().(desktop.Canvas); ok {
		desk.SetOnKeyDown(func(e *fyne.KeyEvent) {
			// check if this is a gameboy key
			if k, ok := keyMap[e.Name]; ok {
				pressed <- k
			} else if h, ok := keyHandlers[e.Name]; ok {
				h(a.gb1)
			} else if e.Name == fyne.KeyEscape {

			}
		})
		desk.SetOnKeyUp(func(e *fyne.KeyEvent) {
			if k, ok := keyMap[e.Name]; ok {
				release <- k
			}
		})
	}

	// TODO reimplement multiplayer
	go gb.Start(fb, events, pressed, release)
}

func (a *Application) AddGameBoy(gb *gameboy.GameBoy) {
	a.gb2 = gb
}

// TODO
// - add a way to close windows
// - implement Resettable interface for remaining components (apu, cpu, interrupts, joypad, mmu, ppu, timer, types)
