package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"os"
	"path/filepath"
	"strings"
)

const (
	samesuiteROMPath = "roms/same-suite"
)

func newSamesuiteTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	// more or less the same as mooneye as it uses the same pass/fail mechanism
	romDir := filepath.Join(samesuiteROMPath, dir)
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

		t := &mooneyeTest{
			basicTest: &basicTest{
				romPath: filepath.Join(romDir, file.Name()),
				name:    strings.Split(file.Name(), ".")[0],
			},
			emulatedSeconds: 5,
		}
		if strings.Contains(dir, "apu/channel") || file.Name() == "blocking_bgpi_increase.gb" || strings.Contains(file.Name(), "dma") {
			t.model = types.CGBABC
		}
		tc.AddTests(t)
	}

	return tc
}

func testSamesuite(roms *TestTable) {
	// create top level test suite
	tS := roms.NewTestSuite("samesuite")

	// apu
	newSamesuiteTestCollectionFromDir(tS, "apu")
	newSamesuiteTestCollectionFromDir(tS, "apu/channel_1")
	newSamesuiteTestCollectionFromDir(tS, "apu/channel_2")
	newSamesuiteTestCollectionFromDir(tS, "apu/channel_3")
	newSamesuiteTestCollectionFromDir(tS, "apu/channel_4")

	// dma
	newSamesuiteTestCollectionFromDir(tS, "dma")

	// interrupt
	newSamesuiteTestCollectionFromDir(tS, "interrupt")

	// ppu
	newSamesuiteTestCollectionFromDir(tS, "ppu")

	// sgb
	newSamesuiteTestCollectionFromDir(tS, "sgb")
}
