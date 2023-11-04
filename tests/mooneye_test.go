package tests

import (
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	mooneyeROMPath = "roms/mooneye"
)

type mooneyeTest struct {
	*basicTest
	emulatedSeconds int
}

var modelSuffixes = map[string]types.Model{
	"-dmg0":     types.DMG0,
	"-mgb":      types.MGB,
	"-sgb":      types.SGB,
	"-S":        types.SGB,
	"-sgb2":     types.SGB2,
	"2-S":       types.SGB2,
	"-cgb0":     types.CGB0,
	"-cgb":      types.CGBABC,
	"-C":        types.CGBABC,
	"-A":        types.AGB,
	"-cgbABCDE": types.CGBABC,
}

func assertModel(file os.DirEntry) types.Model {
	model := types.DMGABC
	if len(file.Name()) < 8 {
		return model
	}
	for s, m := range modelSuffixes {
		if strings.HasSuffix(strings.Split(file.Name(), ".")[0], s) {
			// handle -S and 2-S being matched as the same
			if m == types.SGB && strings.HasSuffix(strings.Split(file.Name(), ".")[0], "2-S") {
				m = types.SGB2
			}
			model = m
			break
		}
	}

	return model
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
			basicTest: &basicTest{
				romPath: filepath.Join(romDir, file.Name()),
				name:    strings.Split(file.Name(), ".")[0],
				model:   assertModel(file),
			},
		})
	}

	return tc
}

func (m *mooneyeTest) Run(t *testing.T) {
	m.passed = true
	t.Run(filepath.Base(m.name), func(t *testing.T) {
		var g *gameboy.GameBoy
		var err error
		if m.emulatedSeconds > 0 {
			g, err = runGameboy(m.romPath, m.emulatedSeconds, CycleBreakpoint, gameboy.Debug(), gameboy.AsModel(m.model))
		} else {
			g, err = runGameboy(m.romPath, 5, DebugBreakpoint, gameboy.Debug(), gameboy.AsModel(m.model))
		}
		if err != nil {
			m.passed = false
			t.Errorf("failed to run gameboy: %s", err)
		}

		expectedRegisters := []uint8{3, 5, 8, 13, 21, 34}
		for i, r := range []uint8{g.CPU.B, g.CPU.C, g.CPU.D, g.CPU.E, g.CPU.H, g.CPU.L} {
			if r != expectedRegisters[i] {
				t.Errorf("expected register %d to be %d, got %d", i, expectedRegisters[i], r)
				m.passed = false
			}
		}
	})
}

func testMooneye(roms *TestTable) {
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
	madness := tS.NewTestCollection("madness")
	madness.Add(newImageTest("mgb_oam_dma_halt_sprites", withEmulatedSeconds(2), asModel(types.MGB)))

	// misc
	misc := newMooneyeTestCollectionFromDir(tS, "misc")
	newMooneyeTestCollectionFromCollection(misc, "bits")
	newMooneyeTestCollectionFromCollection(misc, "ppu")

	// sprite_priority (image test)
	manualOnly := tS.NewTestCollection("manual-only")
	manualOnly.AddTests(imageTestForModels("sprite_priority", 1, types.DMGABC, types.CGBABC)...)
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
			basicTest: &basicTest{
				romPath: filepath.Join(romDir, file.Name()),
				name:    strings.Split(file.Name(), ".")[0],
				model:   assertModel(file),
			},
		})
	}

	return tc
}
