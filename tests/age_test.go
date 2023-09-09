package tests

import (
	"context"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	ageROMPath = "roms/age"
)

// assertModelsPassed is a helper function to assert which models
// should pass a test given the filename.
//
// e.g.
//
//	ei-halt-dmgC-cgbBCE.gb should pass on DMGABC and CGBABC
func assertModelsPassed(file os.DirEntry) []types.Model {
	// get name
	name := file.Name()

	// ends with cgbE -> should pass on CGBE
	if name[len(name)-len("cgbE.gb"):] == "cgbE.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBE
	}

	// ends with ncmE -> should pass on CGBE (non CGB mode)
	if name[len(name)-len("ncmE.gb"):] == "ncmE.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBE
	}

	// ends with ncmBC -> should pass on CGBBC (non CGB mode)
	if name[len(name)-len("ncmBC.gb"):] == "ncmBC.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBBC
	}

	// ends with cgbBC -> should pass on CGBBC
	if name[len(name)-len("cgbBC.gb"):] == "cgbBC.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBBC
	}

	// ends with dmgC-cgbBC -> should pass on DMGABC and CGBBC
	if name[len(name)-len("dmgC-cgbBC.gb"):] == "dmgC-cgbBC.gb" {
		return []types.Model{types.DMGABC, types.CGBABC} // TODO correctly differentiate between CGBABC and CGBBC
	}

	// ends with dmgC-cgbBCE -> should pass on DMGABC and CGBABC
	if name[len(name)-len("dmgC-cgbBCE.gb"):] == "dmgC-cgbBCE.gb" {
		return []types.Model{types.DMGABC, types.CGBABC}
	}

	// default to DMGABC
	return []types.Model{types.DMGABC}
}

type ageTest struct {
	romPath string
	name    string
	passed  bool
	model   types.Model
}

func (a *ageTest) Run(t *testing.T) {
	a.passed = testAGERom(t, a.romPath, a.model)
}

func (a *ageTest) Passed() bool {
	return a.passed
}

func (a *ageTest) Name() string {
	return fmt.Sprintf("%s (%s)", a.name, a.model)
}

func testAGERom(t *testing.T, romFile string, model types.Model) bool {
	passed := true
	t.Run(fmt.Sprintf("%s (%s)", filepath.Base(romFile[:len(romFile)-len(filepath.Ext(romFile))]), model), func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romFile)
		if err != nil {
			panic(err)
		}

		// create the gameboy
		g := gameboy.NewGameBoy(b, gameboy.Debug(), gameboy.AsModel(model), gameboy.NoAudio(), gameboy.WithLogger(log.NewNullLogger()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		go func() {
			<-ctx.Done()
			g.CPU.DebugBreakpoint = true
		}()
		frame := 0
		// run until breakpoint
		for {
			g.Frame()
			if g.CPU.DebugBreakpoint || frame > (60*10) { // 10 seconds
				break
			}
			frame++
		}

		expectedRegisters := []uint8{3, 5, 8, 13, 21, 34}
		for i, r := range []uint8{g.CPU.B, g.CPU.C, g.CPU.D, g.CPU.E, g.CPU.H, g.CPU.L} {
			if r != expectedRegisters[i] {
				t.Errorf("expected register %d to be %d, got %d", i, expectedRegisters[i], r)
				passed = false
			}
		}
	})
	return passed
}

func newAgeTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(ageROMPath, dir)
	tc := suite.NewTestCollection(dir)

	// read the directory
	files, err := os.ReadDir(romDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".gb" {
			continue
		}

		// get models that should pass
		models := assertModelsPassed(file)

		// create test for each model
		for _, model := range models {
			tc.Add(&ageTest{
				romPath: filepath.Join(romDir, file.Name()),
				name:    file.Name(),
				model:   model,
			})
		}
	}

	return tc
}

func TestAge(t *testing.T) {
	// create top level test
	//tS := table.NewTestSuite("age")
	table := &TestTable{}
	tS := table.NewTestSuite("age")
	// halt
	newAgeTestCollectionFromDir(tS, "halt").Run(t)
	// lcd-align-ly
	newAgeTestCollectionFromDir(tS, "lcd-align-ly").Run(t)
	// ly
	newAgeTestCollectionFromDir(tS, "ly").Run(t)
	// oam
	newAgeTestCollectionFromDir(tS, "oam").Run(t)
	// stat-interrupt
	newAgeTestCollectionFromDir(tS, "stat-interrupt").Run(t)
	// stat-mode
	newAgeTestCollectionFromDir(tS, "stat-mode").Run(t)
	// stat-mode-sprites
	newAgeTestCollectionFromDir(tS, "stat-mode-sprites").Run(t)
	// stat-mode-window
	newAgeTestCollectionFromDir(tS, "stat-mode-window").Run(t)
	// vram
	newAgeTestCollectionFromDir(tS, "vram").Run(t)
}

func testAge(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("age")

	// halt
	newAgeTestCollectionFromDir(tS, "halt")
	// lcd-align-ly
	newAgeTestCollectionFromDir(tS, "lcd-align-ly")
	// ly
	newAgeTestCollectionFromDir(tS, "ly")
	// oam
	newAgeTestCollectionFromDir(tS, "oam")
	// stat-interrupt
	newAgeTestCollectionFromDir(tS, "stat-interrupt")
	// stat-mode
	newAgeTestCollectionFromDir(tS, "stat-mode")
	// stat-mode-sprites
	newAgeTestCollectionFromDir(tS, "stat-mode-sprites")
	// stat-mode-window
	newAgeTestCollectionFromDir(tS, "stat-mode-window")
	// vram
	newAgeTestCollectionFromDir(tS, "vram")
}
