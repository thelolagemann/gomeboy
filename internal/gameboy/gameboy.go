// Package gameboy provides an emulation of a Nintendo Game Boy.
//

package gameboy

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/serial"
	"github.com/thelolagemann/go-gameboy/internal/serial/accessories"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"
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
	types.NR10: 0x80,
	types.NR11: 0xBF,
	types.NR12: 0xF3,
	types.NR14: 0xBF,
	types.NR21: 0x3F,
	types.NR22: 0x00,
	types.NR24: 0xBF,
	types.NR30: 0x7F,
	types.NR31: 0xFF,
	types.NR32: 0x9F,
	types.NR33: 0xBF,
	types.NR41: 0xFF,
	types.NR42: 0x00,
	types.NR43: 0x00,
	types.NR50: 0x77,
	types.NR51: 0xF3,
	types.NR52: 0xF1,
	types.LCDC: 0x91,
	types.STAT: 0x80,
	types.BGP:  0xFC,
	types.BDIS: 0x01,
}

// GameBoy represents a Game Boy. It contains all the components of the Game Boy.
// It is the main entry point for the emulator.
type GameBoy struct {
	sync.RWMutex
	CPU   *cpu.CPU
	MMU   *mmu.MMU
	PPU   *ppu.PPU
	model types.Model

	APU        *apu.APU
	Joypad     *joypad.State
	Interrupts *interrupts.Service
	Timer      *timer.Controller
	Serial     *serial.Controller

	log.Logger

	currentCycle uint

	paused          bool
	frames          int
	ticks           uint16
	previousFrame   [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
	frameQueue      bool
	attachedGameBoy *GameBoy
	speed           float64
	Printer         *accessories.Printer
}

func (g *GameBoy) StartLinked(
	frames1 chan<- []byte,
	events1 chan<- display.Event,
	pressed1 <-chan joypad.Button,
	released1 <-chan joypad.Button,
	frames2 chan<- []byte,
	events2 chan<- display.Event,
	pressed2 <-chan joypad.Button,
	released2 <-chan joypad.Button,
) {

	// setup the frame buffer
	frameBuffer1 := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*3)
	frameBuffer2 := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*3)
	frame1 := [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8{}
	frame2 := [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8{}

	// start a ticker
	ticker := time.NewTicker(FrameTime)

	for {
		select {
		case p := <-pressed1:
			g.Joypad.Press(p)
		case r := <-released1:
			g.Joypad.Release(r)
		case p := <-pressed2:
			g.attachedGameBoy.Joypad.Press(p)
		case r := <-released2:
			g.attachedGameBoy.Joypad.Release(r)
		case <-ticker.C:
			// lock the gameboy
			g.Lock()
			// update the fps counter
			g.frames++

			// render frame
			if !g.paused && !g.CPU.Paused {
				frame1, frame2 = g.LinkFrame()
			}

			// turn frame into image
			for y := 0; y < ppu.ScreenHeight; y++ {
				for x := 0; x < ppu.ScreenWidth; x++ {
					frameBuffer1[(y*ppu.ScreenWidth+x)*3] = frame1[y][x][0]
					frameBuffer1[(y*ppu.ScreenWidth+x)*3+1] = frame1[y][x][1]
					frameBuffer1[(y*ppu.ScreenWidth+x)*3+2] = frame1[y][x][2]

					frameBuffer2[(y*ppu.ScreenWidth+x)*3] = frame2[y][x][0]
					frameBuffer2[(y*ppu.ScreenWidth+x)*3+1] = frame2[y][x][1]
					frameBuffer2[(y*ppu.ScreenWidth+x)*3+2] = frame2[y][x][2]
				}
			}

			// TODO reimplment this
			/*if time.Since(start) > time.Second {
				// average frame time
				avgFrameTime := avgTime(frameTimes)
				avgRenderTime := avgTime(renderTimes)
				frameTimes = frameTimes[:0]
				renderTimes = renderTimes[:0]

				// append to avg render times
				avgRenderTimes = append(avgRenderTimes, avgRenderTime)
				total := avgFrameTime + avgRenderTime

				totalAvgRenderTime := avgTime(avgRenderTimes)

				events <- display.Event{Type: display.EventTypeTitle, Data: fmt.Sprintf("Render: %s (AVG:%s) + Frame: %v | FPS: (%v:%s)", avgRenderTime.String(), totalAvgRenderTime.String(), avgFrameTime.String(), g.frames, total.String())}
				g.frames = 0
				start = time.Now()

				// make sure avg render times doesn't get too big
				if len(avgRenderTimes) > 144 {
					avgRenderTimes = avgRenderTimes[1:]
				}
			}*/

			// send frame events
			events1 <- display.Event{Type: display.EventTypeFrame, State: struct{ CPU display.CPUState }{CPU: struct {
				Registers struct {
					AF uint16
					BC uint16
					DE uint16
					HL uint16
					SP uint16
					PC uint16
				}
			}{Registers: struct {
				AF uint16
				BC uint16
				DE uint16
				HL uint16
				SP uint16
				PC uint16
			}{AF: g.CPU.AF.Uint16(), BC: g.CPU.Registers.BC.Uint16(), DE: g.CPU.Registers.DE.Uint16(), HL: g.CPU.Registers.HL.Uint16(), SP: g.CPU.SP, PC: g.CPU.PC}}}}
			events2 <- display.Event{Type: display.EventTypeFrame}

			// send frames
			frames1 <- frameBuffer1
			frames2 <- frameBuffer2

			// unlock the gameboy
			g.Unlock()
		}
	}
}

func (g *GameBoy) Start(frames chan<- []byte, events chan<- display.Event, pressed <-chan joypad.Button, released <-chan joypad.Button) {
	// setup fps counter
	g.frames = 0
	start := time.Now()
	frameStart := time.Now()
	frameTimes := make([]time.Duration, 0, FrameRate)
	renderTimes := make([]time.Duration, 0, FrameRate)
	g.APU.Play()

	// set initial image
	avgRenderTimes := make([]time.Duration, 0, FrameRate)

	// start a ticker
	ticker := time.NewTicker(FrameTime / time.Duration(g.speed))
	frameBuffer := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*3)
	var frame [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8

	for {
		select {
		case p := <-pressed:
			g.Joypad.Press(p)
		case r := <-released:
			g.Joypad.Release(r)
		case <-ticker.C:
			// lock the gameboy
			g.Lock()
			// update the fps counter
			g.frames++

			if !g.paused && !g.CPU.Paused {
				// render frame
				frameStart = time.Now()

				frame = g.Frame()
				renderTimes = append(renderTimes, time.Since(frameStart))

			} else {
				continue
			}
			frameStart = time.Now()

			// turn frame into image
			for y := 0; y < ppu.ScreenHeight; y++ {
				for x := 0; x < ppu.ScreenWidth; x++ {
					frameBuffer[(y*ppu.ScreenWidth+x)*3] = frame[y][x][0]
					frameBuffer[(y*ppu.ScreenWidth+x)*3+1] = frame[y][x][1]
					frameBuffer[(y*ppu.ScreenWidth+x)*3+2] = frame[y][x][2]
				}
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

				events <- display.Event{Type: display.EventTypeTitle, Data: fmt.Sprintf("Render: %s (AVG:%s) + Frame: %v | FPS: (%v:%s)", avgRenderTime.String(), totalAvgRenderTime.String(), avgFrameTime.String(), g.frames, total.String())}
				g.frames = 0
				start = time.Now()

				// make sure avg render times doesn't get too big
				if len(avgRenderTimes) > 144 {
					avgRenderTimes = avgRenderTimes[1:]
				}
			}

			// send frame events
			events <- display.Event{Type: display.EventTypeFrame, State: struct{ CPU display.CPUState }{CPU: struct {
				Registers struct {
					AF uint16
					BC uint16
					DE uint16
					HL uint16
					SP uint16
					PC uint16
				}
			}{Registers: struct {
				AF uint16
				BC uint16
				DE uint16
				HL uint16
				SP uint16
				PC uint16
			}{AF: g.CPU.AF.Uint16(), BC: g.CPU.Registers.BC.Uint16(), DE: g.CPU.Registers.DE.Uint16(), HL: g.CPU.Registers.HL.Uint16(), SP: g.CPU.SP, PC: g.CPU.PC}}}}

			// send frame
			frames <- frameBuffer

			// update frame times
			frameTimes = append(frameTimes, time.Since(frameStart))

			// check printer for queued data
			if g.Printer != nil && g.Printer.HasPrintJob() {
				events <- display.Event{Type: display.EventTypePrint, Data: g.Printer.GetPrintJob()}
			}

			// unlock the gameboy
			g.Unlock()
		}
	}

}

func (g *GameBoy) Pause() {
	g.paused = true
}

func (g *GameBoy) Unpause() {
	g.paused = false
}

func (g *GameBoy) TogglePause() {
	g.paused = !g.paused
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

func AsModel(m types.Model) func(gb *GameBoy) {
	return func(gb *GameBoy) {
		gb.SetModel(m)
		gb.initializeCPU()
	}
}

func SerialConnection(gbFrom *GameBoy) GameBoyOpt {
	return func(gbTo *GameBoy) {
		gbTo.Serial.Attach(gbFrom.Serial)
		gbFrom.Serial.Attach(gbTo.Serial)

		gbFrom.attachedGameBoy = gbTo
	}
}

func WithLogger(log log.Logger) GameBoyOpt {
	return func(gb *GameBoy) {
		gb.Logger = log
		gb.MMU.Log = log
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

func WithPrinter(printer *accessories.Printer) GameBoyOpt {
	return func(gb *GameBoy) {
		gb.Printer = printer
		gb.Serial.Attach(printer)
	}
}

func Speed(speed float64) GameBoyOpt {
	return func(gb *GameBoy) {
		gb.speed = speed
	}
}

// NewGameBoy returns a new GameBoy.
func NewGameBoy(rom []byte, opts ...GameBoyOpt) *GameBoy {
	cart := cartridge.NewCartridge(rom)
	interrupt := interrupts.NewService()
	pad := joypad.New(interrupt)
	serialCtl := serial.NewController(interrupt)
	timerCtl := timer.NewController(interrupt)
	sound := apu.NewAPU()
	memBus := mmu.NewMMU(cart, sound)
	video := ppu.New(memBus, interrupt)
	memBus.AttachVideo(video)

	g := &GameBoy{
		CPU: cpu.NewCPU(memBus, interrupt, video.DMA, timerCtl, video, sound, serialCtl),
		MMU: memBus,
		PPU: video,

		APU:        sound,
		Joypad:     pad,
		Interrupts: interrupt,
		Timer:      timerCtl,
		Serial:     serialCtl,
		model:      types.Unset, // default to DMGABC
		speed:      1.0,
	}

	// apply options
	for _, opt := range opts {
		opt(g)
	}

	// setup memory bus
	memBus.Map()

	// set model
	if g.model == types.Unset {
		if memBus.IsGBC() || memBus.IsGBCCompat() {
			g.model = types.CGBABC
		} else {
			g.model = types.DMGABC
		}
	}

	// setup starting register values
	if g.MMU.BootROM == nil {
		// TODO switch to using model to determine starting register values
		for addr, val := range startingRegisterValues {
			g.MMU.Write(addr, val)
		}
		g.PPU.Status.Mode = 3

		g.initializeCPU()
	}
	if g.MMU.IsGBCCompat() {
		video.LoadCompatibilityPalette()
	}

	video.StartRendering()

	return g
}

func (g *GameBoy) initializeCPU() {
	// setup initial cpu state
	g.CPU.PC = 0x100
	g.CPU.SP = 0xFFFE

	// set CPU registers from model
	registers := g.model.Registers()
	for i, val := range []*uint8{&g.CPU.A, &g.CPU.F, &g.CPU.B, &g.CPU.C, &g.CPU.D, &g.CPU.E, &g.CPU.H, &g.CPU.L} {
		*val = registers[i]
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

// LinkFrame will step the emulation until the PPU has finished
// rendering the current frame. It will then prepare the frame
// for display, and return it.
func (g *GameBoy) LinkFrame() ([ppu.ScreenHeight][ppu.ScreenWidth][3]uint8, [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8) {
	// step until the first GameBoy has finished rendering a frame
	for !g.PPU.HasFrame() {
		g.CPU.Step()
		g.attachedGameBoy.CPU.Step()
	}

	// step until the second GameBoy has finished rendering a frame
	for !g.attachedGameBoy.PPU.HasFrame() {
		g.CPU.Step()
		g.attachedGameBoy.CPU.Step()
	}

	// clear the refresh flags
	g.PPU.ClearRefresh()
	g.attachedGameBoy.PPU.ClearRefresh()

	// return the prepared frames
	return g.PPU.PreparedFrame, g.attachedGameBoy.PPU.PreparedFrame
}

// Frame will step the emulation until the PPU has finished
// rendering the current frame. It will then prepare the frame
// for display, and return it.
func (g *GameBoy) Frame() [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8 {
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
			var img1, img2 *image.RGBA
			g.PPU.DumpTileMaps(img1, img2)

			f, err := os.Create("tilemap.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			if err := png.Encode(f, img1); err != nil {
				panic(err)
			}

			img1 = g.PPU.DumpTiledata().(*image.RGBA)

			f, err = os.Create("tiledata.png")
			if err != nil {
				panic(err)
			}
			defer f.Close()

			if err := png.Encode(f, img1); err != nil {
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
			g.PPU.SaveCGBPalettes()
		},
		15: func() {
			g.PPU.SaveCompatibilityPalette()
		},
	}
}

func (g *GameBoy) SetModel(m types.Model) {
	// re-initialize MMU
	g.MMU.SetModel(m)
	g.model = m
}
