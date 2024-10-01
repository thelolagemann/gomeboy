package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

func littleThings() []ROMTest {
	return append(
		imageTestForModels("firstwhite", 1, types.DMGABC, types.CGBABC),
		&inputTest{
			basicTest: &basicTest{
				name:    "tellinglys (DMG)",
				romPath: "roms/little-things-gb/tellinglys.gb",
				model:   types.DMGABC,
			},
			expectedImagePath: "roms/little-things-gb/tellinglys-dmg.png",
		},
		&inputTest{
			basicTest: &basicTest{
				name:    "tellinglys (CGB)",
				romPath: "roms/little-things-gb/tellinglys.gb",
				model:   types.CGBABC,
			},
			expectedImagePath: "roms/little-things-gb/tellinglys-cgb.png",
		},
	)
}

func Test_LittleThings(t *testing.T) {
	testROMs(t, littleThings()...)
}

func testLittleThings(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("little-things-gb")

	te := littleThings()
	// firstwhite
	tS.NewTestCollection("firstwhite").AddTests(te[:2]...)
	// tellinglys
	tS.NewTestCollection("tellinglys").AddTests(te[2:]...)
}
