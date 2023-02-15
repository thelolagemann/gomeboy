package tests

import (
	"bytes"
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"testing"
)

const readmeBlurb = `<hr/>
GomeBoy is automatically tested against the following test suites:

* **[Blargg's test roms](https://github.com/retrio/gb-test-roms)**  
  <sup>by [Shay Green (a.k.a. Blargg)](http://www.slack.net/~ant/) </sup>
* **[Bully](https://github.com/Hacktix/BullyGB)**
  and **[Strikethrough](https://github.com/Hacktix/strikethrough.gb)**  
  <sup>by [Hacktix](https://github.com/Hacktix) </sup>
* **[cgb-acid-hell](https://github.com/mattcurrie/cgb-acid-hell)**,
  **[cgb-acid2](https://github.com/mattcurrie/cgb-acid2)** and
  **[dmg-acid2](https://github.com/mattcurrie/dmg-acid2)**  
  <sup>by [Matt Currie](https://github.com/mattcurrie) </sup>
* **[(parts of) little-things-gb](https://github.com/pinobatch/little-things-gb)**  
  <sup>by [Damian Yerrick](https://github.com/pinobatch) </sup>
* **[Mooneye Test Suite](https://github.com/Gekkio/mooneye-test-suite)**  
  <sup>by [Joonas Javanainen](https://github.com/Gekkio) </sup>
* **[SameSuite](https://github.com/LIJI32/SameSuite)**  
  <sup>by [Lior Halphon](https://github.com/LIJI32) </sup>

Different test suites use different pass/fail criteria. Some may write output to the serial port such as
[Blargg's test roms](https://github.com/retrio/gb-test-roms), others may write to the CPU registers, such as 
[Mooneye Test Suite](https://github.com/Gekkio/mooneye-test-suite) and [SameSuite](https://github.com/LIJI32/SameSuite).
If the test suite does not provide a way to automatically determine a pass/fail criteria, then the emulator's output
is compared against a reference image from a known good emulator.
<hr/>

`

func Test_All(t *testing.T) {
	testTable := &TestTable{
		testSuites: make([]*TestSuite, 0),
	}
	testAcid2(testTable)
	testBully(testTable)
	testBlarrg(t, testTable)
	testLittleThings(testTable)
	testMooneye(t, testTable)
	testSamesuite(t, testTable)
	testStrikethrough(testTable)

	// execute tests
	for _, top := range testTable.testSuites {
		t.Run(top.name, func(t *testing.T) {
			for _, collection := range top.collections {
				t.Run(collection.name, func(t *testing.T) {
					collection.Run(t)
				})
			}
		})
	}

	// wait for all tests to finish

	// write markdown table to README.md
	f, err := os.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.WriteString(testTable.CreateReadme())
}

func ImgCompare(img1, img2 image.Image) (int64, image.Image, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if bounds1 != bounds2 {
		return math.MaxInt64, nil, fmt.Errorf("image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
	}

	accumError := int64(0)
	resultImg := image.NewNRGBA(image.Rect(
		bounds1.Min.X,
		bounds1.Min.Y,
		bounds1.Max.X,
		bounds1.Max.Y,
	))
	draw.Draw(resultImg, resultImg.Bounds(), img1, image.Point{0, 0}, draw.Src)

	for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
		for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
			r1, g1, b1, a1 := img1.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()

			diff := int64(sqDiffUInt32(r1, r2))
			diff += int64(sqDiffUInt32(g1, g2))
			diff += int64(sqDiffUInt32(b1, b2))
			diff += int64(sqDiffUInt32(a1, a2))

			if diff > 0 {
				accumError += diff
				resultImg.Set(
					bounds1.Min.X+x,
					bounds1.Min.Y+y,
					color.NRGBA{R: 255, A: 128})
			}
		}
	}

	return int64(math.Sqrt(float64(accumError))), resultImg, nil
}

func sqDiffUInt32(x, y uint32) uint64 {
	d := uint64(x) - uint64(y)
	return d * d
}

// TestTable is a collection of many TestSuite(s).
type TestTable struct {
	// Top level tests
	testSuites []*TestSuite
}

func createProgressBar(suite *TestSuite) string {
	total := 0
	passed := 0
	for _, collection := range suite.AllCollections() {
		for _, test := range collection.tests {
			total++
			if test.Passed() {
				passed++
			}
		}
	}

	passRate := float64(passed) / float64(total)

	progressBar := fmt.Sprintf(
		"![progress](https://progress-bar.dev/%s/?scale=100&title=passing%%20%s,%%20failing%%20%s&width=500)",
		fmt.Sprintf("%d", int(passRate*100)),
		fmt.Sprintf("%d", passed),
		fmt.Sprintf("%d", total-passed))

	return progressBar
}

