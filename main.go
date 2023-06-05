package main

import (
	"flag"
	"fyne.io/fyne/v2/app"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/serial/accessories"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/display/fyne"
	"github.com/thelolagemann/go-gameboy/pkg/display/views"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
	"net/http"
	_ "net/http/pprof"
	"strings"
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

	romFile := flag.String("rom", "", "The rom file to load")
	bootROM := flag.String("boot", "", "The boot rom file to load")
	// saveFolder := flag.String("save", "", "The folder to ")
	state := flag.String("state", "", "The state file to load") // TODO determine state file from ROM file
	asModel := flag.String("model", "auto", "The model to emulate. Can be auto, dmg or cgb")
	debugViews := flag.Bool("debug", false, "Show debug views")
	activeDebugViews := flag.String("active-debug", "vram", "Comma separated list of debug views to show")
	dualView := flag.Bool("dual", false, "Show dual view")
	printer := flag.Bool("printer", false, "enable printer")
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

	a := fyne.NewApplication(app.NewWithID("com.thelolagemann.gomeboy"), gb)

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
				a.NewWindow("Tiles", &views.Tiles{PPU: gb.PPU})
			case "logger":
				l := &views.Log{}
				gb.Logger = logger
				a.NewWindow("Log", l)
			case "system":
				a.NewWindow("System", &views.System{})
			case "render":
				a.NewWindow("Render", &views.Render{Video: gb.PPU})
			}
		}
	}

	if *printer {
		a.NewWindow("Printer", &views.Printer{Printer: gb.Printer, DrawMode: 1})
	}

	logger.Infof("Loaded rom %s", *romFile)

	if *dualView {
		opts = append(opts, gameboy.SerialConnection(gb))
		// create a new gameboy
		gb2 := gameboy.NewGameBoy(rom, opts...)
		a.AddGameBoy(gb2)
	}

	if err := a.Run(); err != nil {
		panic(err)
	}
}
