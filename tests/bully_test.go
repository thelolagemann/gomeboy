package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

var (
	bullyTests = imageTestForModels("bully", 1, types.DMGABC, types.CGBABC)
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
