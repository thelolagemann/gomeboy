package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

func bully() []ROMTest {
	return imageTestForModels("bully", 1, types.DMGABC, types.CGBABC)
}

func Test_Bully(t *testing.T) {
	testROMs(t, bully()...)
}

func testBully(table *TestTable) {
	// create top level test
	table.NewTestSuite("bully").
		NewTestCollection("bully").
		AddTests(bully()...)
}
