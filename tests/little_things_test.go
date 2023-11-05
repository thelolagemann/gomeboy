package tests

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

const perCycle = 70224 * 30 // 70224 cycles per frame, 60 frames per second

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
			inputs:            tellingLysInputSequence,
		},
		&inputTest{
			basicTest: &basicTest{
				name:    "tellinglys (CGB)",
				romPath: "roms/little-things-gb/tellinglys.gb",
				model:   types.CGBABC,
			},
			expectedImagePath: "roms/little-things-gb/tellinglys-cgb.png",
			inputs:            tellingLysInputSequence,
		},
	)
}

var (
	tellingLysInputSequence = []testInput{
		{button: io.ButtonA, atEmulatedCycle: perCycle * 1},
		{button: io.ButtonB, atEmulatedCycle: perCycle * 1.5},
		{button: io.ButtonSelect, atEmulatedCycle: perCycle * 2},
		{button: io.ButtonStart, atEmulatedCycle: perCycle * 2.5},
		{button: io.ButtonRight, atEmulatedCycle: perCycle * 3},
		{button: io.ButtonLeft, atEmulatedCycle: perCycle * 3.5},
		{button: io.ButtonUp, atEmulatedCycle: perCycle * 4},
		{button: io.ButtonDown, atEmulatedCycle: perCycle * 4.5},
	}
)

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
