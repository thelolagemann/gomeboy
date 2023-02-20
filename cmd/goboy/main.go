package main

import (
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
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
	fmt.Println(gb.MMU.Cart.Title())

	a := app.New()
	w := a.NewWindow("GomeBoy")
	w.Resize(fyne.NewSize(ppu.ScreenWidth*4, ppu.ScreenHeight*4))

	w.SetMaster()
	w.Show()

	w2 := a.NewWindow("CPU")
	w2.Show()

	w3 := a.NewWindow("PPU")
	w3.Show()

	c := views.NewCPU(gb.CPU)
	g := views.NewPPU(gb.PPU)

	go func() {
		if err := gb.Run(w); err != nil {
			panic(err)
		}
	}()
	go func() {
		if err := c.Run(w2); err != nil {
			panic(err)
		}
	}()
	go func() {
		if err := g.Run(w3); err != nil {
			panic(err)
		}
	}()

	a.Run()
}
