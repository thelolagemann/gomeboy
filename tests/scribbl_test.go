package tests

import (
	"testing"
)

func scribbl() []ROMTest {
	return []ROMTest{
		newImageTest("scribbl/lycscx"),
		newImageTest("scribbl/lycscy"),
		newImageTest("scribbl/palettely"),
		newImageTest("scribbl/scxly"),
		newImageTest("scribbl/statcount", withEmulatedSeconds(6)),
	}
}

func Test_Scribbl(t *testing.T) {
	testROMs(t, scribbl()...)
}

func testScribbl(t *TestTable) {
	t.NewTestSuite("scribbltests").NewTestCollection("scribbltests").AddTests(scribbl()...)
}
