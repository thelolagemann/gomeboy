package tests

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	ageROMPath = "roms/age"
)

func newAgeTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(ageROMPath, dir)
	tc := suite.NewTestCollection(dir)

	// read the directory
	files, err := os.ReadDir(romDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".gb" {
			continue
		}

		tc.Add(&mooneyeTest{
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
		})
	}

	return tc
}

func testAge(t *testing.T, table *TestTable) {
	// create top level test
	tS := table.NewTestSuite("age")

	// halt
	newAgeTestCollectionFromDir(tS, "halt")
}
