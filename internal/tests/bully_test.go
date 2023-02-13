package tests

import "github.com/thelolagemann/go-gameboy/internal/gameboy"

func testBully(table *TestTable) {
	// create top level test
	tS := table.NewTestSuite("bully")

	// bully
	tS.NewTestCollection("bully").Add(&genericImageTest{
		romPath:         "roms/bully/bully.gb",
		name:            "bully",
		expectedImage:   "roms/bully/bully.png",
		emulatedSeconds: 5,
		model:           gameboy.ModelDMG,
	})
}
