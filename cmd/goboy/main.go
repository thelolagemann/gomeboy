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
	"strings"
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

	log := views.Log{}

	romFile := flag.String("rom", "", "The rom file to load")
	bootROM := flag.String("boot", "", "The boot rom file to load")
	asModel := flag.String("model", "auto", "The model to emulate. Can be auto, dmg or cgb")
	debugViews := flag.Bool("debug", false, "Show debug views")
	activeDebugViews := flag.String("active-debug", "cpu,log,mmu,ppu,vram", "Comma separated list of debug views to show")
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
	opts = append(opts, gameboy.WithLogger(&log))
	gb := gameboy.NewGameBoy(rom, opts...)

	a := fyne.NewApplication(app.NewWithID("com.github.thelolagemann.gomeboy"), gb)

	if *debugViews {
		for _, view := range strings.Split(*activeDebugViews, ",") {
			switch view {
			case "cpu":
				a.NewWindow("CPU", views.NewCPU(gb.CPU))
			case "ppu":
				a.NewWindow("PPU", views.NewPPU(gb.PPU))
			case "mmu":
				a.NewWindow("MMU", views.NewMMU(gb.MMU))
			case "vram":
				a.NewWindow("VRAM", &views.VRAM{PPU: gb.PPU})
			case "system":
				a.NewWindow("System", &views.System{})
			case "log":
				a.NewWindow("Log", &log)
			}
		}
	}

	log.Infof("Loaded rom %s", *romFile)

	a.Run()
}
