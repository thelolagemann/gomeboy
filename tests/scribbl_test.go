package tests

import "testing"

func scribblTests() []ROMTest {
	return []ROMTest{
		newImageTest("scribbl/lycscx"),
		newImageTest("scribbl/lycscy"),
		newImageTest("scribbl/palettely"),
		newImageTest("scribbl/scxly"),
		newImageTest("scribbl/statcount"),
		// newImageTest("scribbl/winpos"), TODO: make gif test
	}
}

func Test_Scribbl(t *testing.T) {
	testROMs(t, scribblTests()...)
}
