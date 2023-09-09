//go:build !test

package fyne

import "C"
import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	dialog2 "github.com/sqweek/dialog"
	"github.com/thelolagemann/gomeboy/internal/cartridge"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/views"
	"image"
	"image/color"
	"image/png"
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

	mainWindow1 fyne.Window

	gb1       *gameboy.GameBoy
	gb1Raster *canvas.Raster
	gb1Image  *image.RGBA
	gb2       *gameboy.GameBoy
	mainMenu  *fyne.MainMenu
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
	w.SetOnClosed(func() {
		// close the events channel
		close(b.events)
		// remove the window from the list of windows
		for i, win := range a.Windows {
			if win == &b {
				a.Windows = append(a.Windows[:i], a.Windows[i+1:]...)
			}
		}
	})
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
	a.mainWindow1 = mainWindow1
	mainWindow1.SetMaster()

	mainWindow1.Resize(fyne.NewSize(160*4, 144*4))
	mainWindow1.SetPadded(false)

	// create the gameboy2 window (for multiplayer) if it exists
	var mainWindow2 fyne.Window
	if a.gb2 != nil {
		mainWindow2 = a.app.NewWindow("GomeBoy 2")
		mainWindow2.Resize(fyne.NewSize(160*4, 144*4))
		mainWindow2.SetPadded(false)
	}

	// create the image to draw to
	a.gb1Image = image.NewRGBA(image.Rect(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))

	// create the canvas
	a.gb1Raster = canvas.NewRasterFromImage(a.gb1Image)
	a.gb1Raster.ScaleMode = canvas.ImageScalePixels
	a.gb1Raster.SetMinSize(fyne.NewSize(ppu.ScreenWidth, ppu.ScreenHeight))

	// set the content of the window and show it
	mainWindow1.SetContent(a.gb1Raster)
	// mainWindow1.SetMainMenu(mainMenu)
	mainWindow1.Show()

	// start the gameboy if ROM is loaded
	if a.gb1.MMU.Cart.MD5 != "" {
		a.StartGameView(a.gb1, mainWindow1)
	} else {
		// toggle the main menu or ask for a ROM (depending on user settings)
		if a.app.Preferences().BoolWithFallback("askForROM", false) {
			rom, err := askForROM()
			if err != nil {
				a.gb1.Logger.Errorf("failed to ask for ROM: %v", err)
			}

			// load the ROM
			a.gb1.MMU.Cart = cartridge.NewCartridge(rom)
		} else {
			a.toggleMainMenu()
		}
	}

	// main menu listener (for escape key)
	mainWindow1.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.toggleMainMenu()
		}
	})

	// run the application
	a.app.Run()

	// close the gameboy on exit
	if !a.gb1.IsPaused() {
		a.gb1.Lock()
		// close the current gameboy
		a.gb1.Close <- struct{}{}
		a.gb1.Close <- struct{}{}
		a.gb1.Unlock()
	}

	// wait for the gameboy to close
	for !a.gb1.IsRunning() {
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

var compressionMap = []byte{
	1: 3,
	2: 5,
	3: 7,
	4: 9,
}

func askForROM() ([]byte, error) {
	// open a file dialog to select a ROM
	fileName, err := dialog2.File().Filter("GameBoy ROMs (*.gb, *.gbc)", "gb", "gbc").Load()
	if err != nil {
		return nil, err
	}

	// read the ROM file
	rom, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	return rom, nil
}

func (a *Application) toggleMainMenu() {
	if a.mainMenu != nil {
		a.mainMenu = nil
		a.mainWindow1.SetMainMenu(nil)
		// reset the window size to current size + menu bar height
		w, h := a.mainWindow1.Content().Size().Width, a.mainWindow1.Content().Size().Height
		a.mainWindow1.Resize(fyne.NewSize(w, h+26))
		a.mainWindow1.Resize(fyne.NewSize(w, h+25)) // TODO why is this needed?

		// unpause the gameboy
		a.gb1.Unpause()
	} else {
		// create submenus
		menuItemOpenROM := fyne.NewMenuItem("Open ROM", func() {
			// open a file dialog to select a ROM
			rom, err := askForROM()
			if err != nil {
				// TODO handle error
				return
			}
			// close the current gameboy if it's running
			if !a.gb1.IsPaused() {
				a.gb1.Lock()
				// close the current gameboy
				a.gb1.Close <- struct{}{}
				a.gb1.Unlock()
			}
			// load the ROM
			a.gb1 = gameboy.NewGameBoy(rom, a.gb1.Options...)

			// start the gameboy
			a.StartGameView(a.gb1, a.mainWindow1)

			// hide the main menu
			a.toggleMainMenu()
		})
		menuItemSaveState := fyne.NewMenuItem("Save State", func() {
			// TODO
		})
		menuItemLoadState := fyne.NewMenuItem("Load State", func() {
			// TODO
		})
		menuItemSettings := fyne.NewMenuItem("Settings", func() {
			// open the settings window
			a.openWindowIfNotOpen(&views.Settings{
				Preferences: a.app.Preferences(),
			})
		})

		// add menu items to submenus
		fileMenu := fyne.NewMenu("File", menuItemOpenROM, menuItemSaveState, menuItemLoadState, fyne.NewMenuItemSeparator(), menuItemSettings)

		// create emulation menu
		emuReset := fyne.NewMenuItem("Reset", func() {

		})
		emuSpeed := fyne.NewMenuItem("Speed", func() {
			// TODO
		})
		emuSpeed.ChildMenu = fyne.NewMenu("",
			fyne.NewMenuItem("0.25x", func() {
				// TODO
			}),
			fyne.NewMenuItem("0.5x", func() {
				// TODO
			}),
			fyne.NewMenuItem("1x", func() {
				// TODO
			}))
		// add 1x - through 4x
		for i := 2; i <= 4; i++ {
			emuSpeed.ChildMenu.Items = append(emuSpeed.ChildMenu.Items, fyne.NewMenuItem(fmt.Sprintf("%dx", i), func() {
				// TODO
			}))
		}

		emuMultiplayer := fyne.NewMenuItem("Multiplayer", func() {
			// TODO
		})
		emuPrinter := fyne.NewMenuItem("Printer", func() {
			a.openWindowIfNotOpen(&views.Printer{})
		})

		emuCheats := fyne.NewMenuItem("Cheats", func() {
			a.openWindowIfNotOpen(views.NewCheatManager(views.WithGameShark(a.gb1.MMU.GameShark), views.WithGameGenie(a.gb1.MMU.GameGenie))) // TODO determine which cheats are enabled
		})

		emuMenu := fyne.NewMenu("Emulation",
			emuReset,
			emuSpeed,
			fyne.NewMenuItemSeparator(),
			emuMultiplayer,
			emuPrinter,
			fyne.NewMenuItemSeparator(),
			emuCheats,
		)

		audioMute := fyne.NewMenuItem("Mute", func() {

		})
		audioMute.Checked = true
		audioChannels := fyne.NewMenuItem("Audio Channels", func() {
			// TODO
		})
		audioChannels.ChildMenu = fyne.NewMenu("",
			fyne.NewMenuItem("1 (Square)", func() {
				// TODO
			}),
			fyne.NewMenuItem("2 (Square)", func() {
				// TODO
			}),
			fyne.NewMenuItem("3 (Wave)", func() {
				// TODO
			}),
			fyne.NewMenuItem("4 (Noise)", func() {
				// TODO
			}),
		)

		// create audio menu
		audioMenu := fyne.NewMenu("Audio",
			audioMute,
			audioChannels,
			fyne.NewMenuItem("Visualizer", func() {
				a.openWindowIfNotOpen(&views.Visualizer{})
			}),
		)

		videoFrameSize := fyne.NewMenuItem("Frame Size", func() {

		})

		videoLayers := fyne.NewMenuItem("Layers", func() {

		})
		videoLayers.ChildMenu = fyne.NewMenu("",
			fyne.NewMenuItem("Background", func() {
				a.gb1.PPU.Debug.BackgroundDisabled = !a.gb1.PPU.Debug.BackgroundDisabled
				videoLayers.ChildMenu.Items[0].Checked = !a.gb1.PPU.Debug.BackgroundDisabled
			}),
			fyne.NewMenuItem("Window", func() {
				a.gb1.PPU.Debug.WindowDisabled = !a.gb1.PPU.Debug.WindowDisabled
				videoLayers.ChildMenu.Items[1].Checked = !a.gb1.PPU.Debug.WindowDisabled
			}),
			fyne.NewMenuItem("Sprites", func() {
				a.gb1.PPU.Debug.SpritesDisabled = !a.gb1.PPU.Debug.SpritesDisabled
				videoLayers.ChildMenu.Items[2].Checked = !a.gb1.PPU.Debug.SpritesDisabled
			}),
		)

		// mark layers that are currently enabled
		videoLayers.ChildMenu.Items[0].Checked = !a.gb1.PPU.Debug.BackgroundDisabled
		videoLayers.ChildMenu.Items[1].Checked = !a.gb1.PPU.Debug.WindowDisabled
		videoLayers.ChildMenu.Items[2].Checked = !a.gb1.PPU.Debug.SpritesDisabled

		videoTakeScreenshot := fyne.NewMenuItem("Take Screenshot", func() {
			// get the current time
			now := time.Now()

			// create the file name
			fileName := fmt.Sprintf("screenshot-%d-%d-%d-%d-%d-%d.png", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

			// create the file
			file, err := os.Create(fileName)
			if err != nil {
				// TODO
			}

			// create the encoder
			encoder := png.Encoder{
				CompressionLevel: png.BestCompression,
			}

			// dump current frame
			img := image.NewRGBA(image.Rect(0, 0, 160, 144))
			for y := 0; y < 144; y++ {
				for x := 0; x < 160; x++ {
					img.Set(x, y, color.RGBA{R: a.gb1.PPU.PreparedFrame[y][x][0], G: a.gb1.PPU.PreparedFrame[y][x][1], B: a.gb1.PPU.PreparedFrame[y][x][2], A: 255})
				}
			}

			// encode the image
			err = encoder.Encode(file, img)
			if err != nil {
				// TODO
			}

			// close the file
			err = file.Close()

			// TODO
		})
		videoRecord := fyne.NewMenuItem("Record", func() {

		})

		videoMenu := fyne.NewMenu("Video",
			videoFrameSize,
			videoLayers,
			fyne.NewMenuItemSeparator(),
			videoTakeScreenshot,
			videoRecord,
		)

		// create debug menu
		debugViews := fyne.NewMenuItem("Views", func() {

		})
		debugViews.ChildMenu = fyne.NewMenu("")

		// add views to debug menu
		for _, view := range []string{"Palette Viewer", "Frame Renderer", "Tile Viewer", "Tilemap Viewer", "", "Cartridge Info"} {
			// copy the view name to a new variable
			newView := view
			if view == "" {
				debugViews.ChildMenu.Items = append(debugViews.ChildMenu.Items, fyne.NewMenuItemSeparator())
				continue
			}
			debugViews.ChildMenu.Items = append(debugViews.ChildMenu.Items, fyne.NewMenuItem(view, func() {
				switch newView {
				case "Palette Viewer":
					a.openWindowIfNotOpen(views.Palette{PPU: a.gb1.PPU})
				case "Frame Renderer":
					a.openWindowIfNotOpen(&views.Render{Video: a.gb1.PPU})
				case "Tile Viewer":
					a.openWindowIfNotOpen(&views.Tiles{PPU: a.gb1.PPU})
				case "Tilemap Viewer":
					a.openWindowIfNotOpen(&views.Tilemaps{PPU: a.gb1.PPU})
				case "Cartridge Info":
					a.openWindowIfNotOpen(&views.Cartridge{C: a.gb1.MMU.Cart})
				}
			}))
		}

		debugMenu := fyne.NewMenu("Debug",
			debugViews,
			fyne.NewMenuItem("Performance", func() {
				a.openWindowIfNotOpen(&views.Performance{})
			}),
		)

		// create help menu
		helpMenu := fyne.NewMenu("Help")

		// create main menu
		mainMenu := fyne.NewMainMenu(
			fileMenu,
			emuMenu,
			audioMenu,
			videoMenu,
			debugMenu,
			helpMenu,
		)
		mainMenu.Refresh()
		a.mainWindow1.SetMainMenu(mainMenu)
		a.mainMenu = mainMenu

		// pause the gameboy
		a.gb1.Pause()
	}
}

func (a *Application) openWindowIfNotOpen(view display.View) {
	// iterate over all windows and see if the window is open
	for _, w := range a.Windows {
		// check if the window is open by asserting type
		if w.view.Title() == view.Title() {
			// window is already open, return
			return
		}
	}

	win := a.NewWindow(view.Title(), view).(fyneWindow)
	win.Show()

	if err := view.Run(win, win.events); err != nil {
		panic(err)
	}
}

func (a *Application) StartGameView(gb *gameboy.GameBoy, window fyne.Window) {
	// setup framebuffer
	fb := make(chan []byte, 144)

	go func() {
		for {
			select {
			case f := <-fb:
				// copy the framebuffer to the image
				for i := 0; i < ppu.ScreenHeight*ppu.ScreenWidth; i++ {
					r, g, b := f[i*3], f[i*3+1], f[i*3+2]
					a.gb1Image.Pix[i*4] = r
					a.gb1Image.Pix[i*4+1] = g
					a.gb1Image.Pix[i*4+2] = b
					a.gb1Image.Pix[i*4+3] = 255
				}

				a.gb1Raster.Refresh()
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
