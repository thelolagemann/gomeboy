package lcd

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

// Status represents the LCD status register. It contains information about the
// current state of the LCD controller. Its value is stored in the registers.STAT
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

	reg *registers.Hardware
}

func (s *Status) init(handler registers.WriteHandler) {
	// setup the register
	s.reg = registers.NewHardware(
		registers.STAT,
		registers.WithReadFunc(func(h *registers.Hardware, address uint16) uint8 {
			val := h.Value() | types.Bit7
			// set the R_ONLY bits
			if s.Coincidence {
				val |= types.Bit2 // set the coincidence flag
			}
			val |= uint8(s.Mode) // set the mode
			return val
		}),
		registers.WithWriteFunc(func(h *registers.Hardware, address uint16, value uint8) {
			s.CoincidenceInterrupt = utils.Test(value, 6)
			s.OAMInterrupt = utils.Test(value, 5)
			s.VBlankInterrupt = utils.Test(value, 4)
			s.HBlankInterrupt = utils.Test(value, 3)

			// set raw value
			h.Set(value & 0xF8)
		}),
		registers.WithWriteHandler(handler),
	)
}

// NewStatus returns a new Status.
func NewStatus(writeHandler registers.WriteHandler) *Status {
	s := &Status{
		CoincidenceInterrupt: false,
		OAMInterrupt:         false,
		VBlankInterrupt:      false,
		HBlankInterrupt:      false,
		Coincidence:          false,
		Mode:                 0,
	}
	s.init(writeHandler)
	return s
}

// SetMode sets the mode of the LCD controller and triggers the appropriate
// interrupts.
func (s *Status) SetMode(mode Mode) {
	s.Mode = mode
}