func (t *TestTable) CreateReadme() string {
	// create the table of contents with links
	tableOfContents := "# Test Results\n"
	// create table of global results
	tableOfContents += "| Test Suite | Pass Rate | Tests Passed | Tests Failed | Tests Total |\n| --- | --- | --- | --- | --- |"
	for _, suite := range t.testSuites {
		tableOfContents += suite.CreateTableEntry()
	}
	tableOfContents += "\n\nExplore the individual tests for each suite using the table of contents below.\n\n## Table of Contents\n"
	for _, suite := range t.testSuites {
		tableOfContents += "* [" + suite.name + "](#" + suite.name + ")\n"
		for _, collection := range suite.collections {
			tableOfContents += "  * [" + collection.name + "](#" + collection.name + ")\n"
			// check for subcollections
			for _, sub := range collection.subCollections {
				tableOfContents += "    * [" + sub.name + "](#" + sub.name + ")\n"
			}
		}
	}

	// create a progress bar for overall test pass rate
	passed := 0
	total := 0
	for _, suite := range t.testSuites {
		for _, collection := range suite.AllCollections() {
			// TODO get all ROM tests including sub-collections for correct total
			for _, test := range collection.tests {
				total++
				if test.Passed() {
					passed++
				}
			}
		}
	}
	passRate := float64(passed) / float64(total)

	progressBar := fmt.Sprintf(
		"![progress](https://progress-bar.dev/%s/?scale=100&title=passing%%20%s,%%20failing%%20%s&width=500)",
		fmt.Sprintf("%d", int(passRate*100)),
		fmt.Sprintf("%d", passed),
		fmt.Sprintf("%d", total-passed))

	// create the test results
	table := ""
	for _, suite := range t.testSuites {
		table += "# " + suite.name + "\n"
		table += createProgressBar(suite) + "\n"
		for _, collection := range suite.AllCollections() {
			table += "## " + collection.name + "\n"
			table += CreateMarkdownTableFromTests(collection.tests)
		}
	}

	// create document timestamp and commit hash
	commitHash := "unknown"
	if commitHashBytes, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		// get the first 8 characters of the commit hash
		commitHash = string(commitHashBytes[:8])
	}

	// create formatted timestamp
	timeStr := fmt.Sprintf("#### This document was automatically generated from commit %s\n", commitHash)
	return `# Automated test results
` + progressBar + "\n\n" + timeStr + readmeBlurb + "\n" + tableOfContents + "\n" + table
}

// TestSuite is a collection of tests (often by a single author, or for a single
// feature) that can be run together.
type TestSuite struct {
	name        string
	collections []*TestCollection
}

func (t *TestSuite) AllCollections() []*TestCollection {
	tests := []*TestCollection{}
	for _, collection := range t.collections {
		tests = append(tests, collection)

		for _, subCollection := range collection.subCollections {
			// TODO recursively get all sub-collections
			tests = append(tests, subCollection)

		}
	}

	return tests
}

func (t *TestSuite) NewTestCollection(name string) *TestCollection {
	collection := &TestCollection{name: name, tests: make([]ROMTest, 0)}
	t.collections = append(t.collections, collection)
	return collection
}

func (t *TestSuite) CreateTableEntry() string {
	total := 0
	passed := 0
	for _, collection := range t.AllCollections() {
		for _, test := range collection.tests {
			total++
			if test.Passed() {
				passed++
			}
		}
	}

	passRate := float64(passed) / float64(total)

	return fmt.Sprintf("\n| %s | %s | %d | %d | %d |", t.name, fmt.Sprintf("%d%%", int(passRate*100)), passed, total-passed, total)
}

func (t *TestTable) NewTestSuite(name string) *TestSuite {
	suite := &TestSuite{name: name, collections: make([]*TestCollection, 0)}
	t.testSuites = append(t.testSuites, suite)
	return suite
}

type TestCollection struct {
	tests          []ROMTest
	name           string
	subCollections []*TestCollection
}

func (tC *TestCollection) Add(test ROMTest) {
	tC.tests = append(tC.tests, test)
}

func (tC *TestCollection) AddTests(tests ...ROMTest) {
	for _, test := range tests {
		tC.tests = append(tC.tests, test)
	}
}

func (tC *TestCollection) AllTests() []ROMTest {
	tests := []ROMTest{}
	for _, test := range tC.tests {
		tests = append(tests, test)
	}
	for _, subCollection := range tC.subCollections {
		// handle recursive sub collections
		for _, subTest := range subCollection.AllTests() {
			tests = append(tests, subTest)
		}
	}
	return tests
}

// Run runs all the tests in the collection, including any tests in sub-collections.
func (tC *TestCollection) Run(t *testing.T) {
	for _, test := range tC.tests {
		test.Run(t)
	}
	for _, subCollection := range tC.subCollections {
		t.Run(subCollection.name, func(t *testing.T) {
			subCollection.Run(t)
		})
	}
}

