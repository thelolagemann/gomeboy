package tests

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// imageTest is a test that compares the output of a rom to an expected image
type imageTest struct {
	romPath         string
	name            string
	emulatedSeconds int
	expectedImage   string
	passed          bool
	model           types.Model
}

type imageTestOption func(*imageTest)

func withEmulatedSeconds(secs int) imageTestOption {
	return func(t *imageTest) {
		t.emulatedSeconds = secs
	}
}

func asModel(model types.Model) imageTestOption {
	return func(t *imageTest) {
		t.model = model
	}
}

func findImage(name string, model types.Model) string {
	// search for .png file in roms/name
	dir := "roms/" + name

	// does the directory exist?
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// nope, search for roms/*
		dir = "roms/"
	}
	imagePath := ""
	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(d.Name()) == ".png" {
			// does the image name contain the rom name?
			if !strings.Contains(d.Name(), name) {
				return nil
			}
			// check if file name contains model (last 3 chars before .png)
			modelSuffix := d.Name()[len(d.Name())-7 : len(d.Name())-4]

			if strings.Contains(modelSuffix, "dmg") && model == types.DMGABC {
				imagePath = path
				return nil
			}
			if strings.Contains(modelSuffix, "cgb") && model == types.CGBABC {
				imagePath = path
				return nil
			}
			if modelSuffix != "dmg" && modelSuffix != "cgb" {
				imagePath = path
				return nil
			}

			// is the image for both models? (last 7 chars are dmg-cgb or cgb-dmg)
			modelSuffix = d.Name()[len(d.Name())-11 : len(d.Name())-4]
			if strings.Contains(modelSuffix, "dmg-cgb") || strings.Contains(modelSuffix, "cgb-dmg") {
				imagePath = path
				return nil
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}

	if imagePath == "" {
		panic(fmt.Sprintf("could not find image for rom %s", name))
	}

	return imagePath
}

func findROM(name string) string {
	// search for .gb/.gbc file in roms/name
	dir := "roms/" + name

	// does the directory exist?
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// nope, search roms/* for rom
		dir = "roms/"
	}
	romPath := ""
	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}
		if !d.IsDir() && (filepath.Ext(d.Name()) == ".gb" || filepath.Ext(d.Name()) == ".gbc") {
			// were we searching roms/* or roms/name?
			if dir == "roms/" {
				// we were searching roms/*, so we need to check if the name of the rom
				// matches the name of the directory
				if strings.Split(name, "/")[0] != d.Name()[:len(d.Name())-3] {
					return nil
				}
			}
			romPath = path
		}
		return nil
	}); err != nil {
		panic(err)
	}
	if romPath == "" {
		panic("no rom found for " + name)
	}

	return romPath
}

func newImageTest(name string, opts ...imageTestOption) *imageTest {
	// discover rom path
	romPath := findROM(name)

	t := &imageTest{
		romPath:         romPath,
		name:            name,
		model:           types.DMGABC,
		emulatedSeconds: 2,
	}
	for _, opt := range opts {
		opt(t)
	}
	if t.model == types.CGBABC {
		t.name = t.name + "-cgb"
	}

	// discover expected image path based on name and model
	t.expectedImage = findImage(name, t.model)

	// does the expected image exist?
	if _, err := os.Stat(t.expectedImage); os.IsNotExist(err) {
		panic("expected image does not exist: " + t.expectedImage)
	}
	return t
}

func (t *imageTest) Run(tester *testing.T) {
	t.passed = testROMWithExpectedImage(tester, t.romPath, t.expectedImage, t.model, t.emulatedSeconds, t.name)
}

func (t *imageTest) Name() string {
	return t.name
}

func (t *imageTest) Passed() bool {
	return t.passed
}

func ImgCompare(img1, img2 image.Image) (int64, image.Image, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if bounds1 != bounds2 {
		return math.MaxInt64, nil, fmt.Errorf("image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
	}

	accumError := int64(0)
	resultImg := image.NewNRGBA(image.Rect(
		bounds1.Min.X,
		bounds1.Min.Y,
		bounds1.Max.X,
		bounds1.Max.Y,
	))
	draw.Draw(resultImg, resultImg.Bounds(), img1, image.Point{0, 0}, draw.Src)

	for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
		for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
			r1, g1, b1, a1 := img1.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()

			diff := int64(sqDiffUInt32(r1, r2))
			diff += int64(sqDiffUInt32(g1, g2))
			diff += int64(sqDiffUInt32(b1, b2))
			diff += int64(sqDiffUInt32(a1, a2))

			if diff > 0 {
				accumError += diff
				resultImg.Set(
					bounds1.Min.X+x,
					bounds1.Min.Y+y,
					color.NRGBA{R: 255, A: 128})
			}
		}
	}

	return int64(math.Sqrt(float64(accumError))), resultImg, nil
}

func sqDiffUInt32(x, y uint32) uint64 {
	d := uint64(x) - uint64(y)
	return d * d
}

func testROMWithExpectedImage(t *testing.T, romPath string, expectedImagePath string, asModel types.Model, emulatedSeconds int, name string) bool {
	passed := true
	t.Run(name, func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romPath)
		if err != nil {
			panic(fmt.Sprintf("failed to read rom: %s", err))
		}

		// create the emulator
		g := gameboy.NewGameBoy(b, gameboy.AsModel(asModel))

		// custom test loop
		for frame := 0; frame < 60*emulatedSeconds; frame++ {
			for i := uint32(0); i < gameboy.TicksPerFrame; {
				i += uint32(g.CPU.Step())
			}

			// wait until frame is done
			for !g.PPU.HasFrame() {
				g.CPU.Step()
			}
			g.PPU.ClearRefresh()
		}

		img := g.PPU.PreparedFrame

		// create image.Image from the byte array
		img1 := image.NewNRGBA(image.Rect(0, 0, 160, 144))
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
				img1.Set(x, y, col)
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

		img2 := imageFromFilename(expectedImagePath)
		// create a new paletted image
		img3 := image.NewPaletted(img1.Bounds(), palette)
		draw.Draw(img3, img3.Bounds(), img1, image.Point{0, 0}, draw.Src)

		// compare the images
		diff, diffResult, err := ImgCompare(img2, img3)
		if err != nil {
			passed = false
			t.Fatal(err)
		}

		if diff > 0 {
			passed = false
			t.Errorf("Test %s failed. Difference: %d", name, diff)
			// save the diff image
			f, err := os.Create("results/" + name + ".png")
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, diffResult); err != nil {
				t.Fatal(err)
			}

			// save the actual image
			f, err = os.Create("results/" + name + "_actual.png")
			if err != nil {
				t.Fatal(err)
			}

			if err = png.Encode(f, img3); err != nil {
				t.Fatal(err)
			}

			// save the expected image
			f, err = os.Create("results/" + name + "_expected.png")
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, img2); err != nil {
				t.Fatal(err)
			}
		}
	})
	return passed
}

func imageFromFilename(filename string) image.Image {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	return img
}
