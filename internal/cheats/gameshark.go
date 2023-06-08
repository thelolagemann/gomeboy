package cheats

import (
	"fmt"
	"strconv"
)

type GameShark struct {
	Codes []GameSharkCode
}

// A GameSharkCode consists of eight-digit hex numbers, formatted
// as ABCDEFGH. Where AB represents the external RAM bank, CD is
// the new data, and GHEF is the memory address.
type GameSharkCode struct {
	ExternalRAMBank uint8
	Address         uint16
	NewData         uint8

	Name string // name provided by the user

	Enabled bool // disabled by default

	rawCode string // raw code provided by the user
}

func parseGameSharkCode(code string) (GameSharkCode, error) {
	var c GameSharkCode

	// make sure the code is 8 characters long
	if len(code) != 8 {
		return c, fmt.Errorf("invalid code length: %v", len(code))
	}

	// get the individual hex codes (AB, CD, GHEF)
	hexCodes := []string{
		code[0:2],
		code[2:4],
		code[4:8],
	}

	// reorganize GHEF to EFGH
	hexCodes[2] = hexCodes[2][2:4] + hexCodes[2][0:2]

	// parse the hex codes
	hexCodeAB, err := strconv.ParseUint(hexCodes[0], 16, 8)
	if err != nil {
		return c, err
	}

	// set the external RAM bank (AB)
	c.ExternalRAMBank = uint8(hexCodeAB & 0xFF)

	hexCodeCD, err := strconv.ParseUint(hexCodes[1], 16, 8)
	if err != nil {
		return c, err
	}

	// set the new data (CD)
	c.NewData = uint8(hexCodeCD & 0xFF)

	hexCodeEFGH, err := strconv.ParseUint(hexCodes[2], 16, 16)
	if err != nil {
		return c, err
	}

	// set the address (EFGH)
	c.Address = uint16(hexCodeEFGH & 0xFFFF)

	return c, nil
}

// NewGameShark creates a new GameShark.
func NewGameShark() *GameShark {
	return &GameShark{}
}

// Load loads a GameShark code.
func (g *GameShark) Load(code string, name string) error {
	// is the code already loaded?
	for i := range g.Codes {
		if g.Codes[i].Name == name {
			return fmt.Errorf("code already loaded: %s", name)
		}
	}

	c, err := parseGameSharkCode(code)
	if err != nil {
		return fmt.Errorf("failed to parse GameShark code: %v", err)
	}

	if c.Address >= 0xA000 && c.Address <= 0xBFFF {
		panic("Cartride RAM patching unimplemented")
	}

	fmt.Printf("Parse GameShark Code: %v -> NewData: %v, Address: %x, ExternalRAMBank: %v\n", code, c.NewData, c.Address, c.ExternalRAMBank)

	c.Name = name
	c.rawCode = code
	g.Codes = append(g.Codes, c)
	return nil
}

// Enable enables the given GameShark code.
func (g *GameShark) Enable(name string) error {
	for i := range g.Codes {
		if g.Codes[i].Name == name {
			g.Codes[i].Enabled = true
			return nil
		}
	}

	return fmt.Errorf("code not found: %s", name)
}

// Disable disables the given GameShark code.
func (g *GameShark) Disable(name string) error {
	for i := range g.Codes {
		if g.Codes[i].Name == name {
			g.Codes[i].Enabled = false
			return nil
		}
	}

	return fmt.Errorf("code not found: %s", name)
}
