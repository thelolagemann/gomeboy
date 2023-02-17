package main

import (
	"flag"
	"fmt"
	"github.com/faiface/pixel/pixelgl"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
	"net/http"
	"time"

	_ "net/http/pprof"
)

func main() {
	// start pprof
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			return
		}
	}()

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
	gb := gameboy.NewGameBoy(rom, opts...)

	pixelgl.Run(func() {
		// create a new pixel binding
		mon := display.NewDisplay(gb.MMU.Cart.Header().String())

		// render boot animation
		// mon.RenderBootAnimation()
		fmt.Println("Boot animation finished")

		// start the gameboy
		gb.Start(mon)
	})
}
