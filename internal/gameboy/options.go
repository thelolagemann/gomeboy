package gameboy

import (
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/internal/types"
	"strings"
)

// Opt is a function that modifies the emulator.
type Opt func(gb *GameBoy)

// Debug enables debug breakpoints. This causes the emulator to halt
// execution under certain conditions, such as running the ld b, b instruction.
func Debug() Opt { return func(gb *GameBoy) { gb.CPU.Debug = true } }

// SerialDebugger overrides the default types.SB handler to intercept ASCII
// characters being written to the serial port.
func SerialDebugger(output *string) Opt {
	return func(gb *GameBoy) {
		gb.Bus.ReserveAddress(types.SB, func(v byte) byte {
			*output += string(v)
			if strings.Contains(*output, "Passed") || strings.Contains(*output, "Failed") {
				gb.CPU.DebugBreakpoint = true
			}

			return 0
		})
	}
}

// AsModel overrides the model inferred from the cartridge with the provided model.
func AsModel(m types.Model) Opt { return func(gb *GameBoy) { gb.model = m } }

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

		gb.model = types.Which(rom)
	}
}

// WithPrinter creates a new accessories.Printer and attaches it to the emulator.
func WithPrinter() Opt { return func(gb *GameBoy) { gb.Serial.Attach(accessories.NewPrinter()) } }
