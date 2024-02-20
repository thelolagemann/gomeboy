package gameboy

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"strings"
)

// Opt is a function that modifies a GameBoy
// instance.
type Opt func(gb *GameBoy)

// Debug
func Debug() Opt {
	return func(gb *GameBoy) {
		gb.CPU.Debug = true
	}
}

func NoAudio() Opt {
	return func(gb *GameBoy) {
		gb.APU.Pause()
	}
}

func SerialDebugger(output *string) Opt {
	return func(gb *GameBoy) {
		// used to intercept serial output and store it in a string
		gb.Bus.ReserveAddress(types.SB, func(v byte) byte {
			*output += string(v)
			if strings.Contains(*output, "Passed") || strings.Contains(*output, "Failed") {
				gb.CPU.DebugBreakpoint = true
			}

			return 0
		})
	}
}

func AsModel(m types.Model) func(gb *GameBoy) {
	return func(gb *GameBoy) {
		gb.SetModel(m)
	}
}

func SerialConnection(gbFrom *GameBoy) Opt {
	return func(gbTo *GameBoy) {
		gbTo.Serial.Attach(gbFrom.Serial)
		gbFrom.Serial.Attach(gbTo.Serial)

		gbFrom.attachedGameBoy = gbTo
	}
}

func WithLogger(log log.Logger) Opt {
	return func(gb *GameBoy) {
		gb.Logger = log
	}
}

// WithBootROM sets the boot ROM for the emulator.
func WithBootROM(rom []byte) Opt {
	return func(gb *GameBoy) {
		gb.dontBoot = true // don't hle boot process

		romContents := make([]byte, 0x0900)
		gb.Bus.CopyFrom(0, 0x0900, romContents)
		gb.Bus.RegisterBootHandler(func() {
			gb.Bus.CopyTo(0, 0x0900, romContents)
		})

		gb.Bus.CopyTo(0, 0x0100, rom)
		if len(rom) == 0x900 {
			gb.Bus.CopyTo(0x0200, 0x0900, rom[0x200:])
		}

		gb.model = io.Which(rom)
	}
}

func WithPrinter(printer *accessories.Printer) Opt {
	return func(gb *GameBoy) {
		gb.Printer = printer
		gb.Serial.Attach(printer)
	}
}

func Speed(speed float64) Opt {
	return func(gb *GameBoy) {
		gb.speed = speed
	}
}
