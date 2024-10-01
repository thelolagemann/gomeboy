package gameboy

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/apu"
	"github.com/thelolagemann/gomeboy/internal/cpu"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/serial"
	"github.com/thelolagemann/gomeboy/internal/timer"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"path/filepath"
	"strings"
)

// GameBoy represents the state and components of the Game Boy emulator.
type GameBoy struct {
	CPU       *cpu.CPU
	PPU       *ppu.PPU
	APU       *apu.APU
	Timer     *timer.Controller
	Serial    *serial.Controller
	Scheduler *scheduler.Scheduler
	Bus       *io.Bus

	save            *emulator.Save
	filename        string
	model           types.Model
	dontBoot        bool
	rumbling        bool
	paused, running bool
	initialised     bool

	ROM     []byte
	options []Opt
}

// NewGameBoy creates a new GameBoy with the provided Opt(s).
func NewGameBoy(opts ...Opt) *GameBoy { return &GameBoy{options: opts} }

// LoadROM loads a ROM file from the specified path and initializes the Game Boy.
//
// It accepts a string representing the absolute path to the ROM file.
// If loading fails, it will return an error.
//
// Example:
//
//	gb.LoadROM("path/to/rom.gb")
func (g *GameBoy) LoadROM(romPath string) error {
	var err error
	g.ROM, err = utils.LoadFile(romPath)
	if err != nil {
		return err
	}
	g.filename = strings.TrimSuffix(filepath.Base(romPath), filepath.Ext(romPath))
	g.Init()
	return nil
}

// Init initializes the Game Boy and its components, including CPU, PPU, APU, Timer, and Bus.
//
// It sets up the scheduler, maps memory, and configures the system according to the loaded ROM and model.
// This function also handles loading save files for battery-backed cartridges.
func (g *GameBoy) Init() {
	sched := scheduler.NewScheduler()

	b := io.NewBus(sched, g.ROM)
	serialCtl := serial.NewController(b, sched)
	sound := apu.New(b, sched)
	timerCtl := timer.NewController(b, sched, sound)
	video := ppu.New(b, sched)
	processor := cpu.NewCPU(b, sched)

	var model = types.DMGABC
	if b.Cartridge().IsCGBCartridge() {
		model = types.CGBABC
	}

	g.CPU = processor
	g.PPU = video
	g.Bus = b
	g.Serial = serialCtl
	g.Timer = timerCtl
	g.APU = sound
	g.Scheduler = sched
	g.model = model

	for _, o := range g.options {
		o(g)
	}

	// does the cartridge have battery backed RAM? (and therefore a save file)
	if b.Cartridge().Features.Battery {
		// try to load the save file
		var err error
		g.save, err = emulator.NewSave(g.filename, uint(b.Cartridge().RAMSize))

		if err != nil {
			// was there an error loading the save files?
			log.Errorf(fmt.Sprintf("error loading save files: %s", err))
		} else {
			copy(g.Bus.Cartridge().RAM, g.save.Bytes())
			var length = len(g.save.Bytes())
			if length > 0x2000 {
				length = 0x2000
			}
			g.Bus.CopyTo(0xA000, 0xC000, g.save.Bytes()[:length])
		}
	}
	g.Bus.Map(g.model)
	g.Colourise()
	if !g.dontBoot {
		g.CPU.Boot(g.model)
		g.Bus.Boot()
	}

	g.Bus.Cartridge().RumbleCallback = func(b bool) {
		g.rumbling = b
	}

	// schedule the frame sequencer event for the next 8192 ticks
	g.Scheduler.ScheduleEvent(scheduler.APUFrameSequencer, uint64(8192-g.Scheduler.SysClock()&0x0fff))
	g.Scheduler.ScheduleEvent(scheduler.APUFrameSequencer2, uint64(8192-g.Scheduler.SysClock()&0x0fff)+4096)
	g.initialised = true
}

// Colourise applies color palettes to the Game Boy's PPU based on the cartridge type and system model.
//
// If the loaded cartridge is not a Game Boy Color (CGB) cartridge and the Game Boy model is set to CGB, a color palette
// is selected based on the cartridge's licensee and title. If no specific palette is found, a default palette is used.
//
// For non-CGB models, a greyscale palette is applied to the PPU.
func (g *GameBoy) Colourise() {
	if !g.Bus.Cartridge().IsCGBCartridge() && (g.model == types.CGBABC || g.model == types.CGB0) {
		var pal = ppu.ColourisationPalettes[0]
		if g.Bus.Cartridge().Licensee() == "Nintendo" {
			// compute title hash
			hash := uint8(0)
			title := []byte(g.Bus.Cartridge().Title)
			for i := 0; i < len(title); i++ {
				hash += title[i]
			}
			var ok bool
			pal, ok = ppu.ColourisationPalettes[uint16(hash)]

			if !ok {
				pal, ok = ppu.ColourisationPalettes[uint16(title[3])<<8|uint16(hash)]
				if !ok {
					pal = ppu.ColourisationPalettes[0]
				}
			}
		}
		g.PPU.BGColourisationPalette = pal.BG
		g.PPU.OBJ0ColourisationPalette = pal.OBJ0
		g.PPU.OBJ1ColourisationPalette = pal.OBJ1
	} else {
		g.PPU.BGColourisationPalette = ppu.ColourPalettes[ppu.Greyscale]
		g.PPU.OBJ0ColourisationPalette = ppu.ColourPalettes[ppu.Greyscale]
		g.PPU.OBJ1ColourisationPalette = ppu.ColourPalettes[ppu.Greyscale]
	}
}

// Frame generates the next frame of the Game Boy's display and applies any visual effects.
func (g *GameBoy) Frame() [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8 {
	g.running = true
	g.CPU.Frame()

	if g.rumbling {
		// utils.ShakeFrame(&g.PPU.PreparedFrame, rand.N(100))
	}
	if g.Bus.Cartridge().Features.Accelerometer {
		// utils.Rotate2DFrame(&g.PPU.PreparedFrame, -float64(g.Bus.Cartridge().AccelerometerX), float64(g.Bus.Cartridge().AccelerometerY)) // TODO make configurable
	}
	g.running = false
	return g.PPU.PreparedFrame
}

// Save writes the current state of the Game Boy's RAM to the save file, if a save is present.
func (g *GameBoy) Save() error {
	// close the save file
	if g.save != nil {
		b := g.Bus.Cartridge().RAM
		if err := g.save.SetBytes(b); err != nil {
			return err
		}
		if err := g.save.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (g *GameBoy) Initialised() bool { return g.initialised } // has the Game Boy been initialised?
func (g *GameBoy) Paused() bool      { return g.paused }      // is the Game Boy paused?
func (g *GameBoy) Pause()            { g.paused = true }      // pause execution of the Game Boy
func (g *GameBoy) Resume()           { g.paused = false }     // resume execution of the Game Boy
func (g *GameBoy) Running() bool     { return g.running }     // is the emulator currently running?
