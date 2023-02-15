package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"testing"
)

var (
	strikethroughTests = []ROMTest{
		newImageTest("strikethrough"),
		newImageTest("strikethrough", asModel(gameboy.ModelCGB)),
	}
)

func Test_Strikethrough(t *testing.T) {
	testROMs(t, strikethroughTests...)
}

func testStrikethrough(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("strikethrough")

	// strikethrough
	strikethrough := tS.NewTestCollection("strikethrough")
	strikethrough.AddTests(strikethroughTests...)
}
