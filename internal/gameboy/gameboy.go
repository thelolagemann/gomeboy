// Package gameboy provides an emulation of a Nintendo Game Boy.
//

package gameboy

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/internal/display"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/io"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"time"

	io2 "io"
)

const (
	// ClockSpeed is the clock speed of the Game Boy.
	ClockSpeed = 4194304 // 4.194304 MHz
	// CyclesPerFrame is the number of clock cycles per frame.
	CyclesPerFrame = ClockSpeed / 60
)

// GameBoy represents a Game Boy. It contains all the components of the Game Boy.
// It is the main entry point for the emulator.
type GameBoy struct {
	CPU *cpu.CPU
	MMU *mmu.MMU
	ppu *ppu.PPU

	APU        *apu.APU
	Joypad     *joypad.State
	Interrupts *interrupts.Service
	Timer      *timer.Controller
	Serial     *io.Serial

	LastSave time.Time

	log.Logger

	currentCycle uint
	w            io2.Writer

	paused bool
}

type GameBoyOpt func(gb *GameBoy)

func Debug() GameBoyOpt {
	return func(gb *GameBoy) {
		gb.CPU.Debug = true
	}
}

// NoBios disables the BIOS by setting CPU.CPU.PC to 0x100.
func NoBios() GameBoyOpt {
	return func(gb *GameBoy) {
		gb.CPU.PC = 0x0100
	}
}

// NewGameBoy returns a new GameBoy.
func NewGameBoy(rom []byte, opts ...GameBoyOpt) *GameBoy {
	cart := cartridge.NewCartridge(rom)
	interrupt := interrupts.NewService()
	pad := joypad.New(interrupt)
	serial := io.NewSerial()
	timerCtl := timer.NewController(interrupt)
	sound := apu.NewAPU()
	memBus := mmu.NewMMU(cart, pad, serial, timerCtl, interrupt, sound)
	video := ppu.New(memBus, interrupt)
	memBus.AttachVideo(video)

	g := &GameBoy{
		CPU: cpu.NewCPU(memBus, interrupt),
		MMU: memBus,
		ppu: video,

		APU:        sound,
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
	fmt.Println("Starting emulation")
	// setup fps counter
	frames := 0
	start := time.Now()
	g.APU.Play()
	defer t.Stop()
	for !mon.IsClosed() {
		select {
		case <-t.C:
			frames++

			inputs := mon.PollKeys()
			g.ProcessInputs(inputs)

			g.Update(uint(float32(ClockSpeed) / 60.0 * g.CPU.Speed))
			mon.Render(g.ppu.PreparedFrame)

			if time.Since(start) > time.Second {
				title := fmt.Sprintf("%s | FPS: %v", g.MMU.Cart.Header().String(), frames)
				mon.SetTitle(title)

				frames = 0
				start = time.Now()
			}
		}
	}
}

func (g *GameBoy) keyHandlers() map[uint8]func() {
	return map[uint8]func(){
		8: func() {
			palette.CyclePalette()
		},
		9: func() {
			g.paused = !g.paused
			if g.paused {
				g.APU.Pause()
			} else {
				g.APU.Play()
			}
		},
	}
}

// ProcessInputs processes the inputs.
func (g *GameBoy) ProcessInputs(inputs display.Inputs) {
	for _, key := range inputs.Pressed {
		// check if it's a gameboy key
		if key > joypad.ButtonDown {
			g.keyHandlers()[key]()
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
	if g.paused {
		return
	}
	// TODO handle io

	cycles := uint(0)
	for cycles <= cyclesPerFrame {
		cyclesCPU := g.CPU.Step()
		cycles += cyclesCPU
		g.ppu.Step(uint16(cyclesCPU))
		g.Timer.Step(uint8(cyclesCPU))

		g.APU.Step(int(cyclesCPU), 1)
		cycles += g.DoInterrupts()
	}

	// TODO handle save
}

// DoInterrupts handles all the interrupts.
func (g *GameBoy) DoInterrupts() uint {
	// check if interrupts are enabling (EI is delayed by 1 instruction)
	if g.Interrupts.Enabling {
		g.Interrupts.Enabling = false
		g.Interrupts.IME = true
		return 0
	}

	if g.Interrupts.IME {
		for i := uint8(0); i < 5; i++ {
			if g.Interrupts.Enable&(1<<i) != 0 && g.Interrupts.Flag&(1<<i) != 0 {
				cycles := 0
				if g.CPU.Halted {
					cycles += 1
				}

				if g.serviceInterrupt(i) {
					cycles += 5
				}
				return uint(cycles)
			}
		}
	}

	return 0
}

// serviceInterrupt handles the given interrupt.
func (g *GameBoy) serviceInterrupt(interrupt uint8) bool {
	// if halted without IME enabled, just clear the halt flag
	if !g.Interrupts.IME && g.CPU.Halted {
		g.CPU.Halted = false
		return false
	}

	g.Interrupts.IME = false
	g.CPU.Halted = false
	g.Interrupts.Clear(interrupt)

	// save the current execution address by pushing it to the stack
	g.CPU.PushStack(g.CPU.PC)

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

	return true
}
