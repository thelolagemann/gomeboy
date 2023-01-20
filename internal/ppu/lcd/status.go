package lcd

import (
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

const (
	// StatusRegister is the address of the status register.
	StatusRegister = 0xFF41
)

// Status represents the LCD status register. It contains information about the
// current state of the LCD controller. Its value is stored in the StatusRegister
// (0xFF41) as follows:
//
//	Bit 6 - LYC=LY Coincidence Interrupt (1=Enable) (Read/Write)
//	Bit 5 - Mode 2 OAM Interrupt         (1=Enable) (Read/Write)
//	Bit 4 - Mode 1 V-Blank Interrupt     (1=Enable) (Read/Write)
//	Bit 3 - Mode 0 H-Blank Interrupt     (1=Enable) (Read/Write)
//	Bit 2 - Coincidence Flag  (0:LYC<>LY, 1:LYC=LY) (Read Only)
//	Bit 1-0 - Mode Flag       (Mode 0-3, see below) (Read Only)
//		0: During H-Blank
//		1: During V-Blank
//		2: During Searching OAM-RAM
//		3: During Transferring Data to LCD Driver
type Status struct {
	// CoincidenceInterrupt is set when the LYC=LY coincidence interrupt is
	// enabled.
	CoincidenceInterrupt bool
	// OAMInterrupt is set when the OAM interrupt is enabled.
	OAMInterrupt bool
	// VBlankInterrupt is set when the V-Blank interrupt is enabled.
	VBlankInterrupt bool
	// HBlankInterrupt is set when the H-Blank interrupt is enabled.
	HBlankInterrupt bool
	// Coincidence is set when the LYC=LY coincidence flag is set.
	Coincidence bool
	// Mode is the current mode of the LCD controller.
	Mode Mode
}

// NewStatus returns a new Status.
func NewStatus() *Status {
	return &Status{
		CoincidenceInterrupt: false,
		OAMInterrupt:         false,
		VBlankInterrupt:      false,
		HBlankInterrupt:      false,
		Coincidence:          false,
		Mode:                 0,
	}
}

// SetMode sets the mode of the LCD controller and triggers the appropriate
// interrupts.
func (s *Status) SetMode(mode Mode) {
	s.Mode = mode
}

// Write writes the value to the status register.
func (s *Status) Write(address uint16, value uint8) {
	if address != StatusRegister {
		panic("illegal write for LCDStatus")
	}
	s.CoincidenceInterrupt = utils.Test(value, 6)
	s.OAMInterrupt = utils.Test(value, 5)
	s.VBlankInterrupt = utils.Test(value, 4)
	s.HBlankInterrupt = utils.Test(value, 3)
}

// Read returns the value of the status register.
func (s *Status) Read(address uint16) uint8 {
	if address != StatusRegister {
		panic("illegal read for LCDStatus")
	}
	var value uint8
	if s.CoincidenceInterrupt {
		value |= 1 << 6
	}
	if s.OAMInterrupt {
		value |= 1 << 5
	}
	if s.VBlankInterrupt {
		value |= 1 << 4
	}
	if s.HBlankInterrupt {
		value |= 1 << 3
	}
	if s.Coincidence {
		value |= 1 << 2
	}
	// set the mode bits 1 and 0
	value |= uint8(s.Mode) & 0x03
	return value | 0b10000000 // bit 7 is always set
}
