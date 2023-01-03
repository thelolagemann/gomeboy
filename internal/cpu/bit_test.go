package cpu

import (
	"testing"
)

func TestBit(t *testing.T) {
	c := NewCPU(nil)
	t.Run("set", func(t *testing.T) {
		c.A = c.setBit(c.A, 0)
		if c.A != 0x01 {
			t.Errorf("expected 0x02, got 0x%02x", c.A)
		}
	})
	t.Run("clear", func(t *testing.T) {
		c.A = c.clearBit(c.A, 0)
		if c.A != 0x00 {
			t.Errorf("expected A to be 0x00, got 0x%02X", c.A)
		}
	})
	t.Run("test", func(t *testing.T) {
		c.testBit(c.A, 0)
		if !c.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be set, got unset")
		}
		c.A = 0x01
		c.testBit(c.A, 0)
		if c.isFlagSet(FlagZero) {
			t.Errorf("expected zero flag to be unset, got set")
		}
	})
}

func TestInstruction_16Bit_Bits(t *testing.T) {
	// 0x40 - 0x7F BIT b,r

}
