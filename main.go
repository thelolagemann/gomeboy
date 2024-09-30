package main

import (
	"flag"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/audio"
	"github.com/thelolagemann/gomeboy/pkg/display"
	_ "github.com/thelolagemann/gomeboy/pkg/display/fyne"
	_ "github.com/thelolagemann/gomeboy/pkg/display/glfw"
	//_ "github.com/thelolagemann/gomeboy/pkg/display/web"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	// init display package
	display.Init()

	// start pprof
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			return
		}
	}()

	var logger = log.New()
	// create framebuffer
	fb := make(chan []byte, 120)

	// create various channels
	pressed := make(chan io.Button, 1)
	released := make(chan io.Button, 1)

	romFile := flag.String("rom", "", "The rom file to load")
	bootROM := flag.String("boot", "", "The boot rom file to load")
	// saveFolder := flag.String("save", "", "The folder to ")
	asModel := flag.String("model", "auto", "The model to emulate. Can be auto, dmg or cgb")
	printer := flag.Bool("printer", false, "enable printer")
	displayDriver := flag.String("driver", "auto", "The display driver to use. Can be auto, glfw, fyne or web")

	flag.Parse()

	var gb *gameboy.GameBoy
	var opts []gameboy.Opt

	if *bootROM != "" {
		boot, err := utils.LoadFile(*bootROM)
		if err != nil {
			panic(err)
		}

		opts = append(opts, gameboy.WithBootROM(boot))
	}

	if *printer {
		opts = append(opts, gameboy.WithPrinter())
	}

	// has model been set?
	if *asModel != "auto" {
		opts = append(opts, gameboy.AsModel(types.StringToModel(*asModel)))
	}

	// opts = append(opts, gameboy.SaveEvery(time.Second*10))
	opts = append(opts)
	// create a new gameboy
	gb = gameboy.NewGameBoy(opts...)

	if *romFile != "" {
		if err := gb.LoadROM(*romFile); err != nil {
			logger.Errorf("unable to load ROM %s: %s", *romFile, err)
		}
	}

	if err := audio.OpenAudio(gb, fb); err != nil {
		logger.Errorf("unable to open audio device %s", err)
	}

	driver := display.GetDriver(*displayDriver)

	// check to make sure the driver is valid
	if driver == nil {
		logger.Fatal("invalid display driver")
	}

	// is the driver capable of debugging?
	if debugger, ok := driver.(display.DriverDebugger); ok {
		debugger.AttachGameboy(gb)
	}

	// handle input
	go func() {
		for {
			select {
			case b := <-pressed:
				gb.Bus.Press(b)
			case b := <-released:
				gb.Bus.Release(b)
			}
		}
	}()

	// start the display driver (blocking)
	if err := driver.Start(gb, fb, pressed, released); err != nil {
		logger.Fatal(err.Error())
	}

	// save after the driver has stopped TODO important stop audio from driving gameboy somehow
	if err := gb.Save(); err != nil {
		logger.Fatal(fmt.Sprintf("unable to save: %v", err))
	}

}
