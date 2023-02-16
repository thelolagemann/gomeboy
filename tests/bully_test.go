package tests

import (
	"testing"
)

var (
	// bullyTest is a test for the bully rom
	bullyTest = newImageTest("bully", withEmulatedSeconds(5))
)

func Test_Bully(t *testing.T) {
	bullyTest.Run(t)
}

func testBully(table *TestTable) {
	// create top level test
	table.NewTestSuite("bully").
		NewTestCollection("bully").
		Add(bullyTest)
}
