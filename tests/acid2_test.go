package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

func acid2() []ROMTest {
	return append(
		imageTestForModels("dmg-acid2", 1, types.DMGABC, types.CGBABC),
		newImageTest("cgb-acid2", asModel(types.CGBABC)),
		newImageTest("cgb-acid-hell", asModel(types.CGBABC)),
	)
}

func Test_Acid2(t *testing.T) {
	testROMs(t, acid2()...)
}

func testAcid2(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("acid2")

	t := acid2()
	// add test collections
	tS.NewTestCollection("dmg-acid2").AddTests(t[0:2]...)
	tS.NewTestCollection("cgb-acid2").AddTests(t[2:]...)
}
