package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

var (
	dmgAcid2Tests = imageTestForModels("dmg-acid2", 1, types.DMGABC, types.CGBABC)
	cgbAcid2Tests = []ROMTest{
		newImageTest("cgb-acid2", asModel(types.CGBABC)),
		newImageTest("cgb-acid-hell", asModel(types.CGBABC)),
	}
)

func Test_Acid2(t *testing.T) {
	testROMs(t, dmgAcid2Tests...)
	testROMs(t, cgbAcid2Tests...)
}

func testAcid2(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("acid2")

	// add test collections
	tS.NewTestCollection("dmg-acid2").AddTests(dmgAcid2Tests...)
	tS.NewTestCollection("cgb-acid2").AddTests(cgbAcid2Tests...)
}
