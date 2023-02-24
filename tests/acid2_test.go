package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"testing"
)

var (
	dmgAcid2Tests = []ROMTest{
		newImageTest("dmg-acid2"),
		newImageTest("dmg-acid2", asModel(gameboy.ModelCGB)),
	}
	cgbAcid2Tests = []ROMTest{
		newImageTest("cgb-acid2", asModel(gameboy.ModelCGB)),
		newImageTest("cgb-acid-hell", asModel(gameboy.ModelCGB)),
	}
)

func Test_Acid2(t *testing.T) {
	testROMs(t, dmgAcid2Tests...)
	testROMs(t, cgbAcid2Tests...)
}

func testAcid2(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("acid2")

	// create test collection
	tc := tS.NewTestCollection("dmg-acid2")
	tc.AddTests(dmgAcid2Tests...)

	tc2 := tS.NewTestCollection("cgb-acid2")
	tc2.AddTests(cgbAcid2Tests...)
}