package main

import (
	"flag"
	"fmt"
	"github.com/faiface/pixel/pixelgl"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"net/http"
	"os"

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
	flag.Parse()

	// open the rom file
	f, err := os.Open(*romFile)
	if err != nil {
		panic(err)
	}

	// read the rom file into a byte slice
	rom := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := f.Read(buf)
		if err != nil {
			break
		}
		rom = append(rom, buf[:n]...)
	}

	// create a new gameboy
	gb := gameboy.NewGameBoy(rom)

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
