// Package gameboy provides an emulation of a Nintendo Game Boy.
//

package gameboy

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/internal/display"
	"github.com/thelolagemann/go-gameboy/internal/io"
	"github.com/thelolagemann/go-gameboy/internal/io/timer"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
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

	Joypad     *joypad.State
	Interrupts *io.Interrupts
	Timer      *timer.Controller
	Serial     *io.Serial

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
	interrupt := io.NewInterrupts()
	pad := joypad.New(interrupt)
	serial := io.NewSerial()
	timerCtl := timer.NewController(interrupt)
	memBus := mmu.NewMMU(cart, pad, serial, timerCtl, interrupt, ram.NewRAM(0x2000))
	video := ppu.New(memBus, interrupt)
	memBus.AttachVideo(video)

	g := &GameBoy{
		CPU: cpu.NewCPU(memBus, interrupt),
		MMU: memBus,
		ppu: video,

		Joypad:     pad,
		Interrupts: interrupt,
		Timer:      timerCtl,
		Serial:     serial,
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
			g.ProcessInputs(inputs)

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

var keyHandlers = map[uint8]func(){
	8: func() {
		palette.Current++
		if palette.Current > 3 {
			palette.Current = 0
		}
	},
}

// ProcessInputs processes the inputs.
func (g *GameBoy) ProcessInputs(inputs display.Inputs) {
	for _, key := range inputs.Pressed {
		// check if it's a gameboy key
		if key > joypad.ButtonDown {
			keyHandlers[key]()
		} else {
			g.Joypad.Press(key)
		}
	}
	for _, key := range inputs.Released {
		if key <= joypad.ButtonDown {
			g.Joypad.Release(key)
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
		g.Timer.Step(cyclesCPU)
		g.DoInterrupts()
	}

	// TODO handle save
}

// DoInterrupts handles all the interrupts.
func (g *GameBoy) DoInterrupts() {
	if g.Interrupts.IME {
		if g.Interrupts.IF > 0 {
			for i := uint8(0); i < 5; i++ {
				if utils.Test(g.Interrupts.IF, i) && utils.Test(g.Interrupts.IE, i) {
					g.serviceInterrupt(i)
				}
			}
		}
	}
}

// serviceInterrupt handles the given interrupt.
func (g *GameBoy) serviceInterrupt(interrupt uint8) {
	g.Interrupts.IME = false
	g.Interrupts.IF = utils.Reset(g.Interrupts.IF, interrupt)

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
