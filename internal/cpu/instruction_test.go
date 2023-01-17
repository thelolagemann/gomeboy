package cpu

import (
	"github.com/thelolagemann/go-gameboy/internal/cartridge"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/io"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ram"
	"github.com/thelolagemann/go-gameboy/internal/timer"
	"testing"
)

var (
	cpu *CPU
)

func TestInstruction_Timing(t *testing.T) {
	timings := []uint8{
		1, 3, 2, 2, 1, 1, 2, 1, 5, 2, 2, 2, 1, 1, 2, 1,
		0, 3, 2, 2, 1, 1, 2, 1, 3, 2, 2, 2, 1, 1, 2, 1,
		2, 3, 2, 2, 1, 1, 2, 1, 2, 2, 2, 2, 1, 1, 2, 1,
		2, 3, 2, 2, 3, 3, 3, 1, 2, 2, 2, 2, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		2, 2, 2, 2, 2, 2, 0, 2, 1, 1, 1, 1, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		2, 3, 3, 4, 3, 4, 2, 4, 2, 4, 3, 0, 3, 6, 2, 4,
		2, 3, 3, 0, 3, 4, 2, 4, 2, 4, 3, 0, 3, 0, 2, 4,
		3, 3, 2, 0, 0, 4, 2, 4, 4, 1, 4, 0, 0, 0, 2, 4,
		3, 3, 2, 1, 0, 4, 2, 4, 3, 2, 4, 1, 0, 0, 2, 4,
	}
	for i, timing := range timings {
		if timing == 0 {
			continue
		}

		testInstruction(t, InstructionSet[uint8(i)].Name(), uint8(i), func(t *testing.T, i Instructor) {
			if i.Cycles() != timing {
				t.Errorf("expected %d cycles, got %d", timing, i.Cycles())
			}
		})
	}

	cbTiming := []uint8{
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 3, 2, 2, 2, 2, 2, 2, 2, 3, 2,
		2, 2, 2, 2, 2, 2, 3, 2, 2, 2, 2, 2, 2, 2, 3, 2,
		2, 2, 2, 2, 2, 2, 3, 2, 2, 2, 2, 2, 2, 2, 3, 2,
		2, 2, 2, 2, 2, 2, 3, 2, 2, 2, 2, 2, 2, 2, 3, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
		2, 2, 2, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2, 4, 2,
	}

	for i, timing := range cbTiming {
		if timing == 0 {
			continue
		}

		testInstructionCB(t, InstructionSetCB[uint8(i)].Instruction().Name(), uint8(i), func(t *testing.T, i Instructor) {
			if i.Cycles() != timing {
				t.Errorf("expected %d cycles, got %d", timing, i.Cycles())
			}
		})
	}
}

