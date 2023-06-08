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
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/serial"
	"github.com/thelolagemann/go-gameboy/internal/serial/accessories"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"github.com/thelolagemann/go-gameboy/pkg/emu"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	maxArraySize = 1 << 30                                // 1 GB
	frameSize    = ppu.ScreenWidth * ppu.ScreenHeight * 3 // 0.75 MB
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

// GameBoy represents a Game Boy. It contains all the components of the Game Boy.
// It is the main entry point for the emulator.
type GameBoy struct {
	sync.RWMutex
	CPU             *cpu.CPU
	Close           chan struct{}
	MMU             *mmu.MMU
	PPU             *ppu.PPU
	model           types.Model
	loadedFromState bool

	APU        *apu.APU
	Joypad     *joypad.State
	Interrupts *interrupts.Service
	Timer      *timer.Controller
	Serial     *serial.Controller

	log.Logger

	currentCycle uint

	paused, running bool
	frames          int
	previousFrame   [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
	frameQueue      bool
	attachedGameBoy *GameBoy
	speed           float64
	Printer         *accessories.Printer
	save            *emu.Save
	Scheduler       *scheduler.Scheduler

	Options []GameBoyOpt
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
		case <-g.Close:
			return
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
			if !g.paused {
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
	g.running = true
	// setup fps counter
	g.frames = 0
	start := time.Now()
	frameStart := time.Now()
	renderTimes := make([]time.Duration, 0, int(FrameRate))
	g.APU.Play()

	// check if the cartridge has a ram controller and start a ticker to save the ram
	var saveTicker *time.Ticker
	var ram cartridge.RAMController
	if r, ok := g.MMU.Cart.MemoryBankController.(cartridge.RAMController); ok && g.MMU.Cart.Header().RAMSize > 0 {

		// start a ticker
		saveTicker = time.NewTicker(time.Second * 3)
		ram = r

	} else {
		// create a fake ticker that never ticks
		saveTicker = &time.Ticker{
			C: make(chan time.Time),
		}
	}

	// set initial image
	avgRenderTimes := make([]time.Duration, 0, int(FrameRate))

	// start a ticker
	ticker := time.NewTicker(FrameTime / time.Duration(g.speed))
	frameBuffer := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*3)
	var frame [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8

	// Get a pointer to the first element of the frame array
	framePtr := unsafe.Pointer(&frame[0][0][0])

	// Get a pointer to the first element of the frameBuffer array
	frameBufferPtr := unsafe.Pointer(&frameBuffer[0])

	// update window title
	events <- display.Event{Type: display.EventTypeTitle, Data: fmt.Sprintf("GomeBoy (%s)", g.MMU.Cart.Header().Title)}

	// create a goroutine to handle the input
	go func() {
		for {
			select {
			case <-g.Close:
				return
			case p := <-pressed:
				g.Joypad.Press(p)
			case r := <-released:
				g.Joypad.Release(r)
			}
		}
	}()

emuLoop:
	for {
		select {
		case <-g.Close:
			// once the gameboy is closed, stop the ticker
			ticker.Stop()
			//g.Logger.Debugf("closing gameboy")

			// close the save file
			if g.save != nil {
				b := ram.SaveRAM()
				if err := g.save.SetBytes(b); err != nil {
					g.Logger.Errorf("error saving emu: %v", err)
				}
				if err := g.save.Close(); err != nil {
					g.Logger.Errorf("error closing save file: %v", err)
				}
			}
			g.MMU.PrintLoggedReads()
			g.CPU.LogUsedInstructions()
			break emuLoop
		case <-ticker.C:
			// lock the gameboy
			g.Lock()

			// update the fps counter
			g.frames++

			if !g.paused {
				// render frame
				frameStart = time.Now()

				frame = g.Frame()
				frameEnd := time.Now()
				renderTimes = append(renderTimes, frameEnd.Sub(frameStart))
				// copy the memory block from frame to frameBuffer
				copy((*[maxArraySize]byte)(frameBufferPtr)[:frameSize:frameSize], (*[maxArraySize]byte)(framePtr)[:frameSize:frameSize])

				if time.Since(start) > time.Second {
					// average frame time
					avgRenderTime := avgTime(renderTimes)
					renderTimes = renderTimes[:0]

					// append to avg render times
					avgRenderTimes = append(avgRenderTimes, avgRenderTime)

					//totalAvgRenderTime := avgTime(avgRenderTimes)

					//events <- display.Event{Type: display.EventTypeTitle, Data: fmt.Sprintf("GomeBoy: %s (AVG:%s) | FPS: %v", avgRenderTime.String(), totalAvgRenderTime.String(), g.frames)}
					events <- display.Event{Type: display.EventTypeFrameTime, Data: avgRenderTimes}
					g.frames = 0
					start = time.Now()

					// make sure avg render times doesn't get too big
					if len(avgRenderTimes) > 60 {
						avgRenderTimes = avgRenderTimes[1:]
					}

					// send sample data
					events <- display.Event{Type: display.EventTypeSample, Data: g.APU.Samples}
				}

				// send frame events
				events <- display.Event{Type: display.EventTypeFrame}

				// send frame
				frames <- frameBuffer

				// check printer for queued data
				if g.Printer != nil && g.Printer.HasPrintJob() {
					events <- display.Event{Type: display.EventTypePrint, Data: g.Printer.GetPrintJob()}
				}
			}

			// unlock the gameboy
			g.Unlock()
		case <-saveTicker.C:
			g.Lock()
			// get the data from the RAM
			data := ram.SaveRAM()
			// write the data to the save
			if err := g.save.SetBytes(data); err != nil {
				g.Logger.Errorf("error saving emu: %v", err)
				g.Unlock()
			}
			g.Unlock()
		}
	}

	g.running = false
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

func NoAudio() GameBoyOpt {
	return func(gb *GameBoy) {
		gb.APU.Pause()
	}
}

func SerialDebugger(output *string) GameBoyOpt {
	return func(gb *GameBoy) {
		// used to intercept serial output and store it in a string
		types.RegisterHardware(types.SB, func(v uint8) {
			*output += string(v)
			fmt.Println(*output)
			if strings.Contains(*output, "Passed") || strings.Contains(*output, "Failed") {
				fmt.Println("BREAKPOINT")
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

func WithState(b []byte) GameBoyOpt {
	return func(gb *GameBoy) {
		// get state from bytes
		state := types.StateFromBytes(b)
		gb.Load(state)
		gb.loadedFromState = true
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
	types.Lock.Lock()
	defer types.Lock.Unlock()

	sched := scheduler.NewScheduler()

	cart := cartridge.NewCartridge(rom)
	interrupt := interrupts.NewService()
	pad := joypad.New(interrupt)
	serialCtl := serial.NewController(interrupt, sched)
	sound := apu.NewAPU(sched)
	timerCtl := timer.NewController(interrupt, sched, sound)
	memBus := mmu.NewMMU(cart, sound)
	sound.AttachBus(memBus)
	video := ppu.New(memBus, interrupt, sched)
	memBus.AttachVideo(video)
	processor := cpu.NewCPU(memBus, interrupt, sched, video)
	video.AttachNotifyFrame(func() {
		processor.HasFrame()
	})
	g := &GameBoy{
		CPU:    processor,
		MMU:    memBus,
		PPU:    video,
		Logger: log.New(),

		APU:        sound,
		Joypad:     pad,
		Interrupts: interrupt,
		Timer:      timerCtl,
		Serial:     serialCtl,
		model:      types.Unset, // default to DMGABC
		speed:      1.0,
		Close:      make(chan struct{}, 2),
		Scheduler:  sched,
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
	video.StartRendering()

	sound.SetModel(g.model)
	processor.SetModel(g.model)

	// setup starting register values
	if g.MMU.BootROM == nil && !g.loadedFromState {
		g.initializeCPU()
		if g.MMU.IsGBCCompat() {
			video.LoadCompatibilityPalette()
		}
	}

	// does the cartridge have RAM? (and therefore a save file)
	if ram, ok := cart.MemoryBankController.(cartridge.RAMController); ok && cart.Header().RAMSize > 0 {
		// try to load the save file
		saveFiles, err := emu.LoadSaves(g.MMU.Cart.Title())

		if err != nil {
			// was there an error loading the save files?
			g.Logger.Errorf("error loading save files: %s", err)
		} else {
			// if there are no save files, create one
			if saveFiles == nil || len(saveFiles) == 0 {
				g.Logger.Debugf("no save file found for %s", g.MMU.Cart.Title())

				g.save, err = emu.NewSave(g.MMU.Cart.Title(), g.MMU.Cart.Header().RAMSize)
				if err != nil {
					g.Logger.Errorf("error creating save file: %s", err)
				} else {
					g.Logger.Debugf("created save file %s : (%dKiB)", g.save.Path, len(g.save.Bytes())/1024)
				}
			} else {
				// load the save file
				g.save = saveFiles[0]
				// g.Logger.Debugf("loading save file %s", g.save.Path)
			}
			ram.LoadRAM(g.save.Bytes())
		}
	}

	// try to load cheats using filename of rom

	g.Options = opts

	return g
}

func (g *GameBoy) initializeCPU() {
	// g.Logger.Debugf("initializing CPU with model %s", g.model)
	// setup initial cpu state
	g.CPU.PC = 0x100
	g.CPU.SP = 0xFFFE

	// set CPU registers from model
	registers := g.model.Registers()
	for i, val := range []*uint8{&g.CPU.A, &g.CPU.F, &g.CPU.B, &g.CPU.C, &g.CPU.D, &g.CPU.E, &g.CPU.H, &g.CPU.L} {
		*val = registers[i]
	}

	// set HW registers from model
	hwRegisters := g.model.IO()

	// sort map by key
	var keys []int
	for k := range hwRegisters {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	// set registers in order
	for _, k := range keys {
		g.MMU.Set(uint16(k), hwRegisters[uint16(k)])
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
	g.CPU.Frame()
	g.attachedGameBoy.CPU.Frame()

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
	g.CPU.Frame()
	g.PPU.RefreshScreen = false

	return g.PPU.PreparedFrame
}

func (g *GameBoy) SetModel(m types.Model) {
	// re-initialize MMU
	g.MMU.SetModel(m)
	g.model = m
}

var _ types.Stater = (*GameBoy)(nil)

func (g *GameBoy) Load(s *types.State) {
	g.CPU.Load(s)
	g.MMU.Load(s)
	g.PPU.Load(s)
	// g.APU.LoadRAM(s) TODO implement APU state
	g.Timer.Load(s)
	g.Joypad.Load(s)
	g.Serial.Load(s)
}

func (g *GameBoy) Save(s *types.State) {
	g.CPU.Save(s)
	g.MMU.Save(s)
	g.PPU.Save(s)
	// g.APU.SaveRAM(s) TODO implement APU state
	g.Timer.Save(s)
	g.Joypad.Save(s)
	g.Serial.Save(s)
}

func (g *GameBoy) IsRunning() bool {
	return g.running
}
