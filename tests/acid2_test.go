package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
	"testing"
)

var (
	dmgAcid2Tests = func() []ROMTest {
		return []ROMTest{
			newImageTest("dmg-acid2"),
			newImageTest("dmg-acid2", asModel(types.CGBABC)),
		}
	}
	cgbAcid2Tests = func() []ROMTest {
		return []ROMTest{
			newImageTest("cgb-acid2", asModel(types.CGBABC)),
			newImageTest("cgb-acid-hell", asModel(types.CGBABC)),
		}
	}
)

func Test_Acid2(t *testing.T) {
	testROMs(t, dmgAcid2Tests()...)
	testROMs(t, cgbAcid2Tests()...)
}

func testAcid2(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("acid2")

	// create test collection
	tc := tS.NewTestCollection("dmg-acid2")
	tc.AddTests(dmgAcid2Tests()...)

	tc2 := tS.NewTestCollection("cgb-acid2")
	tc2.AddTests(cgbAcid2Tests()...)
}
