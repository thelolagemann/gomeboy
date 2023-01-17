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
	// hardware register is used to read the state of the
	// directional keys and the buttons.
	P1 HardwareAddress = 0xFF00
	// SB is the address of the SB hardware register. The SB
	// hardware register is used to transfer data between the
	// CPU and the serial port.
	SB HardwareAddress = 0xFF01
	// SC is the address of the SC hardware register. The SC
	// hardware register is used to control the serial port.
	SC HardwareAddress = 0xFF02
	// DIV is the address of the DIV hardware register. The DIV
	// hardware register is used to read the divider register.
	DIV HardwareAddress = 0xFF04
	// TIMA is the address of the TIMA hardware register. The
	// TIMA hardware register is used to read the timer counter.
	TIMA HardwareAddress = 0xFF05
	// TMA is the address of the TMA hardware register. The TMA
	// hardware register is used to read the timer modulo.
	TMA HardwareAddress = 0xFF06
	// TAC is the address of the TAC hardware register. The TAC
	// hardware register is used to read the timer control.
	TAC HardwareAddress = 0xFF07
)

type HardwareOpt func(*Hardware)

func NewHardware(address HardwareAddress, opts ...HardwareOpt) *Hardware {
	h := &Hardware{
		address: address,
		value:   0,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

func IsReadable() HardwareOpt {
	return func(h *Hardware) {
		h.read = func(address uint16) uint8 {
			return h.value
		}
	}
}

func IsWritable() HardwareOpt {
	return func(h *Hardware) {
		h.write = func(address uint16, value uint8) {
			h.value = value
		}
	}
}

func IsReadableWritable() HardwareOpt {
	return func(h *Hardware) {
		IsReadable()(h)
		IsWritable()(h)
	}
}

func IsReadableMasked(mask uint8) HardwareOpt {
	return func(h *Hardware) {
		h.read = func(address uint16) uint8 {
			return h.value & mask
		}
	}
}

func IsWritableMasked(mask uint8) HardwareOpt {
	return func(h *Hardware) {
		h.write = func(address uint16, value uint8) {
			h.value = value & mask
		}
	}
}

func (h *Hardware) Read(address uint16) uint8 {
	if address != h.address {
		panic(fmt.Sprintf("hardware: illegal read address 0x%04X", address))
	}

	if h.read != nil {
		return h.read(address)
	}

	panic(fmt.Sprintf("hardware: no read function for address 0x%04X", address))
}

func (h *Hardware) Write(address uint16, value uint8) {
	if address != h.address {
		panic(fmt.Sprintf("hardware: illegal write address 0x%04X", address))
	}

	if h.write != nil {
		h.write(address, value)
		return
	}

	panic(fmt.Sprintf("hardware: no write function for address 0x%04X", address))
}
