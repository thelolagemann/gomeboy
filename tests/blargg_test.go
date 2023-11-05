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
	blarggROMPath = "roms/blargg"
)

func blarggImageTests() []ROMTest {
	return append(
		imageTestForModels("halt_bug", 20, types.DMGABC, types.CGBABC),
		append(imageTestForModels("interrupt_time", 2, types.DMGABC, types.CGBABC),
			newImageTest("instr_timing", withEmulatedSeconds(20)))...,
	)
}

type blarrgTest struct {
	*basicTest
}

func newBlarggTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
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

		tc.AddTests(&blarrgTest{newBasicTest(filepath.Join(romDir, file.Name()), types.DMGABC)})
	}

	return tc
}

func (b *blarrgTest) Run(t *testing.T) {
	b.passed = true
	t.Run(filepath.Base(b.name), func(t *testing.T) {
		output := ""
		g, err := runGameboy(b.romPath, 5, DebugBreakpoint, gameboy.SerialDebugger(&output))
		if err != nil {
			t.Error(err)
			return
		}

		// if serial output nothing, check the ram at 0xa000
		if output == "" {
			for i := uint16(0xa000); i < 0xb000; i++ {
				output += string(g.Bus.Get(i))
			}
		}
		// check if the test passed
		if strings.Contains(output, "Failed") || !strings.Contains(output, "Passed") {
			b.passed = false
			t.Errorf("expecting output to contain 'Passed', got '%s'", output)
		}
	})
}

func testBlarrg(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("blarrg")

	newBlarggTestCollectionFromDir(tS, "cpu_instrs")
	newBlarggTestCollectionFromDir(tS, "cgb_sound")
	newBlarggTestCollectionFromDir(tS, "dmg_sound")
	t := blarggImageTests()
	tS.NewTestCollection("halt_bug").AddTests(t[0], t[1])
	tS.NewTestCollection("instr_timing").AddTests(t[4])
	tS.NewTestCollection("interrupt_time").AddTests(t[2], t[3])
	newBlarggTestCollectionFromDir(tS, "mem_timing")
}
