//go:build !test

package fyne

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/fyne/themes"
	"github.com/thelolagemann/gomeboy/pkg/display/fyne/views"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"image"
	"image/color"
	"image/png"
	"os"
	"sync"
	"time"
)

func init() {
	driver := &fyneDriver{
		windows: make(map[string]fyne.Window),
	}
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
	app            fyne.App
	mainMenu       *fyne.MainMenu
	mainMenuOpened bool
	mainWindow     fyne.Window
	gb             *gameboy.GameBoy

	windows    map[string]fyne.Window
	fullscreen bool
	scale      float64
}

func (f *fyneDriver) Start(c emulator.Controller, fb <-chan []byte, pressed chan<- io.Button, released chan<- io.Button) error {
	f.createMainMenu()
	// create new fyne application
	f.app = app.NewWithID("gomeboy.thelolagemann.com")
	f.app.Settings().SetTheme(themes.Default{})

	// create main window
	mainWindow := f.app.NewWindow("GomeBoy")
	mainWindow.SetMaster()
	mainWindow.Resize(fyne.NewSize(float32(ppu.ScreenWidth*f.scale), float32(ppu.ScreenHeight*f.scale)))
	mainWindow.SetPadded(false)

	f.mainWindow = mainWindow

	// create image
	img := image.NewRGBA(image.Rect(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))

	// create canvas to draw to
	raster := canvas.NewRasterFromImage(img)
	raster.ScaleMode = canvas.ImageScalePixels
	raster.SetMinSize(fyne.NewSize(ppu.ScreenWidth, ppu.ScreenHeight))

	if !f.gb.Initialised() {
		f.toggleMainMenu()
	}
	mainWindow.SetContent(raster)

	mainWindow.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		if len(uris) != 1 {
			return // only support loading 1 ROM
		}
		f.error(f.openROM(uris[0].Path()))
	})
	// set the content of the window
	mainWindow.Show()

	// setup menu
	mainWindow.Canvas().SetOnTypedKey(func(event *fyne.KeyEvent) {
		if c.Initialised() && event.Name == fyne.KeyEscape {
			f.toggleMainMenu()
		}
	})

	// Create variables to track FPS and frametime
	var frameCount int
	var lastTime time.Time

	// Setup a ticker to update FPS/Frametime every second
	ticker := time.NewTicker(16675004)
	defer ticker.Stop()
	var latestFrames [][]byte
	var frameMutex sync.Mutex
	go func() {
		for {
			select {
			case b := <-fb:
				frameMutex.Lock()
				latestFrames = append(latestFrames, b)
				frameMutex.Unlock()
			}
		}
	}()

	// Setup the goroutine to render frames and calculate FPS and frametime
	go func() {
		for {
			select {
			case <-ticker.C:
				if f.gb.Paused() {
					continue
				}
				frameMutex.Lock()

				// Process frame
				if len(latestFrames) > 0 {
					latestFrame := latestFrames[0]
					latestFrames = latestFrames[1:]

					// Update image with the latest frame
					for i := 0; i < ppu.ScreenWidth*ppu.ScreenHeight; i++ {
						img.Pix[i*4] = latestFrame[i*3]
						img.Pix[i*4+1] = latestFrame[i*3+1]
						img.Pix[i*4+2] = latestFrame[i*3+2]
						img.Pix[i*4+3] = 255
					}
					// refresh canvas
					raster.Refresh()
				}

				// Send frame event to windows
				for _, w := range f.windows {
					w.Content().Refresh()
				}

				// Calculate FPS and frametime
				frameCount++
				elapsedTime := time.Since(lastTime)

				// Update FPS and frametime every second
				if elapsedTime >= time.Second {
					fps := float64(frameCount) / elapsedTime.Seconds()
					mainWindow.SetTitle(fmt.Sprintf("FPS: %.2f", fps))

					// Reset counters for next interval
					frameCount = 0
					lastTime = time.Now()
				}

				frameMutex.Unlock()
			}
		}
	}()

	// handle input
	if desk, ok := mainWindow.Canvas().(desktop.Canvas); ok {
		desk.SetOnKeyDown(func(e *fyne.KeyEvent) {
			// check if this is a Game Boy key
			if k, isMapped := keyMap[e.Name]; isMapped {
				pressed <- k
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

func (f *fyneDriver) AttachGameboy(b *gameboy.GameBoy) { f.gb = b }

func (f *fyneDriver) error(err error) {
	if err != nil {
		d := dialog.NewError(err, f.mainWindow)
		d.Show()
	}
}

func (f *fyneDriver) openROM(path string) error {
	recreate := false
	if !f.gb.Initialised() {
		recreate = true
	}
	// close all children windows
	for _, w := range f.windows {
		w.Close()
	}
	if err := f.gb.LoadROM(path); err != nil {
		return err
	}
	if f.mainMenuOpened {
		f.toggleMainMenu()
	}
	if recreate {
		f.createMainMenu()
	}
	return nil
}

func (f *fyneDriver) createMainMenu() {
	// create main menu
	f.mainMenu = fyne.NewMainMenu()

	// create submenus
	menuItemOpenROM := fyne.NewMenuItem("Open ROM", func() {
		d := dialog.NewFileOpen(func(closer fyne.URIReadCloser, err error) {
			if err != nil {
				f.error(err)
				return
			}
			if closer == nil {
				return // user cancelled
			}
			f.error(f.openROM(closer.URI().Path()))
		}, f.mainWindow)
		d.SetFilter(storage.NewExtensionFileFilter([]string{".gb", ".gbc", ".7z", ".zip", ".gz", ".xz"}))
		d.Show()
	})

	// add menu items to submenus
	fileMenu := fyne.NewMenu("File", menuItemOpenROM)

	emuMenu := fyne.NewMenu("Emulation",
		NewCustomizedMenuItem("Reset", func() {
			// close all children windows
			for _, w := range f.windows {
				w.Close()
			}
			f.gb.Init() // TODO implement resettable
			f.toggleMainMenu()
		}, Gated(!f.gb.Initialised())),
		fyne.NewMenuItemSeparator(),
		NewCustomizedMenuItem("Camera", func() {
			f.openWindowIfNotOpen("Camera", views.NewCamera(f.gb.Bus.Cartridge().Camera, f.gb.PPU))
		}, Gated(!(f.gb.Initialised() && f.gb.Bus.Cartridge().CartridgeType == io.POCKETCAMERA))),
		NewCustomizedMenuItem("Printer", func() {
			// create and attach printer if gameboy doesn't have one attached
			if _, ok := f.gb.Serial.AttachedDevice.(*accessories.Printer); !ok {
				printer := accessories.NewPrinter()
				f.gb.Serial.Attach(printer)
			}
			f.openWindowIfNotOpen("Printer", views.NewPrinter(f.gb.Serial.AttachedDevice.(*accessories.Printer)))
		}, Gated(!f.gb.Initialised())),
		fyne.NewMenuItemSeparator(),
		NewCustomizedMenuItem("Cheats", func() { f.openWindowIfNotOpen("Cheats", views.NewCheats(f.gb.Bus)) }, Gated(!f.gb.Initialised())),
	)

	audioChannels := NewCustomizedMenuItem("Audio Channels", func() {}, Gated(!f.gb.Initialised()))
	audioChannels.ChildMenu = fyne.NewMenu("",
		NewCustomizedMenuItem("1 (Square)", func() { f.gb.APU.Debug.Square1 = !f.gb.APU.Debug.Square1 }, Checked(true, f.mainMenu.Refresh)),
		NewCustomizedMenuItem("2 (Square)", func() { f.gb.APU.Debug.Square2 = !f.gb.APU.Debug.Square2 }, Checked(true, f.mainMenu.Refresh)),
		NewCustomizedMenuItem("3 (Wave)", func() { f.gb.APU.Debug.Wave = !f.gb.APU.Debug.Wave }, Checked(true, f.mainMenu.Refresh)),
		NewCustomizedMenuItem("4 (Noise)", func() { f.gb.APU.Debug.Noise = !f.gb.APU.Debug.Noise }, Checked(true, f.mainMenu.Refresh)),
	)
	audioMenu := fyne.NewMenu("Audio",
		NewCustomizedMenuItem("Mute", func() { f.gb.APU.ToggleMute() }, Checked(false, f.mainMenu.Refresh), Gated(!f.gb.Initialised())),
		audioChannels,
		NewCustomizedMenuItem("Visualiser", func() { f.openWindowIfNotOpen("Visualiser", views.NewVisualiser(f.gb.APU)) }, Gated(!f.gb.Initialised())),
	)

	videoLayers := NewCustomizedMenuItem("Layers", func() {}, Gated(!f.gb.Initialised()))
	videoTakeScreenshot := fyne.NewMenuItem("Take Screenshot", func() {
		// create the file name
		now := time.Now()
		fileName := fmt.Sprintf("screenshot-%d-%d-%d-%d-%d-%d.png", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

		// create the file
		file, err := os.Create(fileName)
		if err != nil {
			f.error(err)
			return
		}

		// dump current frame
		img := image.NewNRGBA(image.Rect(0, 0, 160, 144))
		for y := 0; y < 144; y++ {
			for x := 0; x < 160; x++ {
				img.Set(x, y, color.NRGBA{R: f.gb.PPU.PreparedFrame[y][x][0], G: f.gb.PPU.PreparedFrame[y][x][1], B: f.gb.PPU.PreparedFrame[y][x][2], A: 255})
			}
		}

		// encode the image
		err = png.Encode(file, img)
		if err != nil {
			f.error(err)
			return
		}

		// close the file
		f.error(file.Close())
	})

	videoMenu := fyne.NewMenu("Video",
		videoLayers,
		fyne.NewMenuItemSeparator(),
		videoTakeScreenshot,
	)
	videoLayers.ChildMenu = fyne.NewMenu("",
		NewCustomizedMenuItem("Background", func() { f.gb.PPU.Debug.BackgroundDisabled = !f.gb.PPU.Debug.BackgroundDisabled }, Checked(true, videoMenu.Refresh)),
		NewCustomizedMenuItem("Window", func() { f.gb.PPU.Debug.WindowDisabled = !f.gb.PPU.Debug.WindowDisabled }, Checked(true, videoMenu.Refresh)),
		NewCustomizedMenuItem("Sprites", func() { f.gb.PPU.Debug.OBJDisabled = !f.gb.PPU.Debug.OBJDisabled }, Checked(true, videoMenu.Refresh)),
	)

	// create debug menu
	debugViews := NewCustomizedMenuItem("Views", func() {}, Gated(!f.gb.Initialised()))
	debugViews.ChildMenu = fyne.NewMenu("")

	type debugContentView struct {
		name string
		fn   func() fyne.CanvasObject
	}
	debugContent := []debugContentView{
		{"CPU", func() fyne.CanvasObject { return views.NewCPU(f.gb.CPU, f.gb.Bus) }},
		{"Palette Viewer", func() fyne.CanvasObject { return views.NewPalette(f.gb.PPU) }},
		{"Tile Viewer", func() fyne.CanvasObject { return views.NewTiles(f.gb.PPU, f.gb.Bus) }},
		{"Tilemap Viewer", func() fyne.CanvasObject { return views.NewTilemaps(f.gb.PPU, f.gb.Bus) }},
		{"OAM", func() fyne.CanvasObject { return views.NewOAM(f.gb.PPU, f.gb.Bus) }},
		{"Cartridge Info", func() fyne.CanvasObject { return views.NewCartridge(f.gb.Bus.Cartridge()) }},
		{"Memory Viewer", func() fyne.CanvasObject { return views.NewMemory(f.gb.Bus) }},
		{"IO", func() fyne.CanvasObject { return views.NewIO(f.gb.Bus) }},
	}

	// add views to debug menu
	debugViews.ChildMenu.Items = make([]*fyne.MenuItem, len(debugContent))
	for i, view := range debugContent {
		debugViews.ChildMenu.Items[i] = fyne.NewMenuItem(view.name, func() { f.openWindowIfNotOpen(view.name, view.fn()) })
	}
	debugMenu := fyne.NewMenu("Debug", debugViews)

	f.mainMenu.Items = []*fyne.Menu{fileMenu, emuMenu, audioMenu, videoMenu, debugMenu}
}

func (f *fyneDriver) toggleMainMenu() {
	if f.mainMenuOpened {
		// if the main menu is already open, close it
		f.mainMenuOpened = false
		f.mainWindow.SetMainMenu(nil)

		// workaround to reset the window size to current size + menu bar height
		w, h := f.mainWindow.Content().Size().Width, f.mainWindow.Content().Size().Height
		f.mainWindow.Resize(fyne.NewSize(w, h+26))
		f.mainWindow.Resize(fyne.NewSize(w, h+25))
		f.mainWindow.Content().Refresh()

		f.gb.Resume()
	} else {
		f.mainMenuOpened = true
		f.mainWindow.SetMainMenu(f.mainMenu)

		// pause the gameboy
		f.gb.Pause()
	}
}

// openWindowIfNotOpen opens a window if it is not already open.
func (f *fyneDriver) openWindowIfNotOpen(name string, view fyne.CanvasObject) {
	// is the window already open
	if _, ok := f.windows[name]; ok {
		return
	}

	// create new window
	win := f.app.NewWindow(name)
	win.SetOnClosed(func() { delete(f.windows, name) })

	win.SetContent(view)
	view.Refresh()
	win.Show()
	f.windows[name] = win
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
