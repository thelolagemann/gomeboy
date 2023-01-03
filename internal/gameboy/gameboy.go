// Package gameboy provides an emulation of a Nintendo Game Boy.
//

package gameboy

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/internal/display"
	"github.com/thelolagemann/go-gameboy/internal/io"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/pkg/bits"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"time"

	io2 "io"
)

const (
	CyclesPerFrame = 70224
)

// GameBoy represents a Game Boy. It contains all the components of the Game Boy.
// It is the main entry point for the emulator.
type GameBoy struct {
	CPU *cpu.CPU
	MMU *mmu.MMU
	ppu *ppu.PPU

	LastSave time.Time

	log.Logger

	currentCycle uint
	w            io2.Writer
}

type GameBoyOpt func(gb *GameBoy)

// NoBios disables the BIOS by setting CPU.PC to 0x100.
func NoBios() GameBoyOpt {
	return func(gb *GameBoy) {
		gb.CPU.PC = 0x0100
	}
}

// DisableROM disables the ROM
func DisableROM() GameBoyOpt {
	return func(gb *GameBoy) {
		gb.MMU.LoadCartridge([]byte{})
	}
}

// NewGameBoy returns a new GameBoy.
func NewGameBoy(rom []byte, opts ...GameBoyOpt) *GameBoy {
	cart := cartridge.NewCartridge(rom)
	memBus := mmu.NewMMU(cart)
	video := ppu.New()
	ioBus := io.NewIO(video)
	memBus.SetBus(ioBus)
	ioBus.AttachMMU(memBus)
	video.AttachBus(memBus.Bus)
	g := &GameBoy{
		CPU: cpu.NewCPU(memBus),
		MMU: memBus,
		ppu: video,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Start starts the Game Boy emulation. It will run until the game is closed.
func (g *GameBoy) Start(mon *display.Display) {
	t := time.NewTicker(time.Second / 60)

	// setup fps counter
	frames := 0
	start := time.Now()
	defer t.Stop()
	for {
		select {
		case <-t.C:
			frames++

			inputs := mon.PollKeys()
			if reqInt := g.MMU.Bus.Input().ProcessInputs(inputs); reqInt {
				g.MMU.Bus.Interrupts().Request(io.InterruptJoypadFlag)
			}

			g.Update(CyclesPerFrame)
			mon.Render(g.ppu.PreparedFrame)

			if time.Since(start) > time.Second {
				title := fmt.Sprintf("GomeBoy %s (FPS: %2v)", g.MMU.Cart.Title(), frames)
				// fmt.Println(title)
				mon.SetTitle(title)

				frames = 0
				start = time.Now()
			}
		}
	}
}

// Update updates all the components of the Game Boy by the given number of cycles.
func (g *GameBoy) Update(cyclesPerFrame uint) {
	// TODO handle stopped
	// TODO handle io
	// TODO handle sound

	cycles := uint(0)
	for cycles < cyclesPerFrame {
		cyclesCPU := g.CPU.Step()
		cycles += uint(cyclesCPU)
		g.ppu.Step(uint16(cyclesCPU))
		if reqInt := g.MMU.Bus.Timer().Update(cyclesCPU); reqInt {
			g.MMU.Bus.Interrupts().Request(io.InterruptTimerFlag)
		}
		g.DoInterrupts()

	}

	// TODO handle save
}

// DoInterrupts handles all the interrupts.
func (g *GameBoy) DoInterrupts() {
	if g.MMU.Bus.Interrupts().IME {
		if g.MMU.Bus.Interrupts().IF > 0 {
			for i := uint8(0); i < 5; i++ {
				if bits.Test(g.MMU.Bus.Interrupts().IF, i) && bits.Test(g.MMU.Bus.Interrupts().IE, i) {
					g.serviceInterrupt(i)
				}
			}
		}
	}
}

// serviceInterrupt handles the given interrupt.
func (g *GameBoy) serviceInterrupt(interrupt uint8) {
	g.MMU.Bus.Interrupts().IME = false
	g.MMU.Bus.Interrupts().IF = bits.Reset(g.MMU.Bus.Interrupts().IF, interrupt)

	// save the current execution address by pushing it to the stack
	g.CPU.PushStack(g.CPU.PC)

	if interrupt != uint8(io.InterruptVBLFlag) {
		fmt.Println("Interrupt", interrupt)
	}
	// jump to the interrupt handler
	switch interrupt {
	case 0:
		g.CPU.PC = 0x0040
	case 1:
		g.CPU.PC = 0x0048
	case 2:
		g.CPU.PC = 0x0050
	case 3:
		g.CPU.PC = 0x0058
	case 4:
		g.CPU.PC = 0x0060
	default:
		panic("illegal interrupt")
	}
}
