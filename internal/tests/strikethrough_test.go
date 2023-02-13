package tests

import "github.com/thelolagemann/go-gameboy/internal/gameboy"

func testStrikethrough(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("strikethrough")

	// strikethrough
	strikethrough := tS.NewTestCollection("strikethrough")
	strikethrough.Add(&genericImageTest{
		romPath:         "roms/strikethrough/strikethrough.gb",
		name:            "strikethrough_dmg",
		expectedImage:   "roms/strikethrough/strikethrough-dmg.png",
		emulatedSeconds: 5,
		model:           gameboy.ModelDMG,
	})
	strikethrough.Add(&genericImageTest{
		romPath:         "roms/strikethrough/strikethrough.gb",
		name:            "strikethrough_cgb",
		expectedImage:   "roms/strikethrough/strikethrough-cgb.png",
		emulatedSeconds: 5,
		model:           gameboy.ModelCGB,
	})
}
