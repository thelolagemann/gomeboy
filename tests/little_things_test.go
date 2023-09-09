package tests

import (
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

const perCycle = 70224 * 30 // 70224 cycles per frame, 60 frames per second

func littleThings() []ROMTest {
	return []ROMTest{
		newImageTest("firstwhite", asModel(types.DMGABC)),
		newImageTest("firstwhite", asModel(types.CGBABC)),
		&inputTest{
			name:              "tellinglys",
			romPath:           "roms/little-things-gb/tellinglys.gb",
			expectedImagePath: "roms/little-things-gb/tellinglys-dmg.png",
			model:             types.DMGABC,
			inputs:            tellingLysInputSequence,
		},
		&inputTest{
			name:              "tellinglys-cgb",
			romPath:           "roms/little-things-gb/tellinglys.gb",
			expectedImagePath: "roms/little-things-gb/tellinglys-cgb.png",
			model:             types.CGBABC,
			inputs:            tellingLysInputSequence,
		},
	}
}

var (
	tellingLysInputSequence = []testInput{
		{button: joypad.ButtonA, atEmulatedCycle: perCycle * 1},
		{button: joypad.ButtonB, atEmulatedCycle: perCycle * 1.5},
		{button: joypad.ButtonSelect, atEmulatedCycle: perCycle * 2},
		{button: joypad.ButtonStart, atEmulatedCycle: perCycle * 2.5},
		{button: joypad.ButtonRight, atEmulatedCycle: perCycle * 3},
		{button: joypad.ButtonLeft, atEmulatedCycle: perCycle * 3.5},
		{button: joypad.ButtonUp, atEmulatedCycle: perCycle * 4},
		{button: joypad.ButtonDown, atEmulatedCycle: perCycle * 4.5},
	}
)

func Test_LittleThings(t *testing.T) {
	testROMs(t, littleThings()...)
}

func testLittleThings(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("little-things-gb")

	// firstwhite
	tS.NewTestCollection("firstwhite").AddTests(littleThings()[:2]...)
	// tellinglys
	tS.NewTestCollection("tellinglys").AddTests(littleThings()[2:]...)
}
