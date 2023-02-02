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
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"image/png"
	"os"
	"time"
)

const (
	// ClockSpeed is the clock speed of the Game Boy.
	ClockSpeed = 4194304 // 4.194304 MHz
	// FrameRate is the frame rate of the emulator.
	FrameRate = 60
	// FrameTime is the time it should take to render a frame.
	FrameTime     = time.Second / FrameRate
	TicksPerFrame = ClockSpeed / FrameRate
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

	log.Logger

	currentCycle uint

	paused        bool
	frames        int
	ticks         uint16
	previousFrame [ppu.ScreenWidth][ppu.ScreenHeight][3]uint8
	frameQueue    bool
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
	memBus := mmu.NewMMU(cart, sound)
	video := ppu.New(memBus, interrupt)
	memBus.AttachVideo(video)

	g := &GameBoy{
		CPU: cpu.NewCPU(memBus, interrupt, video.DMA, timerCtl, video, sound),
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
	fmt.Println("Starting emulation")
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

		if !g.paused {
			// render frame
			g.ProcessInputs(mon.PollKeys())
			frameStart = time.Now()

			frame := g.Frame()
			renderTimes = append(renderTimes, time.Since(frameStart))
			frameStart = time.Now()

			mon.Render(frame)

			// update frametime
			frameTimes = append(frameTimes, time.Since(frameStart))
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
	var avg time.Duration
	for _, d := range t {
		avg += d
	}
	return avg / time.Duration(len(t))
}

// Frame will step the emulation until the PPU has finished
// rendering the current frame. It will then prepare the frame
// for display, and return it.
func (g *GameBoy) Frame() [ppu.ScreenWidth][ppu.ScreenHeight][3]uint8 {
	// was the last frame rendered? (by the PPU)
	if g.frameQueue {
		// if so, tick until the next frame is ready
		for !g.ppu.HasFrame() {
			g.CPU.Step()
		}

		// prepare the frame for display
		g.ppu.PrepareFrame()
		g.ppu.ClearRefresh()

		// return the frame and reset the frame queue
		g.frameQueue = false
		g.previousFrame = g.ppu.PreparedFrame
		return g.previousFrame
	}
	ticks := uint32(0)
	// step until the next frame or until tick threshold is reached
	for ticks <= TicksPerFrame {
		ticks += uint32(g.CPU.Step())
	}

	// did the PPU render a frame?
	if g.ppu.HasFrame() {
		g.ppu.PrepareFrame()
		g.ppu.ClearRefresh()
		g.previousFrame = g.ppu.PreparedFrame
		return g.ppu.PreparedFrame
	} else {
		// if not, create a smoothed frame from the last frame
		// and the current frame (which is not yet finished)
		var smoothedFrame [ppu.ScreenWidth][ppu.ScreenHeight][3]uint8
		for x := uint8(0); x < ppu.ScreenWidth; x++ {
			for y := uint8(0); y < ppu.ScreenHeight; y++ {
				// is the pixel on the current frame black?

				// interpolate the current frame
				for c := 0; c < 3; c++ {
					// smooth by averaging the current and previous frame
					smoothedFrame[x][y][c] = uint8((uint16(g.previousFrame[x][y][c]) + uint16(g.ppu.PreparedFrame[x][y][c])) / 2)
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
			img := g.ppu.DumpTileMap()

			f, err := os.Create("tilemap.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if err := png.Encode(f, img); err != nil {
				panic(err)
			}

			img = g.ppu.DumpTiledata()

			f, err = os.Create("tiledata.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()

			if err := png.Encode(f, img); err != nil {
				panic(err)
			}

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
