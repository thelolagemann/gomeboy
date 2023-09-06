package tests

import (
	"context"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	mooneyeROMPath = "roms/mooneye"
)

type mooneyeTest struct {
	romPath string
	name    string
	passed  bool
	model   types.Model
}

func assertModel(file os.DirEntry) types.Model {
	model := types.DMGABC
	if len(file.Name()) < 8 {
		return model
	}
	// try to determine the model
	// ends with -dmg0.gb
	if file.Name()[len(file.Name())-8:] == "-dmg0.gb" {
		model = types.DMG0
	}
	// ends with -mgb.gb
	if file.Name()[len(file.Name())-7:] == "-mgb.gb" {
		model = types.MGB
	}
	// ends with -sgb.gb
	if file.Name()[len(file.Name())-7:] == "-sgb.gb" {
		model = types.SGB
	}
	// ends with -S.gb
	if file.Name()[len(file.Name())-5:] == "-S.gb" {
		model = types.SGB
	}
	// ends with -sgb2.gb
	if file.Name()[len(file.Name())-8:] == "-sgb2.gb" {
		model = types.SGB2
	}
	// ends with 2-S.gb
	if file.Name()[len(file.Name())-6:] == "2-S.gb" {
		model = types.SGB2
	}
	// ends with -cgb0.gb
	if file.Name()[len(file.Name())-8:] == "-cgb0.gb" {
		model = types.CGB0
	}
	// ends with -cgb.gb
	if file.Name()[len(file.Name())-7:] == "-cgb.gb" {
		model = types.CGBABC
	}
	// ends with -C.gb
	if file.Name()[len(file.Name())-5:] == "-C.gb" {
		model = types.CGBABC
	}
	// ends with -A.gb
	if file.Name()[len(file.Name())-5:] == "-A.gb" {
		model = types.AGB
	}
	// ends with -cgbABCDE.gb
	if strings.Contains(file.Name(), "-cgbABCDE.gb") {
		model = types.CGBABC
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
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
			model:   assertModel(file),
		})
	}

	return tc
}
func (m *mooneyeTest) Name() string {
	return m.name
}

func (m *mooneyeTest) Run(t *testing.T) {
	if pass := testMooneyeROM(t, m.romPath, m.model); pass {
		m.passed = true
	}
}

func (m *mooneyeTest) Passed() bool {
	return m.passed
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
	madness.Add(&imageTest{
		romPath:         filepath.Join(mooneyeROMPath, "madness", "mgb_oam_dma_halt_sprites.gb"),
		expectedImage:   findImage("mgb_oam_dma_halt_sprites", types.MGB),
		name:            "mgb_oam_dma_halt_sprites",
		emulatedSeconds: 5,
		model:           types.MGB,
	})

	// misc
	misc := newMooneyeTestCollectionFromDir(tS, "misc")
	newMooneyeTestCollectionFromCollection(misc, "bits")
	newMooneyeTestCollectionFromCollection(misc, "ppu")

	// sprite_priority (image test)
	manualOnly := tS.NewTestCollection("manual-only")
	manualOnly.Add(&imageTest{
		romPath:         filepath.Join(mooneyeROMPath, "manual-only", "sprite_priority.gb"),
		expectedImage:   findImage("sprite_priority", types.DMGABC),
		name:            "sprite_priority",
		emulatedSeconds: 5,
		model:           types.DMGABC,
	})
	manualOnly.Add(&imageTest{
		romPath:         filepath.Join(mooneyeROMPath, "manual-only", "sprite_priority.gb"),
		expectedImage:   findImage("sprite_priority", types.CGBABC),
		name:            "sprite_priority",
		emulatedSeconds: 5,
		model:           types.CGBABC,
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
			model:   assertModel(file),
		})
	}

	return tc
}

// testMooneyeROM tests a mooneye rom. A passing test will
// execute the rom until the breakpoint is reached (LD B, B),
// and writes the fibonacci sequence 3/5/8/13/21/34 to the
// registers B, C, D, E, H, L. The test will then compare the
// registers to the expected values.
func testMooneyeROM(t *testing.T, romFile string, model types.Model) bool {
	passed := true
	t.Run(filepath.Base(romFile), func(t *testing.T) {
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