func (tC *TestCollection) NewTestCollection(name string) *TestCollection {
	collection := &TestCollection{name: name, tests: make([]ROMTest, 0)}
	tC.subCollections = append(tC.subCollections, collection)
	return collection
}

func (tC *TestCollection) AddTestCollection(dir *TestCollection) {
	tC.subCollections = append(tC.subCollections, dir)
}

type ROMTest interface {
	Run(t *testing.T)
	Passed() bool
	Name() string
}

func CreateMarkdownTableFromTests(tests []ROMTest) string {
	table := "| Test | Passing |\n| ---- | ------- |\n"
	for _, test := range tests {
		// pass is green check, fail is red x
		pass := "✅"
		if !test.Passed() {
			pass = "❌"
		}
		table += "| " + test.Name() + " | " + pass + " |\n"
		fmt.Println(test.Name(), test.Passed())
	}
	return table
}

func testROMWithExpectedImage(t *testing.T, romPath string, expectedImagePath string, asModel gameboy.Model, emulatedSeconds int, name string) bool {
	passed := true
	t.Run(name, func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romPath)
		if err != nil {
			t.Fatal(err)
		}

		// create the emulator
		g := gameboy.NewGameBoy(b, gameboy.AsModel(asModel))

		// custom test loop
		for frame := 0; frame < 60*emulatedSeconds; frame++ {
			for i := uint32(0); i < gameboy.TicksPerFrame; {
				i += uint32(g.CPU.Step())
			}

			// wait until frame is done
			for !g.PPU.HasFrame() {
				g.CPU.Step()
			}
			g.PPU.ClearRefresh()
		}

		img := g.PPU.PreparedFrame

		// create image.Image from the byte array
		img1 := image.NewNRGBA(image.Rect(0, 0, 160, 144))
		palette := []color.Color{}
		for y := 0; y < 144; y++ {
		next:
			for x := 0; x < 160; x++ {
				if asModel == gameboy.ModelDMG {
					col := color.NRGBA{
						R: img[y][x][0],
						G: img[y][x][1],
						B: img[y][x][2],
						A: 255,
					}
					img1.Set(x, y, col)
					// add color if it doesn't exist
					for _, p := range palette {
						r, g, b, _ := p.RGBA()
						r2, g2, b2, _ := col.RGBA()
						if r == r2 && g == g2 && b == b2 {
							continue next
						}
					}
					palette = append(palette, col)
				} else {
					// cgb 5-bit channel is converted
					// to 8-bit channel with the formula (x << 3) | (x >> 2)
					r := img[y][x][0]
					g := img[y][x][1]
					b := img[y][x][2]
					col := color.NRGBA{
						R: r,
						G: g,
						B: b,
						A: 255,
					}
					img1.Set(x, y, col)
					for _, p := range palette {
						r, g, b, _ := p.RGBA()
						r2, g2, b2, _ := col.RGBA()
						if r == r2 && g == g2 && b == b2 {
							continue next
						}
					}
					palette = append(palette, col)
				}
			}
		}

		// compare the image to the expected image
		expectedImg, err := os.ReadFile(expectedImagePath)
		if err != nil {
			t.Fatal(err)
		}
		img2, _, err := image.Decode(bytes.NewReader(expectedImg))
		// create a new paletted image
		img3 := image.NewPaletted(img1.Bounds(), palette)
		draw.Draw(img3, img3.Bounds(), img1, image.Point{0, 0}, draw.Src)

		// compare the images
		diff, diffResult, err := ImgCompare(img2, img3)
		if err != nil {
			t.Fatal(err)
		}

		if diff > 0 {
			passed = false
			t.Errorf("Test %s failed. Difference: %d", name, diff)
			// save the diff image
			f, err := os.Create("results/" + name + ".png")
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, diffResult); err != nil {
				t.Fatal(err)
			}

			// save the actual image
			f, err = os.Create("results/" + name + "_actual.png")
			if err != nil {
				t.Fatal(err)
			}

			if err = png.Encode(f, img3); err != nil {
				t.Fatal(err)
			}

			// save the expected image
			f, err = os.Create("results/" + name + "_expected.png")
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, img2); err != nil {
				t.Fatal(err)
			}
		}
	})
	return passed
}

func testROMs(t *testing.T, roms ...ROMTest) {
	for _, rom := range roms {
		rom.Run(t)
	}
}

// TODO:
// - add a way to run tests in parallel
// - parse description from test roms (maybe)
// - model differentiation (DMG, CGB, SGB)
// - expected image output with actual image in README (with overlay)
// - git commit hook to run tests and update README
// - git clone to download test roms
// - blurb for each test suite
// - tests have table entries for each test, with a link to the test rom, and a link to the expected image
// - refactor tests package out of internal and into root
// - palette compatibility dump
// - individual test run
// - check if test suite has only 1 test (to avoid double header)
// - individual test suite table generation
// - add test that can simulate a button press
// - gameboy doctor
// - jsmoo tests
// - wilbertpol's tests
