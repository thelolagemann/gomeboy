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

var (
	dmgSoundTests = []ROMTest{
		newImageTest("dmg_sound/01-registers", withEmulatedSeconds(2)),
		newImageTest("dmg_sound/02-len ctr", withEmulatedSeconds(10)),
		newImageTest("dmg_sound/03-trigger", withEmulatedSeconds(17)),
		newImageTest("dmg_sound/04-sweep", withEmulatedSeconds(3)),
		newImageTest("dmg_sound/05-sweep details", withEmulatedSeconds(3)),
		newImageTest("dmg_sound/06-overflow on trigger", withEmulatedSeconds(2)),
		newImageTest("dmg_sound/07-len sweep period sync", withEmulatedSeconds(1)),
		newImageTest("dmg_sound/08-len ctr during power", withEmulatedSeconds(3)),
		newImageTest("dmg_sound/09-wave read while on", withEmulatedSeconds(4)),
		newImageTest("dmg_sound/10-wave trigger while on", withEmulatedSeconds(10)),
		newImageTest("dmg_sound/11-regs after power", withEmulatedSeconds(2)),
		newImageTest("dmg_sound/12-wave write while on", withEmulatedSeconds(10)),
	}

	cgbSoundTests = []ROMTest{
		newImageTest("cgb_sound/01-registers", asModel(types.CGBABC), withEmulatedSeconds(2)),
		newImageTest("cgb_sound/02-len ctr", asModel(types.CGBABC), withEmulatedSeconds(10)),
		newImageTest("cgb_sound/03-trigger", asModel(types.CGBABC), withEmulatedSeconds(17)),
		newImageTest("cgb_sound/04-sweep", asModel(types.CGBABC), withEmulatedSeconds(3)),
		newImageTest("cgb_sound/05-sweep details", asModel(types.CGBABC), withEmulatedSeconds(3)),
		newImageTest("cgb_sound/06-overflow on trigger", asModel(types.CGBABC), withEmulatedSeconds(2)),
		newImageTest("cgb_sound/07-len sweep period sync", asModel(types.CGBABC), withEmulatedSeconds(1)),
		newImageTest("cgb_sound/08-len ctr during power", asModel(types.CGBABC), withEmulatedSeconds(3)),
		newImageTest("cgb_sound/09-wave read while on", asModel(types.CGBABC), withEmulatedSeconds(4)),
		newImageTest("cgb_sound/10-wave trigger while on", asModel(types.CGBABC), withEmulatedSeconds(10)),
		newImageTest("cgb_sound/11-regs after power", asModel(types.CGBABC), withEmulatedSeconds(2)),
		newImageTest("cgb_sound/12-wave", asModel(types.CGBABC), withEmulatedSeconds(10)),
	}

	// blarggImageTests holds all the tests that are image based,
	// as they don't output any data to the 0xFF01 register
	blarggImageTests = append(
		imageTestForModels("halt_bug", 20, types.DMGABC, types.CGBABC),
		append(imageTestForModels("interrupt_time", 2, types.DMGABC, types.CGBABC),
			newImageTest("instr_timing", withEmulatedSeconds(20)))...,
	)
)

func Test_Blargg(t *testing.T) {
	testROMs(t, blarggImageTests...)
	testROMs(t, dmgSoundTests...)
	testROMs(t, cgbSoundTests...)
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
		_, err := runGameboy(b.romPath, 5, DebugBreakpoint, gameboy.SerialDebugger(&output))
		if err != nil {
			t.Error(err)
			return
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

	tS.NewTestCollection("cgb_sound").AddTests(cgbSoundTests...)
	newBlarggTestCollectionFromDir(tS, "cpu_instrs")
	tS.NewTestCollection("dmg_sound").AddTests(dmgSoundTests...)
	tS.NewTestCollection("halt_bug").AddTests(blarggImageTests[0], blarggImageTests[1])
	tS.NewTestCollection("instr_timing").AddTests(blarggImageTests[4])
	tS.NewTestCollection("interrupt_time").AddTests(blarggImageTests[2], blarggImageTests[3])
	newBlarggTestCollectionFromDir(tS, "mem_timing")
}