func TestLoadInstructions(t *testing.T) {
	// 0x02 - LD (BC), A - Load A into (BC)
	testInstruction(t, "LD (BC), A", 0x02, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42

		cpu.BC.SetUint16(0x1234)
		instr.Execute(cpu, nil)
		if cpu.mmu.Read(cpu.BC.Uint16()) != 0x42 {
			t.Errorf("expected 0x42 at 0x1234, got 0x%02X", cpu.mmu.Read(0x1234))
		}
	})
	// 0x0A - LD A, (BC) - Load value pointed to by BC into A
	testInstruction(t, "LD A, (BC)", 0x0A, func(t *testing.T, instr Instructor) {
		cpu.BC.SetUint16(0x1234)
		cpu.mmu.Write(cpu.BC.Uint16(), 0x42)
		instr.Execute(cpu, nil)
		if cpu.A != 0x42 {
			t.Errorf("expected 0x42 in A, got 0x%02X", cpu.A)
		}
	})
	// 0x12 - LD (DE), A - Load A into address pointed to by DE
	testInstruction(t, "LD (DE), A", 0x12, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.DE.SetUint16(0x1234)
		instr.Execute(cpu, nil)
		if cpu.mmu.Read(cpu.DE.Uint16()) != 0x42 {
			t.Errorf("expected 0x42 at 0x1234, got 0x%02X", cpu.mmu.Read(0x1234))
		}
	})
	// 0x1A - LD A, (DE) - Load value pointed to by DE into A
	testInstruction(t, "LD A, (DE)", 0x1A, func(t *testing.T, instr Instructor) {
		cpu.DE.SetUint16(0x1234)
		cpu.mmu.Write(cpu.DE.Uint16(), 0x42)
		instr.Execute(cpu, nil)
		if cpu.A != 0x42 {
			t.Errorf("expected 0x42 in A, got 0x%02X", cpu.A)
		}
	})

	// 0x20 - LD (HL+), A - Load A into address pointed to by HL, then increment HL
	testInstruction(t, "LD (HL+), A", 0x22, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.HL.SetUint16(0x1234)
		instr.Execute(cpu, nil)
		if cpu.mmu.Read(cpu.HL.Uint16()-1) != 0x42 {
			t.Errorf("expected 0x42 at 0x1234, got 0x%02X", cpu.mmu.Read(cpu.HL.Uint16()-1))
		}
		if cpu.HL.Uint16() != 0x1235 {
			t.Errorf("expected HL to be 0x1235, got 0x%04X", cpu.HL.Uint16())
		}
	})
	// 0x2A - LD A, (HL+) - Load value pointed to by HL into A, then increment HL
	testInstruction(t, "LD A, (HL+)", 0x2A, func(t *testing.T, instr Instructor) {
		cpu.HL.SetUint16(0x1234)
		cpu.mmu.Write(cpu.HL.Uint16(), 0x42)
		instr.Execute(cpu, nil)
		if cpu.A != 0x42 {
			t.Errorf("expected 0x42 in A, got 0x%02X", cpu.A)
		}
		if cpu.HL.Uint16() != 0x1235 {
			t.Errorf("expected HL to be 0x1235, got 0x%04X", cpu.HL.Uint16())
		}
	})
	// 0x32 - LD (HL-), A - Load A into address pointed to by HL, then decrement HL
	testInstruction(t, "LD (HL-), A", 0x32, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.HL.SetUint16(0x1234)
		instr.Execute(cpu, nil)
		if cpu.mmu.Read(cpu.HL.Uint16()+1) != 0x42 {
			t.Errorf("expected 0x42 at 0x1234, got 0x%02X", cpu.mmu.Read(cpu.HL.Uint16()+1))
		}
		if cpu.HL.Uint16() != 0x1233 {
			t.Errorf("expected HL to be 0x1233, got 0x%04X", cpu.HL.Uint16())
		}
	})
	// 0x36 - LD (HL), n - Load 8-bit immediate value into address pointed to by HL
	testInstruction(t, "LD (HL), n", 0x36, func(t *testing.T, instr Instructor) {
		for i := 0; i < 0xFF; i++ {
			cpu.HL.SetUint16(0x1234)
			instr.Execute(cpu, []uint8{uint8(i)})
			if cpu.mmu.Read(cpu.HL.Uint16()) != uint8(i) {
				t.Errorf("expected 0x%02X at 0x1234, got 0x%02X", i, cpu.mmu.Read(0x1234))
			}
		}
	})
	// 0x3A - LD A, (HL-) - Load value pointed to by HL into A, then decrement HL
	testInstruction(t, "LD A, (HL-)", 0x3A, func(t *testing.T, instr Instructor) {
		cpu.HL.SetUint16(0x1234)
		cpu.mmu.Write(cpu.HL.Uint16(), 0x42)
		instr.Execute(cpu, nil)
		if cpu.A != 0x42 {
			t.Errorf("expected 0x42 in A, got 0x%02X", cpu.A)
		}
		if cpu.HL.Uint16() != 0x1233 {
			t.Errorf("expected HL to be 0x1233, got 0x%04X", cpu.HL.Uint16())
		}
	})
}

