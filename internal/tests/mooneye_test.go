package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	mooneyeROMPath = "roms/mooneye/acceptance"
)

type mooneyeTest struct {
	romPath string
	name    string
	passed  bool
}

func newMooneyeTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(mooneyeROMPath, dir)
	tc := suite.NewTestCollection(dir)
	if dir == "misc" {
		romDir = mooneyeROMPath
		dir = ""
	}

	// read the directory
	files, err := os.ReadDir(romDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		tc.Add(&mooneyeTest{
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
		})
	}

	return tc
}
func (m *mooneyeTest) Name() string {
	return m.name
}

func (m *mooneyeTest) Run(t *testing.T) {
	if pass := testMooneyeROM(t, m.romPath); pass {
		m.passed = true
	}
}

func (m *mooneyeTest) Passed() bool {
	return m.passed
}

func testMooneye(t *testing.T, roms *TestTable) {
	// create top level test
	tS := roms.NewTestSuite("mooneye")

	// bits
	newMooneyeTestCollectionFromDir(tS, "bits")

	// instr
	newMooneyeTestCollectionFromDir(tS, "instr")

	// interrupts
	newMooneyeTestCollectionFromDir(tS, "interrupts")

	// oam_dma
	newMooneyeTestCollectionFromDir(tS, "oam_dma")

	// ppu
	newMooneyeTestCollectionFromDir(tS, "ppu")

	// serial
	newMooneyeTestCollectionFromDir(tS, "serial")

	// timer
	newMooneyeTestCollectionFromDir(tS, "timer")

	// individual
	newMooneyeTestCollectionFromDir(tS, "misc")
}

// testMooneyeROM tests a mooneye rom. A passing test will
// execute the rom until the breakpoint is reached (LD B, B),
// and writes the fibonacci sequence 3/5/8/13/21/34 to the
// registers B, C, D, E, H, L. The test will then compare the
// registers to the expected values.
func testMooneyeROM(t *testing.T, romFile string) bool {
	passed := true
	t.Run(filepath.Base(romFile), func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romFile)
		if err != nil {
			panic(err)
		}

		// create the gameboy
		g := gameboy.NewGameBoy(b, gameboy.Debug(), gameboy.AsModel(gameboy.ModelDMG))

		takenTooLong := false
		go func() {
			// run the gameboy for 10 seconds TODO figure out how long it should take
			time.Sleep(3 * time.Second)
			takenTooLong = true
		}()
		// run until breakpoint
		for {
			g.Frame()
			if g.CPU.DebugBreakpoint || takenTooLong {
				break
			}
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
