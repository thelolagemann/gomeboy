package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

func strikethrough() []ROMTest {
	return imageTestForModels("strikethrough", 1, types.DMGABC, types.CGBABC)
}

func Test_Strikethrough(t *testing.T) {
	testROMs(t, strikethrough()...)
}

func testStrikethrough(t *TestTable) {
	t.NewTestSuite("strikethrough").
		NewTestCollection("strikethrough").
		AddTests(strikethrough()...)
}
