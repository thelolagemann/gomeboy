package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

var (
	// bullyTest is a test for the bully rom
	bullyTests = []ROMTest{
		newImageTest("bully", withEmulatedSeconds(5)),
		newImageTest("bully", withEmulatedSeconds(5), asModel(types.CGBABC)),
	}
)

func Test_Bully(t *testing.T) {
	testROMs(t, bullyTests...)
}

func testBully(table *TestTable) {
	// create top level test
	table.NewTestSuite("bully").
		NewTestCollection("bully").
		AddTests(bullyTests...)
}
