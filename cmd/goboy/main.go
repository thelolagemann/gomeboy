package main

import (
	"flag"
	"github.com/faiface/pixel/pixelgl"
	"github.com/thelolagemann/go-gameboy/internal/display"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"os"
)

func main() {
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
		mon := display.NewDisplay()
		// start the gameboy
		gb.Start(mon)
	})
}
