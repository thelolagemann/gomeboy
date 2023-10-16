// Package gameboy provides an emulation of a Nintendo Game Boy.
//

package gameboy

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/apu"
	"github.com/thelolagemann/gomeboy/internal/cartridge"
	"github.com/thelolagemann/gomeboy/internal/cpu"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/serial"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/internal/timer"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"math/rand"
	"sync"
	"time"
	"unsafe"
)

const (
	maxArraySize = 1 << 30                                // 1 GB
	frameSize    = ppu.ScreenWidth * ppu.ScreenHeight * 3 // 0.75 MB
)

var (
	// FrameRate is the frame rate of the emulator.
	FrameRate = 59.97
	// FrameTime is the time it should take to render a frame.
	FrameTime = time.Second / time.Duration(FrameRate)
)

// GameBoy represents a Game Boy. It contains all the components of the Game Boy.
// It is the main entry point for the emulator.
type GameBoy struct {
	sync.RWMutex
	CPU             *cpu.CPU
	PPU             *ppu.PPU
	model           types.Model
	loadedFromState bool

	cmdChannel chan emulator.CommandPacket

	APU       *apu.APU
	Cartridge *cartridge.Cartridge
	Joypad    *joypad.State
	Timer     *timer.Controller
	Serial    *serial.Controller

	b *io.Bus

	log.Logger

	paused, running bool
	frames          int
	previousFrame   [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
	attachedGameBoy *GameBoy
	speed           float64
	Printer         *accessories.Printer
	save            *emulator.Save
	Scheduler       *scheduler.Scheduler
}

func (g *GameBoy) State() emulator.State {
	if g.paused {
		return emulator.Paused
	}
	if !g.running {
		return emulator.Stopped
	}

	return emulator.Running
}

func (g *GameBoy) SendCommand(command emulator.CommandPacket) emulator.ResponsePacket {
	g.cmdChannel <- command
	return emulator.ResponsePacket{}
}

func (g *GameBoy) AttachAudioListener(player func([]byte)) {
	g.APU.AttachPlayback(player)
}

func (g *GameBoy) StartLinked(
	frames1 chan<- []byte,
	events1 chan<- event.Event,
	pressed1 <-chan joypad.Button,
	released1 <-chan joypad.Button,
	frames2 chan<- []byte,
	events2 chan<- event.Event,
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

				event <- display.Event{Type: display.Title, Data: fmt.Sprintf("Render: %s (AVG:%s) + Frame: %v | FPS: (%v:%s)", avgRenderTime.String(), totalAvgRenderTime.String(), avgFrameTime.String(), g.frames, total.String())}
				g.frames = 0
				start = time.Now()

				// make sure avg render times doesn't get too big
				if len(avgRenderTimes) > 144 {
					avgRenderTimes = avgRenderTimes[1:]
				}
			}*/

			// send frames
			frames1 <- frameBuffer1
			frames2 <- frameBuffer2

			// unlock the gameboy
			g.Unlock()
		}
	}
}

func (g *GameBoy) Start(frames chan<- []byte, events chan<- event.Event, pressed <-chan joypad.Button, released <-chan joypad.Button) {
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
	if r, ok := g.Cartridge.MemoryBankController.(cartridge.RAMController); ok && g.Cartridge.Header().RAMSize > 0 {
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

	var ticker *time.Ticker
	// start a ticker
	if g.speed == 0 {
		ticker = time.NewTicker(1)
	} else {
		ticker = time.NewTicker(FrameTime / time.Duration(g.speed))
	}
	frameBuffer := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*3)
	var frame [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8

	// Get a pointer to the first element of the frame array
	framePtr := unsafe.Pointer(&frame[0][0][0])

	// Get a pointer to the first element of the frameBuffer array
	frameBufferPtr := unsafe.Pointer(&frameBuffer[0])

	// update window title
	events <- event.Event{Type: event.Title, Data: fmt.Sprintf("GomeBoy (%s)", g.Cartridge.Header().Title)}

	// create event handlers for input
	for i := joypad.ButtonA; i <= joypad.ButtonDown; i++ {
		_i := i
		g.Scheduler.RegisterEvent(scheduler.JoypadA+scheduler.EventType(_i), func() {
			g.Joypad.Press(_i)
		})
		g.Scheduler.RegisterEvent(scheduler.JoypadARelease+scheduler.EventType(_i), func() {
			g.Joypad.Release(_i)
		})
	}

emuLoop:
	for {
		select {
		case b := <-pressed:
			// press button with some entropy by pressing at a random cycle in the future
			g.Scheduler.ScheduleEvent(scheduler.EventType(uint8(scheduler.JoypadA)+b), uint64(1024+rand.Intn(4192)*4))
		case b := <-released:
			until := g.Scheduler.Until(scheduler.JoypadA + scheduler.EventType(b))
			g.Scheduler.ScheduleEvent(scheduler.EventType(uint8(scheduler.JoypadARelease)+b), until+uint64(1024+rand.Intn(1024)*4))
		case cmd := <-g.cmdChannel:
			g.Lock()
			switch cmd.Command {
			case emulator.CommandPause:
				g.paused = true
				g.APU.Pause()
			case emulator.CommandResume:
				g.paused = false
				g.APU.Play()
			case emulator.CommandClose:
				// once the gameboy is closed, stop the ticker
				ticker.Stop()

				// close the save file
				if g.save != nil {
					b := ram.SaveRAM()
					if err := g.save.SetBytes(b); err != nil {
						g.Logger.Errorf("error saving emulator: %v", err)
					}
					if err := g.save.Close(); err != nil {
						g.Logger.Errorf("error closing save file: %v", err)
					}
				}
				g.running = false
				break emuLoop
			}
			g.Unlock()
		case <-saveTicker.C:
			g.Lock()
			// get the data from the RAM
			data := ram.SaveRAM()
			// write the data to the save
			if err := g.save.SetBytes(data); err != nil {
				g.Logger.Errorf("error saving emulator: %v", err)
				g.Unlock()
			}
			g.Unlock()
		default:
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

					totalAvgRenderTime := avgTime(avgRenderTimes)

					events <- event.Event{Type: event.Title, Data: fmt.Sprintf("GomeBoy: %s (AVG:%s) | FPS: %v", avgRenderTime.String(), totalAvgRenderTime.String(), g.frames)}
					events <- event.Event{Type: event.FrameTime, Data: avgRenderTimes}
					g.frames = 0
					start = time.Now()

					// make sure avg render times doesn't get too big
					if len(avgRenderTimes) > 60 {
						avgRenderTimes = avgRenderTimes[1:]
					}

					// send sample data
					events <- event.Event{Type: event.Sample, Data: g.APU.Samples}
				}

				// send frame
				frames <- frameBuffer

				// check printer for queued data
				if g.Printer != nil && g.Printer.HasPrintJob() {
					events <- event.Event{Type: event.Print, Data: g.Printer.GetPrintJob()}
				}
			}

			// wait for next tick
			<-ticker.C
		}
	}

	g.running = false
}

