package tests

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/types"
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
	emulatedSeconds int
	expectedImage   string

	*basicTest
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

func asName(name string) imageTestOption {
	return func(t *imageTest) {
		t.name = name
	}
}

func findImage(name string, model types.Model) string {
	// strip dirs from name
	name = filepath.Base(name)
	// search for .png file in roms/name
	dir := "roms/" + name
	// are we handling the special case of blargg's dmg_sound and cgb_sound roms?
	if strings.Contains(name, "dmg_sound") {
		dir = "roms/blargg/dmg_sound/rom_singles"
	}
	if strings.Contains(name, "cgb_sound") {
		dir = "roms/blargg/cgb_sound/rom_singles"
	}

	// does the directory exist?
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// nope, search for roms/*
		dir = "roms/"
	}

	imagePath := ""
	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(d.Name()) == ".png" {
			// does the image name contain the rom name?
			if !strings.Contains(d.Name(), filepath.Base(name)) {
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
	// are we handling the special case of blargg's dmg_sound and cgb_sound roms?
	if strings.Contains(name, "dmg_sound") {
		dir = "roms/blargg/dmg_sound/rom_singles"
	}
	if strings.Contains(name, "cgb_sound") {
		dir = "roms/blargg/cgb_sound/rom_singles"
	}
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
				// check that base of rom path matches rom name (With extension removed)
				if filepath.Base(name) != strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())) {
					return nil
				}
			}
			// does the rom name contain the rom name?
			if !strings.Contains(d.Name(), filepath.Base(name)) {
				return nil
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

func newImageTest(name string, opts ...imageTestOption) ROMTest {
	// discover rom path
	romPath := findROM(name)

	t := &imageTest{
		basicTest:       newBasicTest(romPath, types.DMGABC),
		emulatedSeconds: 2,
	}
	for _, opt := range opts {
		opt(t)
	}

	// discover expected image path based on name and model
	t.expectedImage = findImage(name, t.model)

	// does the expected image exist?
	if _, err := os.Stat(t.expectedImage); os.IsNotExist(err) {
		panic("expected image does not exist: " + t.expectedImage)
	}

	// adjust name to remove any leading dirs
	t.name = filepath.Base(t.name)

	return t
}

func imageTestForModels(name string, emulatedSeconds int, models ...types.Model) []ROMTest {
	var tests []ROMTest
	for _, m := range models {
		t := newImageTest(name, asModel(m), withEmulatedSeconds(emulatedSeconds), asName(fmt.Sprintf("%s (%s)", name, m)))
		tests = append(tests, t)
	}
	return tests
}

func (i *imageTest) Run(t *testing.T) {
	i.passed = true
	t.Run(i.name, func(t *testing.T) {
		opts := []gameboy.Opt{gameboy.AsModel(i.model)}
		g, err := runGameboy(i.romPath, i.emulatedSeconds, CycleBreakpoint, opts...)
		if err != nil {
			t.Errorf("Test %s failed: %s", i.name, err)
			return
		}

		// compare the images
		diff, diffImg, err := compareImage(i.expectedImage, g)
		if err != nil {
			i.passed = false
			t.Fatal(err)
		}

		if diff > 0 {
			i.passed = false
			t.Errorf("Test %s failed. Difference: %d", i.name, diff)

			// write output image to disk
			outFile, err := os.Create(fmt.Sprintf("results/%s_output.png", i.name))
			if err != nil {
				t.Fatal(err)
			}
			defer outFile.Close()

			if err := png.Encode(outFile, diffImg); err != nil {
			}
		}
	})
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
			r1, g1, b1, _ := img1.At(x, y).RGBA()
			r2, g2, b2, _ := img2.At(x, y).RGBA()

			diff := int64(sqDiffUInt32(r1, r2))
			diff += int64(sqDiffUInt32(g1, g2))
			diff += int64(sqDiffUInt32(b1, b2))

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

// imageFromFilename loads an image from a file.
func imageFromFilename(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// compareImage compares an expected image with the output of the
// provided gameboy.GameBoy.
func compareImage(expectedImage string, gb *gameboy.GameBoy) (int64, image.Image, error) {
	// create image.Image from the byte array
	img1 := image.NewNRGBA(image.Rect(0, 0, 160, 144))
	var palette []color.Color
	for y := 0; y < 144; y++ {
	next:
		for x := 0; x < 160; x++ {
			col := color.NRGBA{
				R: gb.PPU.PreparedFrame[y][x][0],
				G: gb.PPU.PreparedFrame[y][x][1],
				B: gb.PPU.PreparedFrame[y][x][2],
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

	img2, err := imageFromFilename(expectedImage)
	if err != nil {
		return math.MaxInt64, nil, err
	}
	// create a new paletted image
	img3 := image.NewPaletted(img1.Bounds(), palette)
	draw.Draw(img3, img3.Bounds(), img1, img2.Bounds().Min, draw.Src)

	// TODO output results?

	return ImgCompare(img2, img3)
}
