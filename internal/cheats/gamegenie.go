package cheats

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type GameGenie struct {
	// Three codes can be loaded at once
	Codes []GameGenieCode
}

// A GameGenieCode consists of nine-digit hex numbers, formatted as
// ABC-DEF-GHI. AB is the new data, FCDE is the memory address XORed
// by 0xF000, GI is the old data XORed by 0xBA and rotated left by 2,
// and H is unknown (possibly a checksum).
type GameGenieCode struct {
	NewData uint8
	Address uint16
	OldData uint8

	// Unknown

	Name    string // name provided by the user
	Enabled bool   // disabled by default
	rawCode string // raw code provided by the user
}

func parseCode(code string) (GameGenieCode, error) {
	// assert correct length
	if len(code) != 11 {
		return GameGenieCode{}, fmt.Errorf("invalid code length: %v", len(code))
	}

	var c GameGenieCode

	// remove the hyphens, and interpret the code
	code = strings.Replace(code, "-", "", -1)

	// get the individual hex codes (AB, FCDE, GI)
	hexCodes := []string{
		code[0:2],
		code[2:6],
		code[6:7] + code[8:9],
	}

	// reorganize CDEF to FCDE
	hexCodes[1] = hexCodes[1][3:4] + hexCodes[1][0:3]

	// parse the hex codes
	hexCodeAB, err := strconv.ParseUint(hexCodes[0], 16, 8)
	if err != nil {
		return c, err
	}
	// set the new data (AB)
	c.NewData = uint8(hexCodeAB & 0xFF)

	hexCodeFCDE, err := strconv.ParseUint(hexCodes[1], 16, 16)
	if err != nil {
		return c, err
	}

	// set the address (FCDE)
	c.Address = uint16(hexCodeFCDE&0xFFFF) ^ 0xF000

	hexCodeGI, err := strconv.ParseUint(hexCodes[2], 16, 8)
	if err != nil {
		return c, err
	}

	// set the old data (GI)
	c.OldData = uint8(hexCodeGI) ^ 0xBA
	c.OldData <<= 2

	// set the unknown data (H)
	// c.Unknown = uint8(hexCode & 0xFF)

	return c, nil
}

// NewGameGenie creates a new GameGenie.
func NewGameGenie() *GameGenie {
	return &GameGenie{}
}

// Load loads the given GameGenie code into the GameGenie.
func (g *GameGenie) Load(code, name string) error {
	// parse the code
	c, err := parseCode(code)
	if err != nil {
		return err
	}

	// set the raw and name
	c.rawCode = code
	c.Name = name

	// add the code to the GameGenie
	g.Codes = append(g.Codes, c)

	fmt.Printf("Parsed Game Genie Code: %s -> New Data: %02X, Address: %04X, Old Data: %02X\n", code, c.NewData, c.Address, c.OldData)

	// TODO emulate the game genie 3 code limit (with option to disable)

	return nil
}

func (g *GameGenie) Cheat(address uint16) bool {
	for _, c := range g.Codes {
		if c.Address == address && c.Enabled {
			return true
		}
	}

	return false
}

func (g *GameGenie) Read(address uint16, oldValue uint8) uint8 {
	for _, c := range g.Codes {
		if c.Address == address {
			if c.OldData != oldValue {
				// TODO fix this
				//return oldValue
			}
			return c.NewData
		}
	}

	return oldValue
}

// Save saves the GameGenie codes to the given file.
func (g *GameGenie) Save(file string) error {
	// open the file
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// write the codes
	for _, c := range g.Codes {
		_, err := f.WriteString(fmt.Sprintf("%s %s\n", c.rawCode, c.Name))
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadFile loads the GameGenie codes from the given file.
func (g *GameGenie) LoadFile(file string) error {
	// open the file
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	// read the file
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		// get the line
		line := scanner.Text()

		// split the line
		split := strings.Split(line, " ")

		// parse the code
		err := g.Load(split[0], strings.Join(split[1:], " "))
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GameGenie) Enable(name string) {
	for i := range g.Codes {
		if g.Codes[i].Name == name {
			g.Codes[i].Enabled = true
		}
	}
}

func (g *GameGenie) Disable(name string) {
	for i := range g.Codes {
		if g.Codes[i].Name == name {
			g.Codes[i].Enabled = false
		}
	}
}
