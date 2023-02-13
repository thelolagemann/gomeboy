package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"testing"
)

type acid2Test struct {
	romPath string
	imgPath string
	name    string
	passed  bool
	model   gameboy.Model
}

func (m *acid2Test) Name() string {
	return m.name
}

func (m *acid2Test) Run(t *testing.T) {
	if pass := testROMWithExpectedImage(t, m.romPath, m.imgPath, m.model, 2, m.name); pass {
		m.passed = true
	}
}

func (m *acid2Test) Passed() bool {
	return m.passed
}

func testAcid2(t *testing.T, table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("acid2")

	// create test collection
	tc := tS.NewTestCollection("acid2")

	// add tests
	tc.Add(&acid2Test{
		romPath: "roms/dmg-acid2/dmg-acid2.gb",
		imgPath: "roms/dmg-acid2/dmg-acid2-dmg.png",
		name:    "dmg-acid2",
		model:   gameboy.ModelDMG,
	})
	tc.Add(&acid2Test{
		romPath: "roms/cgb-acid2/cgb-acid2.gbc",
		imgPath: "roms/cgb-acid2/cgb-acid2.png",
		name:    "cgb-acid2",
		model:   gameboy.ModelCGB,
	})
	tc.Add(&acid2Test{
		romPath: "roms/cgb-acid-hell/cgb-acid-hell.gbc",
		imgPath: "roms/cgb-acid-hell/cgb-acid-hell.png",
		name:    "cgb-acid-hell",
		model:   gameboy.ModelCGB,
	})
}
