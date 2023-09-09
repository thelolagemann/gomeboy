package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"sort"
	"testing"
)

type inputTest struct {
	name              string
	romPath           string
	expectedImagePath string
	model             types.Model
	inputs            []testInput
	passed            bool
}

func (iT *inputTest) Run(t *testing.T) {
	iT.passed = testROMWithInput(t, iT.romPath, iT.expectedImagePath, iT.model, iT.name, iT.inputs...)
}

func (iT *inputTest) Name() string {
	return iT.name
}

func (iT *inputTest) Passed() bool {
	return iT.passed
}

type testInput struct {
	// the button to press
	button joypad.Button
	// the frame to press the button
	atEmulatedCycle uint64
}

func testROMWithInput(t *testing.T, romPath string, expectedImagePath string, asModel types.Model, name string, inputs ...testInput) bool {
	passed := true
	t.Run(name, func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romPath)
		if err != nil {
			t.Fatal(err)
		}

		// create a new gameboy
		gb := gameboy.NewGameBoy(b, gameboy.AsModel(asModel), gameboy.Speed(0), gameboy.NoAudio(), gameboy.WithLogger(log.NewNullLogger()))

		// setup frame, event and input channels
		frames := make(chan []byte, 144)
		events := make(chan display.Event, 144)
		pressed := make(chan joypad.Button, 10)
		released := make(chan joypad.Button, 10)

		// sort the inputs by cycle (so we can press them in order)
		sort.Slice(inputs, func(i, j int) bool {
			return inputs[i].atEmulatedCycle < inputs[j].atEmulatedCycle
		})

		var lastCycle uint64
		// schedule input events on gameboy to occur at emulated cycles (with some degree of randomization TODO make configurable)
		for _, input := range inputs {
			adjustedCycle := input.atEmulatedCycle
			adjustedCycle += (1024 + uint64(rand.Intn(4192))) * 4

			gb.Scheduler.ScheduleEvent(scheduler.JoypadA+scheduler.EventType(input.button), adjustedCycle)
			gb.Scheduler.ScheduleEvent(scheduler.JoypadARelease+scheduler.EventType(input.button), adjustedCycle+72240)
			lastCycle = adjustedCycle + 72240
		}

		done := make(chan struct{})
		go func() {
			// wait for the cycle
			for gb.Scheduler.Cycle() < lastCycle {
			}
			done <- struct{}{}
			done <- struct{}{}
		}()

		go func() {
			for {
				select {
				case <-done:
					return
				default:
					<-frames
					<-events
				}
			}
		}()

		// start the gameboy
		go func() {
			gb.Start(frames, events, pressed, released)
		}()

		// wait for the test to finish
		<-done

		// wait an additional 5 seconds (60 * 5) frames to wait for test completion
		for frame := 0; frame < 60*5; frame++ {
			// get the next frame
			<-frames
			// empty the event channel
			<-events
		}

		// close the channels
		close(done)

		// create the actual image
		img := gb.PPU.PreparedFrame

		actualImage := image.NewNRGBA(image.Rect(0, 0, 160, 144))
		palette := []color.Color{}
		for y := 0; y < 144; y++ {
		next:
			for x := 0; x < 160; x++ {
				col := color.NRGBA{
					R: img[y][x][0],
					G: img[y][x][1],
					B: img[y][x][2],
					A: 255,
				}
				actualImage.Set(x, y, col)

				// add color if it doesn't exist
				for _, p := range palette {
					r, g, b, _ := p.RGBA()
					r2, g2, b2, _ := col.RGBA()
					if r == r2 && g == g2 && b == b2 {
						continue next
					}
				}
				palette = append(palette, col)
			}
		}

		// compare the image to the expected image
		expectedImage := imageFromFilename(expectedImagePath)
		palImg := image.NewPaletted(actualImage.Bounds(), palette)
		draw.Draw(palImg, palImg.Bounds(), actualImage, image.Point{0, 0}, draw.Src)
		diff, _, err := ImgCompare(palImg, expectedImage)
		if err != nil {
			passed = false
			t.Fatal(err)
		}

		if diff > 0 {
			passed = false
			t.Errorf("images are different: %d%%", diff) // TODO percentage

			// save the actual image
			f, err := os.Create("results/" + name + "_actual.png")
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, palImg)
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	return passed
}
