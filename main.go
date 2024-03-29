package main

import (
	"flag"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/audio"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	_ "github.com/thelolagemann/gomeboy/pkg/display/fyne"
	_ "github.com/thelolagemann/gomeboy/pkg/display/glfw"
	_ "github.com/thelolagemann/gomeboy/pkg/display/web"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"time"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	// init display package
	display.Init()

	// start pprof
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			return
		}
	}()

	var logger = log.New()

	romFile := flag.String("rom", "", "The rom file to load")
	bootROM := flag.String("boot", "", "The boot rom file to load")
	// saveFolder := flag.String("save", "", "The folder to ")
	asModel := flag.String("model", "auto", "The model to emulate. Can be auto, dmg or cgb")
	printer := flag.Bool("printer", false, "enable printer")
	displayDriver := flag.String("driver", "auto", "The display driver to use. Can be auto, glfw, fyne or web")
	speed := flag.Float64("speed", 1, "The speed to run the emulator at")

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

	var opts []gameboy.Opt
	if *bootROM != "" {
		boot, err := utils.LoadFile(*bootROM)
		if err != nil {
			panic(err)
		}

		opts = append(opts, gameboy.WithBootROM(boot))
	}

	if *printer {
		printer := accessories.NewPrinter()
		opts = append(opts, gameboy.WithPrinter(printer))
	}

	// has model been set?
	if *asModel != "auto" {
		opts = append(opts, gameboy.AsModel(types.StringToModel(*asModel)))
	}

	// opts = append(opts, gameboy.SaveEvery(time.Second*10))
	opts = append(opts, gameboy.Speed(*speed))
	// create a new gameboy
	opts = append(opts, gameboy.WithLogger(logger))
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
	pressed := make(chan io.Button, 10)
	released := make(chan io.Button, 10)

	// start gameboy in a goroutine
	go gb.Start(fb, events, pressed, released)

	if err := driver.Start(fb, events, pressed, released); err != nil {
		logger.Fatal(err.Error())
	}

	gb.SendCommand(display.Close)
	// wait until gb is no longer running
	for {
		if gb.State() == emulator.Stopped {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
}