func TestArithmetic1(t *testing.T) {
	// 0x86 - ADD A, (HL) - Add value pointed to by HL to A
	testInstruction(t, "ADD A, (HL)", 0x86, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.HL.SetUint16(0x1234)
		cpu.mmu.Write(cpu.HL.Uint16(), 0x42)

		instr.Execute(cpu, nil)

		if cpu.A != 0x84 {
			t.Errorf("expected A to be 0x84, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0, got 0x%02X", cpu.F)
		}

		// test half carry
		cpu.A = 0x0F
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("expected half carry flag to be set")
		}

		// test zero
		cpu.A = 0xFF
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}

		// test carry
		cpu.A = 0xFF

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected carry flag to be set")
		}
	})
	// 0x8E - ADC A, (HL) - Add value at address HL + carry flag to A
	testInstruction(t, "ADC A, (HL)", 0x8E, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.setFlag(FlagCarry)
		cpu.HL.SetUint16(0x1234)

		cpu.mmu.Write(cpu.HL.Uint16(), 0x42)

		instr.Execute(cpu, nil)

		if cpu.A != 0x85 {
			t.Errorf("expected A to be 0x85, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0, got 0x%02X", cpu.F)
		}

		// test half carry
		cpu.A = 0x0F

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("expected half carry flag to be set")
		}

		// test zero
		cpu.setFlag(FlagCarry)
		cpu.A = 0xFE
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Error("expected zero flag to be set", cpu.A)
		}

		// test carry
		cpu.A = 0xFF

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected carry flag to be set")
		}
	})
	// 0x96 - SUB (HL) - Subtract value pointed to by HL from A
	testInstruction(t, "SUB (HL)", 0x96, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.HL.SetUint16(0x1234)

		cpu.mmu.Write(cpu.HL.Uint16(), 0x10)

		instr.Execute(cpu, nil)

		if cpu.A != 0x32 {
			t.Errorf("expected A to be 0x32, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if !cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x40, got 0x%02X", cpu.F)
		}

		// test half carry (borrow)
		cpu.A = 0x01
		cpu.mmu.Write(cpu.HL.Uint16(), 0x0F)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("expected half carry flag to be set")
		}

		// test zero
		cpu.A = 0x01
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}

		// test carry
		cpu.A = 0x00
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected carry flag to be set")
		}
	})
	// 0x9E - SBC A, (HL) - Subtract value pointed to by HL from A with carry
	testInstruction(t, "SBC A, (HL)", 0x9E, func(t *testing.T, instr Instructor) {
		cpu.A = 0x42
		cpu.HL.SetUint16(0x1234)

		cpu.mmu.Write(cpu.HL.Uint16(), 0x10)

		cpu.setFlag(FlagCarry)

		instr.Execute(cpu, nil)

		if cpu.A != 0x31 {
			t.Errorf("expected A to be 0x31, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if !cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x40, got 0x%02X", cpu.F)
		}

		// test half carry (borrow)
		cpu.A = 0x01
		cpu.mmu.Write(cpu.HL.Uint16(), 0x0F)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagHalfCarry) {
			t.Errorf("expected half carry flag to be set")
		}

		// test zero
		cpu.A = 0x02
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}

		// test carry
		cpu.A = 0x00
		cpu.mmu.Write(cpu.HL.Uint16(), 0x01)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected carry flag to be set")
		}
	})
}

