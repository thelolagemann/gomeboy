package tests

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

var (
	_, b, _, _ = runtime.Caller(0)
	basePath   = filepath.Dir(b)
	tableRE    = regexp.MustCompile(`\| ([a-zA-Z0-9-]+) \| ([0-9]+%) \| ([0-9]+) \| ([0-9]+) \| ([0-9]+) \|`)
)

func Test_All(t *testing.T) {
	testTable := testAllTable()

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

	// write markdown table to README.md
	f, err := os.Create("README.md")
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString(testTable.CreateReadme())

	if err != nil {
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}
}

func testAllTable() *TestTable {
	testTable := &TestTable{
		testSuites: make([]*TestSuite, 0),
	}
	testAcid2(testTable)
	// testAge(testTable)
	testBully(testTable)
	testBlarrg(testTable)
	testLittleThings(testTable)
	testMooneye(testTable)
	testSamesuite(testTable)
	testStrikethrough(testTable)

	return testTable
}

type regressionTests map[string]int

func Test_Regressions(t *testing.T) {
	// load README from main branch
	req, err := http.Get("https://raw.githubusercontent.com/thelolagemann/gomeboy/main/tests/README.md")
	if err != nil {
		t.Error(err)
	}
	defer req.Body.Close()

	// read bytes
	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Error(err)
	}

	currentTests := parseTable(string(b))

	// jump to basepath
	if err := os.Chdir(basePath); err != nil {
		t.Error(err)
	}

	// run test with exec (cheeky hack to avoid exit status 1 on failure)
	cmd := exec.Command("go", "test", "-v",
		"acid2_test.go",
		"age_test.go",
		"blarrg_test.go",
		"bully_test.go",
		"image_test.go",
		"input_test.go",
		"little_things_test.go",
		"mealybug_test.go",
		"misc_test.go",
		"mooneye_test.go",
		"rom_test.go",
		"samesuite_test.go",
		"scribbl_test.go",
		"strikethrough_test.go",
		"wilbertpol_test.go",
		"-run", "Test_All")
	var exitError *exec.ExitError
	if err := cmd.Run(); errors.As(err, &exitError) {
		if exitError.ExitCode() > 1 {
			t.Error(err)
		}
	} else {
		t.Error(err)
	}

	// load local README for comparison
	f, err := os.Open("README.md")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	newB, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}

	// READMEs should be different based on the commit hash, so just check byte equality
	if bytes.Equal(b, newB) {
		t.Error("no changes detected in README file")
	}

	newTests := parseTable(string(newB))

	// check that each test suite either passes the same number, or a greater number of tests (TODO per test specificity)
	for suite, passed := range currentTests {
		t.Run(suite, func(t *testing.T) {
			if newTests[suite] < passed {
				t.Errorf("%s has a regression, %d -> %d", suite, passed, newTests[suite])
			}
		})
	}

}

func parseTable(markdown string) regressionTests {
	matches := tableRE.FindAllStringSubmatch(markdown, -1)

	tests := make(regressionTests)

	for _, match := range matches {
		suite := match[1]
		passed, _ := strconv.Atoi(match[3])
		tests[suite] = passed
	}

	return tests
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
			if len(suite.AllCollections()) > 1 {
				table += "## " + collection.name + "\n"
			} else {
				table += "\n"
			}
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
	}
	return table
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
// - git commit hook to run tests and update README
// - git clone to download test roms
// - blurb for each test suite (maybe)
// - tests have table entries for each test, with a link to the test rom, and a link to the expected image
// - palette compatibility dump
// - expected image output with actual image in README (with overlay)
// - individual test run
// - individual test suite table generation (not sure what I meant by this)
// - gameboy doctor
// - jsmoo tests
// - wilbertpol's tests
// - age tests
// - rtc tests
// - mealybug tests
// - failure reasons
// - ROMTest with TableEntry interface (for tests that provide a custom table entry)
