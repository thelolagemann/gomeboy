//go:build !test

package fyne

import "C"
import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	dialog2 "github.com/sqweek/dialog"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	views2 "github.com/thelolagemann/gomeboy/pkg/display/fyne/views"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

func init() {
	driver := &fyneDriver{}
	display.Install("fyne", driver, []display.DriverOption{
		{
			Name:        "fullscreen",
			Type:        "bool",
			Default:     false,
			Description: "Run in fullscreen mode",
			Value:       &driver.fullscreen,
		},
		{
			Name:        "scale",
			Type:        "float",
			Default:     4.0,
			Description: "Scale factor for the display",
			Value:       &driver.scale,
		},
	})
}

type fyneDriver struct {
	app        fyne.App
	mainMenu   *fyne.MainMenu
	mainWindow fyne.Window
	//gb         *gameboy.GameBoy
	gb display.Emulator

	windows []*fyneWindow

	fullscreen bool
	scale      float64
}

func (f *fyneDriver) Initialize(gb display.Emulator) {
	f.gb = gb
}

func (f *fyneDriver) Start(fb <-chan []byte, evts <-chan event.Event, pressed chan<- io.Button, released chan<- io.Button) error {
	// create new fyne application
	fyneApp := app.NewWithID("gomeboy.thelolagemann.com")

	// set default theme
	fyneApp.Settings().SetTheme(&defaultTheme{})
	f.app = fyneApp

	// create main window
	mainWindow := fyneApp.NewWindow("GomeBoy")
	mainWindow.SetMaster()
	mainWindow.Resize(fyne.NewSize(ppu.ScreenWidth*4, ppu.ScreenHeight*4))
	mainWindow.SetPadded(false)
	f.mainWindow = mainWindow

	// create image
	img := image.NewRGBA(image.Rect(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))

	// create canvas to draw to
	raster := canvas.NewRasterFromImage(img)
	raster.ScaleMode = canvas.ImageScalePixels
	raster.SetMinSize(fyne.NewSize(ppu.ScreenWidth, ppu.ScreenHeight))

	// set the content of the window
	mainWindow.SetContent(raster)
	mainWindow.Show()

	// setup menu
	mainWindow.Canvas().SetOnTypedKey(func(event *fyne.KeyEvent) {
		if event.Name == fyne.KeyEscape {
			f.toggleMainMenu()
		}
	})

	// setup goroutine to copy from the framebuffer to the image
	go func() {
		for {
			select {
			case b := <-fb:
				for i := 0; i < ppu.ScreenWidth*ppu.ScreenHeight; i++ {
					img.Pix[i*4] = b[i*3]
					img.Pix[i*4+1] = b[i*3+1]
					img.Pix[i*4+2] = b[i*3+2]
					img.Pix[i*4+3] = 255
				}

				// refresh canvas
				raster.Refresh()

				// send frame event to windows
				for _, w := range f.windows {
					w.events <- event.Event{Type: event.FrameTime}
				}
			}
		}
	}()

	// setup goroutine to handle evts
	go func() {
		for {
			select {
			case e := <-evts:
				switch e.Type {
				case event.Title:
					// only the main window cares about the title event
					mainWindow.SetTitle(e.Data.(string))
				case event.Quit:
					// TODO handle quit event
				default:
					// TODO send to rest of the windows
					for _, w := range f.windows {
						w.events <- e
					}
				}
			}
		}
	}()

	// handle input
	if desk, ok := mainWindow.Canvas().(desktop.Canvas); ok {
		desk.SetOnKeyDown(func(e *fyne.KeyEvent) {
			// check if this is a Game Boy key or event handler
			if k, isMapped := keyMap[e.Name]; isMapped {
				pressed <- k
			} else if h, isHandled := keyHandlers[e.Name]; isHandled {
				// TODO handle key handlers
				h(f.gb.(*gameboy.GameBoy))
			}
		})
		desk.SetOnKeyUp(func(e *fyne.KeyEvent) {
			if k, isMapped := keyMap[e.Name]; isMapped {
				released <- k
			}
		})
	}

	// run the application
	f.app.Run()

	return nil
}

func (f *fyneDriver) Stop() error {
	f.app.Quit()

	return nil
}

