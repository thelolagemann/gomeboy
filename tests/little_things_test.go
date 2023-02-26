package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"testing"
)

var (
	tellingLysInputSequence = []testInput{
		{button: joypad.ButtonA, atEmulatedFrame: 60 * 0.5},
		{button: joypad.ButtonB, atEmulatedFrame: 60 * 1},
		{button: joypad.ButtonStart, atEmulatedFrame: 60 * 1.5},
		{button: joypad.ButtonSelect, atEmulatedFrame: 60 * 2},
		{button: joypad.ButtonRight, atEmulatedFrame: 60 * 2.5},
		{button: joypad.ButtonLeft, atEmulatedFrame: 60 * 3},
		{button: joypad.ButtonUp, atEmulatedFrame: 60 * 3.5},
		{button: joypad.ButtonDown, atEmulatedFrame: 60 * 4},
	}
	littleThingsTests = []ROMTest{
		newImageTest("firstwhite", asModel(types.DMGABC)),
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
)

func Test_LittleThings(t *testing.T) {
	testROMs(t, littleThingsTests...)
}

func testLittleThings(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("little-things-gb")

	// firstwhite
	tS.NewTestCollection("firstwhite").Add(littleThingsTests[0])
	// tellinglys
	tS.NewTestCollection("tellinglys").AddTests(littleThingsTests[1:]...)
}
