package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"testing"
)

type blarrgTest struct {
	romPath       string
	name          string
	expectedImage string
	passed        bool
	model         gameboy.Model
}

func (m *blarrgTest) Name() string {
	return m.name
}

func (m *blarrgTest) Run(t *testing.T) {
	if pass := testROMWithExpectedImage(t, m.romPath, m.expectedImage, m.model); pass {
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
	tS.NewTestCollection("cgb_sound").Add(&blarrgTest{
		romPath:       "roms/blargg/cgb_sound/cgb_sound.gb",
		expectedImage: "roms/blargg/cgb_sound/cgb_sound-cgb.png",
		name:          "cgb_sound",
		model:         gameboy.ModelCGB,
	})

	// cpu_instrs
	tS.NewTestCollection("cpu_instrs").Add(&blarrgTest{
		romPath:       "roms/blargg/cpu_instrs/cpu_instrs.gb",
		expectedImage: "roms/blargg/cpu_instrs/cpu_instrs-dmg-cgb.png",
		name:          "cpu_instrs",
		model:         gameboy.ModelDMG,
	})

	// dmg_sound
	tS.NewTestCollection("dmg_sound").Add(&blarrgTest{
		romPath:       "roms/blargg/dmg_sound/dmg_sound.gb",
		expectedImage: "roms/blargg/dmg_sound/dmg_sound-dmg.png",
		name:          "dmg_sound",
		model:         gameboy.ModelDMG,
	})
}
