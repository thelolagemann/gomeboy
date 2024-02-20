package gameboy

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/apu"
	"github.com/thelolagemann/gomeboy/internal/cpu"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/serial"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"github.com/thelolagemann/gomeboy/internal/timer"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"math"
	"sync"
	"time"
	"unsafe"
)

const (
	maxArraySize = ppu.ScreenWidth * ppu.ScreenHeight * 4 // 90KiB (RGBA frame)
	frameSize    = ppu.ScreenWidth * ppu.ScreenHeight * 3 // 67.5KiB (ppu produced frame)
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

	APU    *apu.APU
	Timer  *timer.Controller
	Serial *serial.Controller

	Bus *io.Bus

	log.Logger

	paused, running bool
	frames          int
	previousFrame   [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
	attachedGameBoy *GameBoy
	speed           float64
	Printer         *accessories.Printer
	save            *emulator.Save
	Scheduler       *scheduler.Scheduler
	dontBoot        bool
	rumbling        bool
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

func (g *GameBoy) AttachAudioListener(player func([]uint8)) {
	g.APU.AttachPlayback(player)
}

func (g *GameBoy) StartLinked(
	frames1 chan<- []byte,
	events1 chan<- event.Event,
	pressed1 <-chan io.Button,
	released1 <-chan io.Button,
	frames2 chan<- []byte,
	events2 chan<- event.Event,
	pressed2 <-chan io.Button,
	released2 <-chan io.Button,
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
			g.Bus.Press(p)
		case r := <-released1:
			g.Bus.Release(r)
		case p := <-pressed2:
			g.attachedGameBoy.Bus.Press(p)
		case r := <-released2:
			g.attachedGameBoy.Bus.Release(r)
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

func (g *GameBoy) Start(frames chan<- []byte, events chan<- event.Event, pressed <-chan io.Button, released <-chan io.Button) {
	g.running = true
	// setup fps counter
	g.frames = 0
	start := time.Now()
	frameStart := time.Now()
	renderTimes := make([]time.Duration, 0, int(FrameRate))
	g.APU.Play()

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
	events <- event.Event{Type: event.Title, Data: fmt.Sprintf("GomeBoy (%s)", g.Bus.Cartridge().Title)}

	// create event handlers for input
	for i := io.ButtonA; i <= io.ButtonDown; i++ {
		_i := i
		g.Scheduler.RegisterEvent(scheduler.JoypadA+scheduler.EventType(_i), func() {
			g.Bus.Press(_i)
		})
		g.Scheduler.RegisterEvent(scheduler.JoypadARelease+scheduler.EventType(_i), func() {
			g.Bus.Release(_i)
		})
	}

emuLoop:
	for {
		select {
		case b := <-pressed:
			// press button with some entropy by pressing at a random cycle in the future
			g.Bus.Press(b)
			//g.Scheduler.ScheduleEvent(scheduler.EventType(uint8(scheduler.bA)+Bus), uint64(1024+rand.Intn(4192)*4))
		case b := <-released:
			g.Bus.Release(b)
			//until := g.Scheduler.Until(scheduler.bA + scheduler.EventType(Bus))
			//g.Scheduler.ScheduleEvent(scheduler.EventType(uint8(scheduler.bARelease)+Bus), until+uint64(1024+rand.Intn(1024)*4))
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
					b := g.Bus.Cartridge().RAM
					if err := g.save.SetBytes(b); err != nil {
						g.Logger.Errorf("error saving emulator: %v", err)
					}
					if err := g.save.Close(); err != nil {
						g.Logger.Errorf("error closing save file: %v", err)
					}
					fmt.Println("am saved")
				}
				g.running = false
				break emuLoop
			}
			g.Unlock()
		default:
			// update the fps counter
			g.frames++

			if !g.paused {
				// render frame
				frameStart = time.Now()

				frame = g.Frame()
				if g.rumbling {
					applyHorizontalShake(&frame, g.frames)
				}
				if g.Bus.Cartridge().Features.Accelerometer {
					frame = Rotate2DFrame(frame, -float64(g.Bus.Cartridge().AccelerometerY), float64(g.Bus.Cartridge().AccelerometerX), 0)
				}

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

	b := io.NewBus(sched, rom)
	serialCtl := serial.NewController(b, sched)
	sound := apu.NewAPU(sched, b)
	timerCtl := timer.NewController(b, sched, sound)
	video := ppu.New(b, sched)
	processor := cpu.NewCPU(b, sched, video)

	var model = types.DMGABC
	if b.Cartridge().IsCGBCartridge() {
		model = types.CGBABC
	}

	g := &GameBoy{
		CPU:    processor,
		PPU:    video,
		Logger: log.New(),
		Bus:    b,

		APU:        sound,
		Timer:      timerCtl,
		Serial:     serialCtl,
		model:      model, // defaults to cart
		speed:      1.0,
		Scheduler:  sched,
		cmdChannel: make(chan emulator.CommandPacket, 10),
	}

	// apply options
	for _, opt := range opts {
		opt(g)
	}

	sound.SetModel(g.model)

	// does the cartridge have battery backed RAM? (and therefore a save file)
	if b.Cartridge().Features.Battery {
		// try to load the save file
		var err error
		g.save, err = emulator.NewSave(b.Cartridge().Title, uint(b.Cartridge().RAMSize))

		if err != nil {
			// was there an error loading the save files?
			g.Logger.Errorf(fmt.Sprintf("error loading save files: %s", err))
		} else {
			copy(g.Bus.Cartridge().RAM, g.save.Bytes())
			var length = len(g.save.Bytes())
			if length > 0x2000 {
				length = 0x2000
			}
			g.Bus.CopyTo(0xA000, 0xC000, g.save.Bytes()[:length])
		}
	}
	// try to load cheats using filename of rom
	g.Bus.Map(g.model)
	if !g.dontBoot {
		g.CPU.Boot(g.model)
		g.Colourise()
		g.Bus.Boot()

		// schedule the frame sequencer event for the next 8192 ticks
		g.Scheduler.ScheduleEvent(scheduler.APUFrameSequencer, uint64(8192-g.Scheduler.SysClock()&0x0fff))
	}

	g.Bus.Cartridge().RumbleCallback = func(b bool) {
		g.rumbling = b
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

	// return the prepared frames
	return g.PPU.PreparedFrame, g.attachedGameBoy.PPU.PreparedFrame
}

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

// Frame will step the emulation until the PPU has finished
// rendering the current frame. It will then prepare the frame
// for display, and return it.
func (g *GameBoy) Frame() [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8 {
	g.CPU.Frame()

	return g.PPU.PreparedFrame
}

func (g *GameBoy) SetModel(m types.Model) {
	// re-initialize MMU
	g.model = m
}

func applyHorizontalShake(frame *[ppu.ScreenHeight][ppu.ScreenWidth][3]uint8, offset int) {
	// Create a temporary frame to store the result
	var tempFrame [ppu.ScreenHeight][ppu.ScreenWidth][3]uint8
	for y := 0; y < ppu.ScreenHeight; y++ {
		for x := 0; x < ppu.ScreenWidth; x++ {
			tempFrame[y][x] = frame[y][x]
		}
	}

	const amplitude, frequency = 2.0, 0
	var phase = float64(offset)

	// Calculate the offset based on sine function
	offsetX := func(t float64) int {
		return int(amplitude * math.Sin(2*math.Pi*frequency*t+phase))
	}

	// Apply the oscillating offset
	for y := 0; y < ppu.ScreenHeight; y++ {
		for x := 0; x < ppu.ScreenWidth; x++ {
			// Calculate the time component based on the current x position
			t := float64(x) / float64(ppu.ScreenWidth)

			// Calculate the offset
			offset := offsetX(t)

			// Apply the offset, ensuring it stays within bounds
			newX := x + offset
			if newX >= 0 && newX < ppu.ScreenWidth {
				tempFrame[y][newX] = frame[y][x]
			}
		}
	}

	// Copy the result back to the original frame
	*frame = tempFrame
}

const (
	ScreenWidth  = ppu.ScreenWidth
	ScreenHeight = ppu.ScreenHeight
)

// Point represents a point in 3D space.
type Point struct {
	X, Y, Z float64
}

// Rotate2DFrame rotates a 2D framebuffer in 3D space with perspective correction.
func Rotate2DFrame(frame [ScreenHeight][ScreenWidth][3]uint8, angleX, angleY, angleZ float64) [ScreenHeight][ScreenWidth][3]uint8 {
	angleX /= 64
	angleY /= 64

	var rotatedFrame [ScreenHeight][ScreenWidth][3]uint8

	// Define the rotation matrices.
	rotateX := func(p Point, angleX float64) Point {
		sinX, cosX := math.Sin(angleX), math.Cos(angleX)
		return Point{
			X: p.X,
			Y: p.Y*cosX - p.Z*sinX,
			Z: p.Y*sinX + p.Z*cosX,
		}
	}

	rotateY := func(p Point, angleY float64) Point {
		sinY, cosY := math.Sin(angleY), math.Cos(angleY)
		return Point{
			X: p.X*cosY + p.Z*sinY,
			Y: p.Y,
			Z: -p.X*sinY + p.Z*cosY,
		}
	}

	// Define the viewer's position.
	viewer := Point{X: 0, Y: 0, Z: -10}

	// Iterate over each pixel in the framebuffer.
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			// Define the point in 3D space corresponding to the pixel.
			point := Point{X: float64(x - ScreenWidth/2), Y: float64(y - ScreenHeight/2), Z: 0}

			// Apply rotations around X, Y, and Z axes.
			point = rotateX(point, angleX)
			point = rotateY(point, angleY)

			// Apply perspective correction.
			scale := viewer.Z / (viewer.Z + point.Z)
			projectedX := int((point.X*scale + viewer.X) + ScreenWidth/2)
			projectedY := int((point.Y*scale + viewer.Y) + ScreenHeight/2)

			// Check if the projected point is within the bounds of the framebuffer.
			if projectedX >= 0 && projectedX < ScreenWidth && projectedY >= 0 && projectedY < ScreenHeight {
				// Copy the color of the pixel from the original frame to the rotated frame.
				rotatedFrame[y][x] = frame[projectedY][projectedX]
			}
		}
	}

	return rotatedFrame
}
