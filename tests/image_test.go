package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
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
	model           gameboy.Model
}

type imageTestOption func(*imageTest)

func withEmulatedSeconds(secs int) imageTestOption {
	return func(t *imageTest) {
		t.emulatedSeconds = secs
	}
}

func asModel(model gameboy.Model) imageTestOption {
	return func(t *imageTest) {
		t.model = model
	}
}

func findImage(name string, model gameboy.Model) string {
	// search for .png file in roms/name
	dir := "roms/" + name
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".png" {
			// check if file name contains model (last 3 chars before .png)
			modelSuffix := file.Name()[len(file.Name())-7 : len(file.Name())-4]

			if strings.Contains(modelSuffix, "dmg") && model == gameboy.ModelDMG {
				return filepath.Join(dir, file.Name())
			}
			if strings.Contains(modelSuffix, "cgb") && model == gameboy.ModelCGB {
				return filepath.Join(dir, file.Name())
			}
			if modelSuffix != "dmg" && modelSuffix != "cgb" {
				return filepath.Join(dir, file.Name())
			}
		}
	}

	panic("no image found for rom " + name)
}

func findROM(name string) string {
	// search for .gb/.gbc file in roms/name
	dir := "roms/" + name
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if !file.IsDir() && (filepath.Ext(file.Name()) == ".gb" || filepath.Ext(file.Name()) == ".gbc") {
			return filepath.Join(dir, file.Name())
		}
	}

	panic("no rom found for " + name)
}

func newImageTest(name string, opts ...imageTestOption) *imageTest {
	// discover rom path
	romPath := findROM(name)

	t := &imageTest{
		romPath:         romPath,
		name:            name,
		model:           gameboy.ModelDMG,
		emulatedSeconds: 2,
	}
	for _, opt := range opts {
		opt(t)
	}
	if t.model == gameboy.ModelCGB {
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
