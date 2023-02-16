// Package gameboy provides an emulation of a Nintendo Game Boy.
//

package gameboy

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/io"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"image/png"
	"os"
	"strings"
	"time"
)

var (
	// ClockSpeed is the clock speed of the Game Boy.
	ClockSpeed = 4194304 // 4.194304 MHz
	// FrameRate is the frame rate of the emulator.
	FrameRate = 60
	// FrameTime is the time it should take to render a frame.
	FrameTime            = time.Second / time.Duration(FrameRate)
	TicksPerFrame uint32 = uint32(ClockSpeed / FrameRate)
)

var startingRegisterValues = map[types.HardwareAddress]uint8{
	types.LCDC: 0x91,
	types.BDIS: 0x01,
}

type Model = uint8

const (
	ModelAutomatic Model = iota
	ModelDMG
	ModelCGB
)

// GameBoy represents a Game Boy. It contains all the components of the Game Boy.
// It is the main entry point for the emulator.
type GameBoy struct {
	CPU *cpu.CPU
	MMU *mmu.MMU
	PPU *ppu.PPU

	APU        *apu.APU
	Joypad     *joypad.State
	Interrupts *interrupts.Service
	Timer      *timer.Controller
	Serial     *io.Serial

	log.Logger

	currentCycle uint

	paused        bool
	frames        int
	ticks         uint16
	previousFrame [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
	frameQueue    bool
}

type GameBoyOpt func(gb *GameBoy)

func Debug() GameBoyOpt {
	return func(gb *GameBoy) {
		gb.CPU.Debug = true
	}
}

func SaveEvery(t time.Duration) GameBoyOpt {
	return func(gb *GameBoy) {
		if _, ok := gb.MMU.Cart.MemoryBankController.(cartridge.RAMController); ok {
			t := time.NewTicker(t)
			go func() {
				for range t.C {
					gb.MMU.Cart.Save()
				}
			}()
		}
	}
}

func SerialDebugger(output *string) GameBoyOpt {
	return func(gb *GameBoy) {
		// used to intercept serial output and store it in a string
		types.RegisterHardware(types.SB, func(v uint8) {
			*output += string(v)
			fmt.Println(*output)
			if strings.Contains(*output, "Passed") || strings.Contains(*output, "Failed") {
				gb.CPU.DebugBreakpoint = true
			}
		}, func() uint8 {
			return 0
		})
	}
}

func AsModel(m Model) func(gb *GameBoy) {
	return func(gb *GameBoy) {
		gb.SetModel(m)
		gb.initializeCPU()
	}
}

// WithBootROM sets the boot ROM for the emulator.
func WithBootROM(rom []byte) GameBoyOpt {
	return func(gb *GameBoy) {
		gb.MMU.SetBootROM(rom)

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
		gb.CPU.L = 0x00
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
	memBus := mmu.NewMMU(cart, sound)
	video := ppu.New(memBus, interrupt)
	memBus.AttachVideo(video)

	g := &GameBoy{
		CPU: cpu.NewCPU(memBus, interrupt, video.DMA, timerCtl, video, sound),
		MMU: memBus,
		PPU: video,

		APU:        sound,
		Joypad:     pad,
		Interrupts: interrupt,
		Timer:      timerCtl,
		Serial:     serial,
	}
	g.initializeCPU()

	for _, opt := range opts {
		opt(g)
	}

	memBus.Map()

	// setup starting register values
	if g.MMU.BootROM == nil {
		for addr, val := range startingRegisterValues {
			g.MMU.Write(addr, val)
		}
	}

	video.StartRendering()
	fmt.Printf("%02x\n", g.MMU.Cart.Header().TitleChecksum())
	return g
}

func (g *GameBoy) initializeCPU() {
	// setup initial cpu state
	g.CPU.PC = 0x100
	g.CPU.SP = 0xFFFE
	if g.MMU.IsGBCCompat() {
		g.CPU.A = 0x11
		g.CPU.F = 0x80
		g.CPU.B = 0x00
		g.CPU.C = 0x00
		g.CPU.D = 0xFF
		g.CPU.E = 0x56
		g.CPU.H = 0x00
		g.CPU.L = 0x0D
	} else {
		g.CPU.A = 0x01
		g.CPU.F = 0xB0
		g.CPU.B = 0x00
		g.CPU.C = 0x13
		g.CPU.D = 0x00
		g.CPU.E = 0xD8
		g.CPU.H = 0x01
		g.CPU.L = 0x4D
	}

}

// Start starts the Game Boy emulation. It will run until the game is closed.
func (g *GameBoy) Start(mon *display.Display) {
	// setup fps counter
	g.frames = 0
	start := time.Now()
	frameStart := time.Now()
	frameTimes := make([]time.Duration, 0, FrameRate)
	renderTimes := make([]time.Duration, 0, FrameRate)
	g.APU.Play()

	avgRenderTimes := make([]time.Duration, 0, FrameRate)

	// create a ticker to update the display
	ticker := time.NewTicker(FrameTime)

	for !mon.IsClosed() {
		g.frames++

		g.ProcessInputs(mon.PollKeys())
		if !g.paused {
			// render frame
			frameStart = time.Now()

			frame := g.Frame()
			renderTimes = append(renderTimes, time.Since(frameStart))
			frameStart = time.Now()

			mon.Render(frame)

			// update frametime
			frameTimes = append(frameTimes, time.Since(frameStart))
		} else {
			// render last frame
			mon.Render(g.previousFrame)
		}
		if time.Since(start) > time.Second {
			// average frame time
			avgFrameTime := avgTime(frameTimes)
			avgRenderTime := avgTime(renderTimes)
			frameTimes = frameTimes[:0]
			renderTimes = renderTimes[:0]

			// append to avg render times
			avgRenderTimes = append(avgRenderTimes, avgRenderTime)
			total := avgFrameTime + avgRenderTime

			totalAvgRenderTime := avgTime(avgRenderTimes)

			title := fmt.Sprintf("Render: %s (AVG:%s) + Frame: %v | FPS: (%v:%s)", avgRenderTime.String(), totalAvgRenderTime.String(), avgFrameTime.String(), g.frames, total.String())
			mon.SetTitle(title)

			g.frames = 0
			start = time.Now()

			// make sure avg render times doesn't get too big
			if len(avgRenderTimes) > 144 {
				avgRenderTimes = avgRenderTimes[1:]
			}
		}

		// wait for tick
		<-ticker.C
	}
}

func avgTime(t []time.Duration) time.Duration {
	if len(t) == 0 {
		return 0
	}
	var avg time.Duration
	for _, d := range t {
		avg += d
	}
	return avg / time.Duration(len(t))
}

// Frame will step the emulation until the PPU has finished
// rendering the current frame. It will then prepare the frame
// for display, and return it.
func (g *GameBoy) Frame() [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8 {
	// was the last frame rendered? (by the PPU)
	/*if g.frameQueue {
		// if so, tick until the next frame is ready
		for !g.PPU.HasFrame() {
			g.CPU.Step()
		}

		// prepare the frame for display
		g.PPU.ClearRefresh()

		// return the frame and reset the frame queue
		g.frameQueue = false
		g.previousFrame = g.PPU.PreparedFrame
		return g.previousFrame
	}*/
	ticks := uint32(0)
	// step until the next frame or until tick threshold is reached
	for ticks <= TicksPerFrame {
		ticks += uint32(g.CPU.Step())
	}

	// did the PPU render a frame?
	if g.PPU.HasFrame() {
		g.PPU.ClearRefresh()
		g.previousFrame = g.PPU.PreparedFrame
		return g.PPU.PreparedFrame
	} else {
		// if not, create a smoothed frame from the last frame
		// and the current frame (which is not yet finished)
		var smoothedFrame [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
		// TODO find a way to make this parallel (maybe use a channel of chunks?)
		for x := uint8(0); x < ppu.ScreenWidth; x++ {
			for y := uint8(0); y < ppu.ScreenHeight; y++ {
				// is the pixel on the current frame black?

				// interpolate the current frame
				for c := 0; c < 3; c++ {
					// smooth by averaging the current and previous frame
					smoothedFrame[y][x][c] = uint8((uint16(g.previousFrame[y][x][c]) + uint16(g.PPU.PreparedFrame[y][x][c])) / 2)
				}
			}
		}
		// flag that the frame is not finished
		g.frameQueue = true
		return smoothedFrame
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
		10: func() {
			img := g.PPU.DumpTileMap()

			f, err := os.Create("tilemap.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if err := png.Encode(f, img); err != nil {
				panic(err)
			}

			img = g.PPU.DumpTiledata()

			f, err = os.Create("tiledata.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()

			if err := png.Encode(f, img); err != nil {
				panic(err)
			}

		},
		11: func() {
			g.PPU.Debug.BackgroundDisabled = !g.PPU.Debug.BackgroundDisabled
		},
		12: func() {
			g.PPU.Debug.WindowDisabled = !g.PPU.Debug.WindowDisabled
		},
		13: func() {
			g.PPU.Debug.SpritesDisabled = !g.PPU.Debug.SpritesDisabled
		},
		14: func() {
			types.SavePaletteDump()
		},
		15: func() {
			g.PPU.SaveCompatibilityPalette()
		},
	}
}

// ProcessInputs processes the inputs.
func (g *GameBoy) ProcessInputs(inputs display.Inputs) {
	for _, key := range inputs.Pressed {
		// check if it's a gameboy key
		if key <= joypad.ButtonDown {
			g.Joypad.Press(key)
		} else {
			// check if it's a debug key
			if handler, ok := g.keyHandlers()[key]; ok {
				handler()
			}
		}
	}
	for _, key := range inputs.Released {
		if key <= joypad.ButtonDown {
			g.Joypad.Release(key)
		}
	}
}

func (g *GameBoy) SetModel(m Model) {
	// re-initialize MMU
	g.MMU.SetModel(m)
	// restart PPU rendering

	// re-initialize CPU
}
