package main

import (
	"flag"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/audio"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	_ "github.com/thelolagemann/gomeboy/pkg/display/fyne"
	_ "github.com/thelolagemann/gomeboy/pkg/display/glfw"
	//_ "github.com/thelolagemann/gomeboy/pkg/display/web"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"github.com/thelolagemann/gomeboy/pkg/utils"

	"net/http"
	_ "net/http/pprof"
	"strings"
)

var (
	_ display.Emulator = &gameboy.GameBoy{}
)

func main() {
	// start pprof
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			return
		}
	}()

	var logger = log.New()

	if len(display.InstalledDrivers) == 0 {
		logger.Fatal("No display drivers installed. Please compile with at least one display driver")
	}

	romFile := flag.String("rom", "", "The rom file to load")
	bootROM := flag.String("boot", "", "The boot rom file to load")
	// saveFolder := flag.String("save", "", "The folder to ")
	state := flag.String("state", "", "The state file to load") // TODO determine state file from ROM file
	asModel := flag.String("model", "auto", "The model to emulate. Can be auto, dmg or cgb")
	printer := flag.Bool("printer", false, "enable printer")
	displayDriver := flag.String("driver", "auto", "The display driver to use. Can be auto, glfw, fyne or web")
	speed := flag.Float64("speed", 1, "The speed to run the emulator at")

	display.RegisterFlags()
	flag.Parse()

	var rom []byte
	var err error
	if *romFile != "" {
		// open the rom file
		rom, err = utils.LoadFile(*romFile)
		if err != nil {
			panic(err)
		}
	}

	var opts []gameboy.GameBoyOpt
	// open the boot rom file
	var boot []byte
	if *bootROM != "" {
		boot, err = utils.LoadFile(*bootROM)
		if err != nil {
			panic(err)
		}

		opts = append(opts, gameboy.WithBootROM(boot))
	}

	if *printer {
		printer := accessories.NewPrinter()
		opts = append(opts, gameboy.WithPrinter(printer))
	}

	if *state != "" {
		state, err := utils.LoadFile(*state)
		if err != nil {
			panic(err)
		}
		opts = append(opts, gameboy.WithState(state))
	}

	switch strings.ToLower(*asModel) {
	case "auto":
		// no-op
		break
	case "dmg":
		opts = append(opts, gameboy.AsModel(types.DMGABC))
	case "dmg0":
		opts = append(opts, gameboy.AsModel(types.DMG0))
	case "mgb":
		opts = append(opts, gameboy.AsModel(types.MGB))
	case "cgb":
		opts = append(opts, gameboy.AsModel(types.CGBABC))
	case "cgb0":
		opts = append(opts, gameboy.AsModel(types.CGB0))
	case "sgb":
		opts = append(opts, gameboy.AsModel(types.SGB))
	case "sgb2":
		opts = append(opts, gameboy.AsModel(types.SGB2))
	case "agb":
		opts = append(opts, gameboy.AsModel(types.AGB))
	}
	// opts = append(opts, gameboy.SaveEvery(time.Second*10))
	opts = append(opts, gameboy.Speed(*speed))
	// create a new gameboy
	opts = append(opts, gameboy.WithLogger(log.NewNullLogger()))
	opts = append(opts, gameboy.Debug())
	gb := gameboy.NewGameBoy(rom, opts...)

	if err := audio.OpenAudio(); err != nil {
		logger.Errorf("unable to open audio device %s", err)
	} else {
		gb.AttachAudioListener(audio.PlaySDL)
	}

	driver := display.GetDriver(*displayDriver)

	// check to make sure the driver is valid
	if driver == nil {
		logger.Fatal("invalid display driver")
	}

	// attach gameboy to driver
	driver.Initialize(gb)

	// create framebuffer
	fb := make(chan []byte, 60)

	// create various channels
	events := make(chan event.Event, 60)
	pressed := make(chan joypad.Button, 10)
	released := make(chan joypad.Button, 10)

	// start gameboy in a goroutine
	go gb.Start(fb, events, pressed, released)

	if err := driver.Start(fb, events, pressed, released); err != nil {
		logger.Fatal(err.Error())
	}
}