func (g *GameBoy) TogglePause() {
	g.paused = !g.paused
}

// NewGameBoy returns a new GameBoy.
func NewGameBoy(rom []byte, opts ...Opt) *GameBoy {
	sched := scheduler.NewScheduler()

	b := io.NewBus(sched)
	cart := cartridge.NewCartridge(rom, b)
	pad := joypad.New(b)
	serialCtl := serial.NewController(b, sched)
	sound := apu.NewAPU(sched, b)
	timerCtl := timer.NewController(b, sched, sound)
	video := ppu.New(b, sched)
	processor := cpu.NewCPU(b, sched, video)
	video.AttachNotifyFrame(func() {
		processor.HasFrame()
	})

	var model = types.DMGABC
	if cart.Header().GameboyColor() {
		model = types.CGBABC
	}

	g := &GameBoy{
		CPU:    processor,
		PPU:    video,
		Logger: log.New(),
		b:      b,

		APU:        sound,
		Joypad:     pad,
		Timer:      timerCtl,
		Serial:     serialCtl,
		model:      model, // defaults to cart
		speed:      1.0,
		Scheduler:  sched,
		cmdChannel: make(chan emulator.CommandPacket, 10),
	}
	g.Cartridge = cart
	// apply options
	for _, opt := range opts {
		opt(g)
	}

	sound.SetModel(g.model)
	processor.SetModel(g.model)

	// does the cartridge have RAM? (and therefore a save file)
	if ram, ok := cart.MemoryBankController.(cartridge.RAMController); ok && cart.Header().RAMSize > 0 {
		// try to load the save file
		saveFiles, err := emulator.LoadSaves(g.Cartridge.Title())

		if err != nil {
			// was there an error loading the save files?
			g.Logger.Errorf("error loading save files: %s", err)
		} else {
			// if there are no save files, create one
			if saveFiles == nil || len(saveFiles) == 0 {
				g.Logger.Debugf("no save file found for %s", g.Cartridge.Title())

				g.save, err = emulator.NewSave(g.Cartridge.Title(), g.Cartridge.Header().RAMSize)
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
	g.b.Map(g.model, cart.Header().GameboyColor(), g.APU.Read)
	g.CPU.Boot(g.model)
	g.b.Boot()

	// handle colourisation
	if !cart.Header().GameboyColor() && (g.model == types.CGBABC || g.model == types.CGB0) {
		video.BGColourisationPalette = &palette.Palette{}
		video.OBJ0ColourisationPalette = &palette.Palette{}
		video.OBJ1ColourisationPalette = &palette.Palette{}
		colourisationPalette := palette.LoadColourisationPalette([]byte(cart.Header().Title))

		for i := 0; i < 4; i++ {
			video.BGColourisationPalette[i] = colourisationPalette.BG[i]
			video.OBJ0ColourisationPalette[i] = colourisationPalette.OBJ0[i]
			video.OBJ1ColourisationPalette[i] = colourisationPalette.OBJ1[i]
		}

	}

	return g
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
	g.model = m
}

var _ types.Stater = (*GameBoy)(nil)

func (g *GameBoy) Load(s *types.State) {
	g.CPU.Load(s)
	g.PPU.Load(s)
	// g.APU.LoadRAM(s) TODO implement APU state
	g.Joypad.Load(s)
	g.Serial.Load(s)
}

func (g *GameBoy) Save(s *types.State) {
	g.CPU.Save(s)
	g.PPU.Save(s)
	// g.APU.SaveRAM(s) TODO implement APU state
	g.Joypad.Save(s)
	g.Serial.Save(s)
}
