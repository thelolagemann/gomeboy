package gameboy

import (
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
		gb.b.ReserveAddress(types.SB, func(v byte) byte {
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

func WithState(b []byte) Opt {
	return func(gb *GameBoy) {
		// get state from bytes
		state := types.StateFromBytes(b)
		gb.Load(state)
		gb.loadedFromState = true
	}
}

// WithBootROM sets the boot ROM for the emulator.
func WithBootROM(rom []byte) Opt {
	return func(gb *GameBoy) {
		// if we have a boot ROM, we need to reset the CPU
		// otherwise the emulator will start at 0x100 with
		// the registers set to the values upon completion
		// of the boot ROM
		gb.CPU.PC = 0x0000
		gb.CPU.SP = 0x0000
		gb.CPU.A = 0x00
		gb.CPU.F = 0x00
		gb.CPU.B = 0x00
		gb.CPU.C = 0x00
		gb.CPU.D = 0x00
		gb.CPU.E = 0x00
		gb.CPU.H = 0x00
		gb.CPU.L = 0x0

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
