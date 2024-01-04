package cpu

import (
	"encoding/json"
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"os"
	"testing"
)

type instructionTest struct {
	Name    string `json:"name"`
	Initial struct {
		Pc  int     `json:"pc"`
		Sp  int     `json:"sp"`
		A   int     `json:"a"`
		B   int     `json:"b"`
		C   int     `json:"c"`
		D   int     `json:"d"`
		E   int     `json:"e"`
		F   int     `json:"f"`
		H   int     `json:"h"`
		L   int     `json:"l"`
		Ime int     `json:"ime"`
		Ei  int     `json:"ei"`
		RAM [][]int `json:"ram"`
	} `json:"initial"`
	Final struct {
		A   int     `json:"a"`
		B   int     `json:"b"`
		C   int     `json:"c"`
		D   int     `json:"d"`
		E   int     `json:"e"`
		F   int     `json:"f"`
		H   int     `json:"h"`
		L   int     `json:"l"`
		Pc  int     `json:"pc"`
		Sp  int     `json:"sp"`
		Ime int     `json:"ime"`
		Ei  int     `json:"ei"`
		RAM [][]int `json:"ram"`
	} `json:"final"`
}

type instructionTests []*instructionTest

func Test_Instructions(t *testing.T) {
	runInstructionTest(t, 0x10)
	//runInstructionTest(t, 0x20)
	/*for i := 0; i < 256; i++ {
		if i == 0xcb || i == 0x76 {
			continue // todo
		}
		runInstructionTest(t, uint8(i))
	}*/
}

func runInstructionTest(t *testing.T, opcode uint8) {
	// skip if no file
	if _, err := os.Stat(fmt.Sprintf("sm83-test-data/v1/%02x.json", opcode)); os.IsNotExist(err) {
		return
	}
	t.Run(fmt.Sprintf("%02x", opcode), func(t *testing.T) {
		t.Parallel()

		tests, err := loadInstructionTests(fmt.Sprintf("sm83-test-data/v1/%02x.json", opcode))
		if err != nil {
			t.Fatal(err)
			return
		}
		for _, xTest := range tests {
			c := &CPU{
				Registers: Registers{},
			}

			c.BC = &RegisterPair{High: &c.B, Low: &c.C}
			c.DE = &RegisterPair{High: &c.D, Low: &c.E}
			c.HL = &RegisterPair{High: &c.H, Low: &c.L}
			c.AF = &RegisterPair{High: &c.A, Low: &c.F}

			s := scheduler.NewScheduler()
			b := io.NewBus(s)
			b.Debug = true
			c.s = s
			c.b = b
			c.A = Register(xTest.Initial.A)
			c.B = Register(xTest.Initial.B)
			c.C = Register(xTest.Initial.C)
			c.D = Register(xTest.Initial.D)
			c.E = Register(xTest.Initial.E)
			c.F = Register(xTest.Initial.F)
			c.H = Register(xTest.Initial.H)
			c.L = Register(xTest.Initial.L)

			c.PC = uint16(xTest.Initial.Pc)
			c.SP = uint16(xTest.Initial.Sp)

			for _, row := range xTest.Initial.RAM {
				b.Write(uint16(row[0]), uint8(row[1]))
			}
			c.decode(c.readOperand())

			if c.A != Register(xTest.Final.A) {
				t.Errorf("A register value is incorrect")
			}
			if c.B != Register(xTest.Final.B) {
				t.Errorf("B expecting %02x, was %02x", xTest.Final.B, c.B)
			}
			if c.C != Register(xTest.Final.C) {
				t.Errorf("C expecting %02x, was %02x", xTest.Final.C, c.C)
			}
			if c.D != Register(xTest.Final.D) {
				t.Errorf("D expecting %02x, was %02x", xTest.Final.D, c.D)
			}
			if c.E != Register(xTest.Final.E) {
				t.Errorf("E expecting %02x, was %02x", xTest.Final.E, c.E)
			}
			if c.F != Register(xTest.Final.F) {
				t.Errorf("F expecting %02x, was %02x", xTest.Final.F, c.F)
			}
			if c.H != Register(xTest.Final.H) {
				t.Errorf("H expecting %02x, was %02x", xTest.Final.H, c.H)
			}
			if c.L != Register(xTest.Final.L) {
				t.Errorf("L expecting %02x, was %02x", xTest.Final.L, c.L)
			}
			if c.PC != uint16(xTest.Final.Pc) {
				fmt.Println(xTest.Initial.Ime, xTest.Initial.Ei)
				t.Errorf("PC expecting %04x, was %04x", xTest.Final.Pc, c.PC)
			}
			if c.SP != uint16(xTest.Final.Sp) {
				t.Errorf("SP expecting %04x, was %04x", xTest.Final.Sp, c.SP)
			}

			for _, row := range xTest.Final.RAM {
				if b.Get(uint16(row[0])) != uint8(row[1]) {
					t.Errorf("RAM expecting %02x at %04x, was %02x", row[1], row[0], b.Get(uint16(row[0])))
				}
			}
		}
	})
}

func loadInstructionTests(jsonFile string) (instructionTests, error) {
	f, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var t instructionTests
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return nil, err
	}

	return t, nil
}