func (f *fyneDriver) toggleMainMenu() {
	if f.mainMenu != nil {
		// if the main menu is already open, close it
		f.mainMenu = nil
		f.mainWindow.SetMainMenu(nil)

		// workaround to reset the window size to current size + menu bar height
		w, h := f.mainWindow.Content().Size().Width, f.mainWindow.Content().Size().Height
		f.mainWindow.Resize(fyne.NewSize(w, h+26))
		f.mainWindow.Resize(fyne.NewSize(w, h+25)) // TODO why is this needed?

		f.gb.SendCommand(display.Resume)
	} else {
		// get reference to underlying gb (for now, this should be handled by display.Emulator in the future)
		gb := f.gb.(*gameboy.GameBoy)

		// create main menu
		// create submenus
		menuItemOpenROM := fyne.NewMenuItem("Open ROM", func() {
			// open a file dialog to select a ROM
			rom, err := askForROM()
			if err != nil {
				// TODO handle error
				return
			}
			// close the current gamebo
			//y if it's running
			if f.gb.State().IsRunning() {
				// close the current gameboy
				f.gb.SendCommand(display.Close)
			}
			if res := f.gb.SendCommand(emulator.CommandPacket{Command: emulator.CommandLoadROM, Data: rom}); res.Error != nil {
				// TODO handle error
				return
			}

			// TODO recreate gameboy
			// hide the main menu
			f.toggleMainMenu()
		})
		menuItemSaveState := fyne.NewMenuItem("Save State", func() {
			// TODO
		})
		menuItemLoadState := fyne.NewMenuItem("Load State", func() {
			// TODO
		})
		menuItemSettings := fyne.NewMenuItem("Settings", func() {
			// open the settings window
			f.openWindowIfNotOpen(&views2.Settings{
				Preferences: f.app.Preferences(),
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
			f.openWindowIfNotOpen(&views2.Printer{})
		})

		emuCheats := fyne.NewMenuItem("Cheats", func() {
			//f.openWindowIfNotOpen(views2.NewCheatManager(views2.WithGameShark(gb.MMU.GameShark), views2.WithGameGenie(gb.MMU.GameGenie))) // TODO determine which cheats are enabled
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
				f.openWindowIfNotOpen(&views2.Visualizer{})
			}),
		)

		videoFrameSize := fyne.NewMenuItem("Frame Size", func() {

		})

		videoLayers := fyne.NewMenuItem("Layers", func() {

		})
		videoLayers.ChildMenu = fyne.NewMenu("",
			fyne.NewMenuItem("Background", func() {
				gb.PPU.Debug.BackgroundDisabled = !gb.PPU.Debug.BackgroundDisabled
				videoLayers.ChildMenu.Items[0].Checked = !gb.PPU.Debug.BackgroundDisabled
			}),
			fyne.NewMenuItem("Window", func() {
				gb.PPU.Debug.WindowDisabled = !gb.PPU.Debug.WindowDisabled
				videoLayers.ChildMenu.Items[1].Checked = !gb.PPU.Debug.WindowDisabled
			}),
			fyne.NewMenuItem("Sprites", func() {
				gb.PPU.Debug.SpritesDisabled = !gb.PPU.Debug.SpritesDisabled
				videoLayers.ChildMenu.Items[2].Checked = !gb.PPU.Debug.SpritesDisabled
			}),
		)

		// mark layers that are currently enabled
		videoLayers.ChildMenu.Items[0].Checked = !gb.PPU.Debug.BackgroundDisabled
		videoLayers.ChildMenu.Items[1].Checked = !gb.PPU.Debug.WindowDisabled
		videoLayers.ChildMenu.Items[2].Checked = !gb.PPU.Debug.SpritesDisabled

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
					img.Set(x, y, color.RGBA{R: gb.PPU.PreparedFrame[y][x][0], G: gb.PPU.PreparedFrame[y][x][1], B: gb.PPU.PreparedFrame[y][x][2], A: 255})
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
		for _, view := range []string{"Palette Viewer", "Frame Renderer", "Tile Viewer", "Tilemap Viewer", "OAM", "", "Cartridge Info"} {
			// copy the view name to a new variable
			newView := view
			if view == "" {
				debugViews.ChildMenu.Items = append(debugViews.ChildMenu.Items, fyne.NewMenuItemSeparator())
				continue
			}
			debugViews.ChildMenu.Items = append(debugViews.ChildMenu.Items, fyne.NewMenuItem(view, func() {
				switch newView {
				case "Palette Viewer":
					f.openWindowIfNotOpen(views2.Palette{PPU: gb.PPU})
				case "Frame Renderer":
					f.openWindowIfNotOpen(&views2.Render{Video: gb.PPU})
				case "Tile Viewer":
					f.openWindowIfNotOpen(&views2.Tiles{PPU: gb.PPU})
				case "OAM":
					f.openWindowIfNotOpen(&views2.OAM{PPU: gb.PPU})
				case "Tilemap Viewer":
					f.openWindowIfNotOpen(&views2.Tilemaps{PPU: gb.PPU})
				case "Cartridge Info":
					//f.openWindowIfNotOpen(&views2.Cartridge{C: gb.MMU.Cart})
				}
			}))
		}

		debugMenu := fyne.NewMenu("Debug",
			debugViews,
			fyne.NewMenuItem("Performance", func() {
				f.openWindowIfNotOpen(&views2.Performance{})
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
		f.mainWindow.SetMainMenu(mainMenu)
		f.mainMenu = mainMenu

		// pause the gameboy
		f.gb.SendCommand(display.Pause)
	}
}

// openWindowIfNotOpen opens a window if it is not already open.
func (f *fyneDriver) openWindowIfNotOpen(view View) {
	// iterate over all windows to see if the window is already open
	for _, w := range f.windows {
		if w.view.Title() == view.Title() {
			// window is already open so we can return
			return
		}
	}

	// create new window
	win := f.newWindow(view.Title(), view).(fyneWindow)
	win.Show()
	if err := view.Run(win, win.events); err != nil {
		panic(err)
	}
}

// newWindow creates a new window with the given name and view.
func (f *fyneDriver) newWindow(name string, view View) fyne.Window {
	w := f.app.NewWindow(name)
	b := fyneWindow{
		Window: w,
		view:   view,
		events: make(chan event.Event, 144),
	}
	f.windows = append(f.windows, &b)
	w.SetOnClosed(func() {
		close(b.events)
		for i, win := range f.windows {
			if win == &b {
				f.windows = append(f.windows[:i], f.windows[i+1:]...)
			}
		}
	})

	return b
}

var keyMap = map[fyne.KeyName]io.Button{
	fyne.KeyA:         io.ButtonA,
	fyne.KeyB:         io.ButtonB,
	fyne.KeyUp:        io.ButtonUp,
	fyne.KeyDown:      io.ButtonDown,
	fyne.KeyLeft:      io.ButtonLeft,
	fyne.KeyRight:     io.ButtonRight,
	fyne.KeyReturn:    io.ButtonStart,
	fyne.KeyBackspace: io.ButtonSelect,
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
}

type fyneWindow struct {
	fyne.Window
	view   View
	events chan event.Event
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

// TODO
// - add a way to close windows
// - implement Resettable interface for remaining components (apu, cpu, interrupts, joypad, mmu, ppu, timer, types)
