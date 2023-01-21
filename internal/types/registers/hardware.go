package registers

import "fmt"

// Hardware represents a hardware register of the Game
// Boy. The hardware registers are used to control and
// read the state of the hardware.
type Hardware struct {
	address HardwareAddress
	value   uint8

	read  func(address uint16) uint8
	write func(address uint16, value uint8)
}

// HardwareAddress represents the address of a hardware
// register of the Game Boy. The hardware registers are mapped
// to memory addresses 0xFF00 - 0xFF7F.
type HardwareAddress = uint16

const (
	// P1 is the address of the P1 hardware register. The P1
	// hardware register is used to select the input keys to
	// be read by the CPU, and to read the state of the joypad.
	P1 HardwareAddress = 0xFF00
	// SB is the address of the SB hardware register. The SB
	// hardware register is used to transfer data between the
	// CPU and the serial port.
	SB HardwareAddress = 0xFF01
	// SC is the address of the SC hardware register. The SC
	// hardware register is used to control the serial port.
	SC HardwareAddress = 0xFF02
	// DIV is the address of the DIV hardware register. The DIV
	// hardware register is incremented at a rate of 16384Hz. Internally
	// it is a 16-bit register, but only the lower 8 bits may be read.
	DIV HardwareAddress = 0xFF04
	// TIMA is the address of the TIMA hardware register. The TIMA
	// hardware register is incremented at a rate specified by the TAC
	// hardware register. When TIMA overflows, it is reset to the value
	// specified by the TMA hardware register, and a timer interrupt is
	// requested. There are some obscure quirks with TIMA, which are
	// not currently emulated.
	TIMA HardwareAddress = 0xFF05
	// TMA is the address of the TMA hardware register. The TMA
	// hardware register is loaded into TIMA when it overflows.
	TMA HardwareAddress = 0xFF06
	// TAC is the address of the TAC hardware register. The TAC
	// hardware register is used to control the timer.
	TAC HardwareAddress = 0xFF07
	// IF is the address of the IF hardware register. The IF
	// hardware register is used to request interrupts. Writing a 1
	// to a bit in IF requests an interrupt, and writing a 0 clears
	// the request.
	//
	//  Bit 0: V-Blank Interrupt Request (INT 40h)  (1=Request)
	//  Bit 1: LCD STAT Interrupt Request (INT 48h) (1=Request)
	//  Bit 2: Timer Interrupt Request (INT 50h)    (1=Request)
	//  Bit 3: Serial Interrupt Request (INT 58h)   (1=Request)
	//  Bit 4: Joypad Interrupt Request (INT 60h)   (1=Request)
	IF HardwareAddress = 0xFF0F
	// IE is the address of the IE hardware register. The IE
	// hardware register is used to enable interrupts. Writing a 1
	// to a bit in IE enables the corresponding interrupt, and writing
	// a 0 disables the interrupt.
	IE HardwareAddress = 0xFFFF
)

// HardwareOpt is a function that configures a hardware register,
// such as making it readable, writable, or both.
type HardwareOpt func(*Hardware)

// NewHardware creates a new hardware register with the given
// address and options.
func NewHardware(address HardwareAddress, opts ...HardwareOpt) *Hardware {
	h := &Hardware{
		address: address,
		value:   0,
	}

	for _, opt := range opts {
		opt(h)
	}

	// TODO - add default read/write functions for registers that are not readable/writable
	// TODO - add global map of hardware registers for easy lookup
	// TODO - redirect MMU read/write to hardware registers if address is in range of hardware registers

	return h
}

// IsReadable allows the hardware register to be read.
func IsReadable() HardwareOpt {
	return func(h *Hardware) {
		h.read = func(address uint16) uint8 {
			return h.value
		}
	}
}

// IsWritable allows the hardware register to be written to.
func IsWritable() HardwareOpt {
	return func(h *Hardware) {
		h.write = func(address uint16, value uint8) {
			h.value = value
		}
	}
}

// IsReadableWritable allows the hardware register to be read
// and written to. This is the default behaviour for most
// hardware registers.
func IsReadableWritable() HardwareOpt {
	return func(h *Hardware) {
		IsReadable()(h)
		IsWritable()(h)
	}
}

// IsReadableMasked allows the hardware register to be read, but
// with a mask applied to the value. In the Game Boy all unused
// bits in a hardware register are set to 1, so this is used to
// emulate that behaviour.
func IsReadableMasked(mask uint8) HardwareOpt {
	return func(h *Hardware) {
		h.read = func(address uint16) uint8 {
			return h.value | mask
		}
	}
}

// IsWritableMasked allows the hardware register to be written to,
// but with a mask applied to the value being written. In the Game
// Boy all unused bits in a hardware register are set to 1, so this
// is used to emulate that behaviour.
func IsWritableMasked(mask uint8) HardwareOpt {
	return func(h *Hardware) {
		h.write = func(address uint16, value uint8) {
			h.value = value | mask
		}
	}
}

// Mask is simply a helper function to call IsReadableMasked and
// IsWritableMasked with the same mask. This is useful for hardware
// registers that are both readable and writable, but have unused
// bits that are set to 1.
func Mask(mask uint8) HardwareOpt {
	return func(h *Hardware) {
		IsReadableMasked(mask)(h)
		IsWritableMasked(mask)(h)
	}
}

func (h *Hardware) Read() uint8 {
	if h.read != nil {
		return h.read(h.address)
	}

	panic(fmt.Sprintf("hardware: no read function for address 0x%04X", h.address))
}

func (h *Hardware) Write(value uint8) {
	if h.write != nil {
		h.write(h.address, value)
		return
	}

	panic(fmt.Sprintf("hardware: no write function for address 0x%04X", h.address))
}

func (h *Hardware) Increment() {
	h.value++
}

func (h *Hardware) Reset() {
	h.value = 0
}

func (h *Hardware) Set(value uint8) {
	h.value = value
}

func (h *Hardware) Value() uint8 {
	return h.value
}
