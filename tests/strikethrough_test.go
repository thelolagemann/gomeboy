package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"testing"
)

var (
	strikethroughTests = imageTestForModels("strikethrough", 1, types.DMGABC, types.CGBABC)
)

func Test_Strikethrough(t *testing.T) {
	testROMs(t, strikethroughTests...)
}

func testStrikethrough(t *TestTable) {
	t.NewTestSuite("strikethrough").
		NewTestCollection("strikethrough").
		AddTests(strikethroughTests...)
}
