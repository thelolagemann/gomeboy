package main

import (
	"flag"
	"fyne.io/fyne/v2/app"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/pkg/display/fyne"
	"github.com/thelolagemann/go-gameboy/pkg/display/views"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
	"net/http"
	_ "net/http/pprof"
	"time"
)

func main() {
	// start pprof
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			return
		}
	}()

	// log := display.NewLog()

	romFile := flag.String("rom", "", "The rom file to load")
	bootROM := flag.String("boot", "", "The boot rom file to load")
	asModel := flag.String("model", "auto", "The model to emulate. Can be auto, dmg or cgb")
	flag.Parse()

	// open the rom file
	rom, err := utils.LoadFile(*romFile)
	if err != nil {
		panic(err)
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

	switch *asModel {
	case "auto":
		// no-op
		break
	case "dmg":
		opts = append(opts, gameboy.AsModel(gameboy.ModelDMG))
	case "cgb":
		opts = append(opts, gameboy.AsModel(gameboy.ModelCGB))
	}
	opts = append(opts, gameboy.SaveEvery(time.Second*10))
	// create a new gameboy
	// opts = append(opts, gameboy.WithLogger(log))
	gb := gameboy.NewGameBoy(rom, opts...)

	a := fyne.NewApplication(app.NewWithID("com.github.thelolagemann.gomeboy"), gb)

	// TODO make optional
	a.NewWindow("CPU", views.NewCPU(gb.CPU))
	a.NewWindow("PPU", views.NewPPU(gb.PPU))
	a.NewWindow("MMU", views.NewMMU(gb.MMU))

	a.Run()
}
