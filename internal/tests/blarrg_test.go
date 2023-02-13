package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	blarggROMPath = "roms/blargg"
)

type blarrgTest struct {
	romPath string
	name    string
	passed  bool
	model   gameboy.Model
}

func newBlargTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(blarggROMPath, dir, "individual")
	// check if individual exists, otherwise check if rom-singles exists
	if _, err := os.Stat(romDir); os.IsNotExist(err) {
		romDir = filepath.Join(blarggROMPath, dir, "rom_singles")
	}
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

		tc.Add(&blarrgTest{
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
		})
	}

	return tc
}

func (m *blarrgTest) Name() string {
	return m.name
}

func (m *blarrgTest) Run(t *testing.T) {
	if pass := testBlarggROM(t, m.romPath); pass {
		m.passed = true
	}
}

func (m *blarrgTest) Passed() bool {
	return m.passed
}

func testBlarrg(t *testing.T, table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("blarrg")

	// cgb_sound
	tS.NewTestCollection("cgb_sound").Add(&genericImageTest{
		romPath:         "roms/blargg/cgb_sound/cgb_sound.gb",
		name:            "cgb_sound",
		expectedImage:   "roms/blargg/cgb_sound/cgb_sound-cgb.png",
		emulatedSeconds: 40,
		model:           gameboy.ModelCGB,
	})

	// cpu_instrs
	newBlargTestCollectionFromDir(tS, "cpu_instrs")
	// dmg_sound
	tS.NewTestCollection("dmg_sound").Add(&genericImageTest{
		romPath:         "roms/blargg/dmg_sound/dmg_sound.gb",
		name:            "dmg_sound",
		expectedImage:   "roms/blargg/dmg_sound/dmg_sound-dmg.png",
		emulatedSeconds: 40,
		model:           gameboy.ModelDMG,
	})
	// halt_bug
	tS.NewTestCollection("halt_bug").Add(&genericImageTest{
		romPath:         "roms/blargg/halt_bug/halt_bug.gb",
		name:            "halt_bug",
		expectedImage:   "roms/blargg/halt_bug/halt_bug-dmg-cgb.png",
		emulatedSeconds: 20,
		model:           gameboy.ModelDMG,
	})
	// instr_timing
	tS.NewTestCollection("instr_timing").Add(&genericImageTest{
		romPath:         "roms/blargg/instr_timing/instr_timing.gb",
		name:            "instr_timing",
		expectedImage:   "roms/blargg/instr_timing/instr_timing-dmg-cgb.png",
		emulatedSeconds: 2,
		model:           gameboy.ModelDMG,
	})
	// interrupt_time (DMG)
	interruptTime := tS.NewTestCollection("interrupt_time")
	interruptTime.Add(&genericImageTest{
		romPath:         "roms/blargg/interrupt_time/interrupt_time.gb",
		name:            "interrupt_time_dmg",
		expectedImage:   "roms/blargg/interrupt_time/interrupt_time-dmg.png",
		emulatedSeconds: 2,
		model:           gameboy.ModelDMG,
	})
	// interrupt_time (CGB)
	interruptTime.Add(&genericImageTest{
		romPath:         "roms/blargg/interrupt_time/interrupt_time.gb",
		name:            "interrupt_time_cgb",
		expectedImage:   "roms/blargg/interrupt_time/interrupt_time-cgb.png",
		emulatedSeconds: 2,
		model:           gameboy.ModelCGB,
	})
	// mem_timing
	newBlargTestCollectionFromDir(tS, "mem_timing")
}

// testBlarggROM tests a blarrg ROM. A passing test will write
// Passed to the 0xFF01 register. A custom handler is used to intercept
// writes to the 0xFF01 register and check if the test passed.
func testBlarggROM(t *testing.T, romFile string) bool {
	passed := true
	t.Run(filepath.Base(romFile), func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romFile)
		if err != nil {
			t.Fatal(err)
		}
		output := ""
		// create the gameboy
		g := gameboy.NewGameBoy(b, gameboy.SerialDebugger(&output))

		// run the gameboy
		for {
			g.Frame()
			if g.CPU.DebugBreakpoint {
				break
			}
		}

		// check if the test passed
		if strings.Contains(output, "Failed") || !strings.Contains(output, "Passed") {
			passed = false
			t.Errorf("expecting output to contain 'Passed', got '%s'", output)
		}
	})

	return passed
}

// TODO
// add way to test specific models (dmg, cgb, agb) for each pass condition (e.g. a cgb test should fail on dmg, but pass on cgb)
