package tests

import (
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"testing"
)

var (
	scribblPalette = palette.Palette{
		{0xe0, 0xf8, 0xd0},
		{0x88, 0xc0, 0x70},
		{0x34, 0x68, 0x56},
		{0x08, 0x18, 0x20},
	}

	scribblTests = []ROMTest{
		newImageTest("scribbl/lycscx", withPalette(scribblPalette)),
		newImageTest("scribbl/lycscy", withPalette(scribblPalette)),
		newImageTest("scribbl/palettely", withPalette(scribblPalette)),
		newImageTest("scribbl/scxly", withPalette(scribblPalette)),
		newImageTest("scribbl/statcount", withPalette(scribblPalette), withEmulatedSeconds(6)),
	}
)

func Test_Scribbl(t *testing.T) {
	testROMs(t, scribblTests...)
}

func testScribbl(t *TestTable) {
	t.NewTestSuite("scribbltests").NewTestCollection("scribbltests").AddTests(scribblTests...)
}
