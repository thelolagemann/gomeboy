package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"os"
	"path/filepath"
	"testing"
)

const (
	mooneyeROMPath = "roms/mooneye"
)

type mooneyeTest struct {
	romPath string
	name    string
	passed  bool
}

func newMooneyeTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(mooneyeROMPath, dir)
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

	// create test collections
	acceptance := newMooneyeTestCollectionFromDir(tS, "acceptance")

	// bits
	newMooneyeTestCollectionFromCollection(acceptance, "bits")

	// instr
	newMooneyeTestCollectionFromCollection(acceptance, "instr")

	// interrupts
	newMooneyeTestCollectionFromCollection(acceptance, "interrupts")

	// oam_dma
	newMooneyeTestCollectionFromCollection(acceptance, "oam_dma")

	// ppu
	newMooneyeTestCollectionFromCollection(acceptance, "ppu")

	// serial
	newMooneyeTestCollectionFromCollection(acceptance, "serial")

	// timer
	newMooneyeTestCollectionFromCollection(acceptance, "timer")

	// emualator-only (mbc1, mbc2, mbc5)
	emulatorOnly := newMooneyeTestCollectionFromDir(tS, "emulator-only")
	newMooneyeTestCollectionFromCollection(emulatorOnly, "mbc1")
	newMooneyeTestCollectionFromCollection(emulatorOnly, "mbc2")
	newMooneyeTestCollectionFromCollection(emulatorOnly, "mbc5")

	// madness
	newMooneyeTestCollectionFromDir(tS, "madness")

	// misc
	misc := newMooneyeTestCollectionFromDir(tS, "misc")
	newMooneyeTestCollectionFromCollection(misc, "bits")
	newMooneyeTestCollectionFromCollection(misc, "ppu")

	// sprite_priority (image test)
	manualOnly := tS.NewTestCollection("manual-only")
	manualOnly.Add(&imageTest{
		romPath:         filepath.Join(mooneyeROMPath, "manual-only", "sprite_priority.gb"),
		expectedImage:   findImage("sprite_priority", gameboy.ModelDMG),
		name:            "sprite_priority",
		emulatedSeconds: 5,
		model:           gameboy.ModelDMG,
	})
}

func newMooneyeTestCollectionFromCollection(collection *TestCollection, s string) *TestCollection {
	romDir := filepath.Join(mooneyeROMPath, collection.name, s)
	tc := collection.NewTestCollection(s)

	// read the directory
	files, err := os.ReadDir(romDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".gb" {
			continue
		}

		tc.Add(&mooneyeTest{
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
		})
	}

	return tc
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
		g := gameboy.NewGameBoy(b, gameboy.Debug())

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
