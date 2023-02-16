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

var (
	// blarggImageTests holds all the tests that are image based,
	// as they don't output any data to the 0xFF01 register
	blarggImageTests = []ROMTest{
		newImageTest("cgb_sound", asModel(gameboy.ModelCGB), withEmulatedSeconds(40)),
		newImageTest("dmg_sound", withEmulatedSeconds(40)),
		newImageTest("halt_bug", withEmulatedSeconds(20)),
		newImageTest("halt_bug", asModel(gameboy.ModelCGB), withEmulatedSeconds(20)),
		newImageTest("instr_timing", withEmulatedSeconds(20)),
		newImageTest("interrupt_time", withEmulatedSeconds(2)),
		newImageTest("interrupt_time", asModel(gameboy.ModelCGB), withEmulatedSeconds(2)),
	}
)

func Test_Blargg(t *testing.T) {
	testROMs(t, blarggImageTests...)
}

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

func testBlarrg(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("blarrg")

	// cgb_sound
	tS.NewTestCollection("cgb_sound").Add(blarggImageTests[0])

	// cpu_instrs
	newBlargTestCollectionFromDir(tS, "cpu_instrs")
	// dmg_sound
	tS.NewTestCollection("dmg_sound").Add(blarggImageTests[1])
	// halt_bug
	tS.NewTestCollection("halt_bug").AddTests(blarggImageTests[2], blarggImageTests[3])
	// instr_timing
	tS.NewTestCollection("instr_timing").Add(blarggImageTests[4])
	// interrupt_time (DMG)
	tS.NewTestCollection("interrupt_time").AddTests(blarggImageTests[5], blarggImageTests[6])
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