func TestLogic(t *testing.T) {
	// 0xA6 - AND (HL) - Logical AND value pointed to by HL with A
	testInstruction(t, "AND (HL)", 0xA6, func(t *testing.T, instr Instructor) {
		cpu.A = 0b10101010
		cpu.HL.SetUint16(0x1234)

		cpu.mmu.Write(cpu.HL.Uint16(), 0b11010101)

		instr.Execute(cpu, nil)

		if cpu.A != 0x80 {
			t.Errorf("expected A to be 0x80, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || !cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x20, got 0x%02X", cpu.F)
		}

		// test zero
		cpu.A = 0b01010101
		cpu.mmu.Write(cpu.HL.Uint16(), 0b10101010)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}
	})
	// 0xAE - XOR (HL) - Logical XOR (HL) with A
	testInstruction(t, "XOR (HL)", 0xAE, func(t *testing.T, instr Instructor) {
		cpu.A = 0b10101010
		cpu.HL.SetUint16(0x1234)

		cpu.mmu.Write(cpu.HL.Uint16(), 0b11010101)

		instr.Execute(cpu, nil)

		if cpu.A != 0x7F {
			t.Errorf("expected A to be 0x7F, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x00, got 0x%02X", cpu.F)
		}

		// test zero
		cpu.A = 0
		cpu.mmu.Write(cpu.HL.Uint16(), 0)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}
	})
	// 0xAF - XOR A - Logical XOR A with A
	testInstruction(t, "XOR A", 0xAF, func(t *testing.T, instr Instructor) {
		cpu.A = 0b10101010

		instr.Execute(cpu, nil)

		if cpu.A != 0 {
			t.Errorf("expected A to be 0, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || !cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x80, got 0x%02X", cpu.F)
		}

		// test zero
		cpu.A = 0

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}
	})
	// 0xB6 - OR (HL) - Logical OR value pointed to by HL with A
	testInstruction(t, "OR (HL)", 0xB6, func(t *testing.T, instr Instructor) {
		cpu.A = 0b10101010
		cpu.HL.SetUint16(0x1234)

		cpu.mmu.Write(cpu.HL.Uint16(), 0b11010101)

		instr.Execute(cpu, nil)

		if cpu.A != 0b11111111 {
			t.Errorf("expected A to be 0xFF, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x00, got 0x%02X", cpu.F)
		}

		// test zero
		cpu.A = 0
		cpu.mmu.Write(cpu.HL.Uint16(), 0)

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}
	})
	// 0xB7 - OR A - Logical OR A with A
	testInstruction(t, "OR A", 0xB7, func(t *testing.T, instr Instructor) {
		cpu.A = 0b10101010

		instr.Execute(cpu, nil)

		if cpu.A != 0b10101010 {
			t.Errorf("expected A to be 0xAA, got 0x%02X", cpu.A)
		}

		// ensure flags are set correctly
		if cpu.isFlagSet(FlagSubtract) || cpu.isFlagSet(FlagZero) || cpu.isFlagSet(FlagHalfCarry) || cpu.isFlagSet(FlagCarry) {
			t.Errorf("expected flags to be 0x00, got 0x%02X", cpu.F)
		}

		// test zero
		cpu.A = 0

		instr.Execute(cpu, nil)

		if !cpu.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set")
		}
	})
	// 0xB8 - CP B - Compare B with A
	testInstruction(t, "CP B", 0xB8, func(t *testing.T, instr Instructor) {
		cpu.A = 0b10101010
		cpu.B = 0b11010101

		instr.Execute(cpu, nil)

	})
}

func testInstruction(t *testing.T, name string, opcode uint8, f func(*testing.T, Instructor)) {
	// reset CPU
	cart := cartridge.NewEmptyCartridge()
	irq := interrupts.NewService()
	pad := joypad.New(irq)
	serial := io.NewSerial()
	tCtl := timer.NewController(irq)
	sound := ram.NewRAM(0x10)

	memBus := mmu.NewMMU(cart, pad, serial, tCtl, irq, sound)
	memBus.EnableMock()

	video := ppu.New(memBus, irq)
	memBus.AttachVideo(video)
	cpu = NewCPU(memBus, irq)

	t.Run(name, func(t *testing.T) {
		f(t, InstructionSet[opcode])
	})
}

func testInstructionCB(t *testing.T, name string, opcode uint8, f func(*testing.T, Instructor)) {
	// reset CPU
	cart := cartridge.NewEmptyCartridge()
	irq := interrupts.NewService()
	pad := joypad.New(irq)
	serial := io.NewSerial()
	tCtl := timer.NewController(irq)
	sound := ram.NewRAM(0x10)

	memBus := mmu.NewMMU(cart, pad, serial, tCtl, irq, sound)
	memBus.EnableMock()

	video := ppu.New(memBus, irq)
	memBus.AttachVideo(video)
	cpu = NewCPU(memBus, irq)

	t.Run(name, func(t *testing.T) {
		f(t, InstructionSetCB[opcode].Instruction())
	})
}
