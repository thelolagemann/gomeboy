package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

type inputTest struct {
	name              string
	romPath           string
	expectedImagePath string
	model             gameboy.Model
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
	atEmulatedFrame int
}

func testROMWithInput(t *testing.T, romPath string, expectedImagePath string, asModel gameboy.Model, name string, inputs ...testInput) bool {
	passed := true
	t.Run(name, func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romPath)
		if err != nil {
			t.Fatal(err)
		}

		// create a new gameboy
		gb := gameboy.NewGameBoy(b, gameboy.AsModel(asModel))

		// custom test loop
		for frame := 0; frame < 60*10; frame++ {
			for i := uint32(0); i < gameboy.TicksPerFrame; {
				i += uint32(gb.CPU.Step())
			}

			// wait until frame is done
			for !gb.PPU.HasFrame() {
				gb.CPU.Step()
			}
			gb.PPU.ClearRefresh()

			// check if we should press a button
			for _, input := range inputs {
				if input.atEmulatedFrame == frame {
					gb.Joypad.Press(input.button)
				} else {
					gb.Joypad.Release(input.button)
				}
			}
		}

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
