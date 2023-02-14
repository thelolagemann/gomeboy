package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"testing"
)

func testAcid2(t *testing.T, table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("acid2")

	// create test collection
	tc := tS.NewTestCollection("dmg-acid2")

	// add tests
	tc.Add(&genericImageTest{
		romPath:       "roms/dmg-acid2/dmg-acid2.gb",
		expectedImage: "roms/dmg-acid2/dmg-acid2-dmg.png",
		name:          "dmg-acid2",
		model:         gameboy.ModelDMG,
	})
	tc.Add(&genericImageTest{
		romPath:       "roms/dmg-acid2/dmg-acid2.gb",
		expectedImage: "roms/dmg-acid2/dmg-acid2-cgb.png",
		name:          "dmg-acid2-cgb",
		model:         gameboy.ModelCGB,
	})

	tc2 := tS.NewTestCollection("cgb-acid2")
	tc2.Add(&genericImageTest{
		romPath:       "roms/cgb-acid2/cgb-acid2.gbc",
		expectedImage: "roms/cgb-acid2/cgb-acid2.png",
		name:          "cgb-acid2",
		model:         gameboy.ModelCGB,
	})
	tc2.Add(&genericImageTest{
		romPath:       "roms/cgb-acid-hell/cgb-acid-hell.gbc",
		expectedImage: "roms/cgb-acid-hell/cgb-acid-hell.png",
		name:          "cgb-acid-hell",
		model:         gameboy.ModelCGB,
	})
}
