package tests

import "github.com/thelolagemann/go-gameboy/internal/gameboy"

func testLittleThings(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("little-things-gb")

	// firstwhite
	tS.NewTestCollection("firstwhite").Add(&genericImageTest{
		romPath:         "roms/little-things-gb/firstwhite.gb",
		name:            "firstwhite",
		expectedImage:   "roms/little-things-gb/firstwhite-dmg-cgb.png",
		emulatedSeconds: 5,
		model:           gameboy.ModelDMG,
	})
}
