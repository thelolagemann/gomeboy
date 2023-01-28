package registers

import "fmt"

// HardwareRegisters is a collection of hardware registers,
// which allows them to be read and written to.
var HardwareRegisters = map[HardwareAddress]*Hardware{}

var newAddresses = map[HardwareAddress]struct{}{
	P1:     {},
	DIV:    {},
	TIMA:   {},
	TMA:    {},
	TAC:    {},
	LCDC:   {},
	STAT:   {},
	SCY:    {},
	SCX:    {},
	LY:     {},
	LYC:    {},
	DMA:    {},
	BGP:    {},
	OBP0:   {},
	OBP1:   {},
	WY:     {},
	WX:     {},
	VBK:    {},
	BCPS:   {},
	BCPD:   {},
	OCPS:   {},
	OCPD:   {},
	0xFF0F: {},
	0xFFFF: {},
}

// Has returns true if the HardwareRegisters map contains the given
// address.
func Has(address HardwareAddress) bool {
	_, ok := HardwareRegisters[address]
	_, ok2 := newAddresses[address]
	return ok && ok2
}

// Read returns the value of the hardware register at the given address.
func Read(address uint16) uint8 {
	return HardwareRegisters[HardwareAddress(address)].Read()
}

// Write writes the given value to the hardware register at the given address.
func Write(address uint16, value uint8) {
	HardwareRegisters[HardwareAddress(address)].Write(value)
}

// Hardware represents a hardware register of the Game
// Boy. The hardware registers are used to control and
// read the state of the hardware.
type Hardware struct {
	address HardwareAddress
	value   uint8
	set     func(v uint8)
	get     func() uint8

	read         func(address uint16) uint8
	write        func(address uint16, value uint8)
	writeHandler WriteHandler
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
	// LCDC is the address of the LCDC hardware register. The LCDC
	// hardware register is used to control the LCD.
	//
	// The register is set as follows:
	//
	//  Bit 7: LCD Enable             (0=Off, 1=On)
	//  Bit 6: Window Tile Map Display Select (0=9800-9BFF, 1=9C00-9FFF)
	//  Bit 5: Window Display Enable          (0=Off, 1=On)
	//  Bit 4: BG & Window Tile Data Select   (0=8800-97FF, 1=8000-8FFF)
	//  Bit 3: BG Tile Map Display Select     (0=9800-9BFF, 1=9C00-9FFF)
	//  Bit 2: OBJ (Sprite) Size              (0=8x8, 1=8x16)
	//  Bit 1: OBJ (Sprite) Display Enable    (0=Off, 1=On)
	//  Bit 0: BG Display (for CGB see below) (0=Off, 1=On)
	LCDC HardwareAddress = 0xFF40
	// STAT is the address of the STAT hardware register. The STAT
	// hardware register contains the status of the LCD, and is used
	// to report the mode the LCD is in, and to request LCD interrupts.
	//
	//  The register is set as follows:
	//
	//  Bit 6: LYC=LY Coincidence Interrupt (1=Enable) (Read/Write)
	//  Bit 5: Mode 2 OAM Interrupt         (1=Enable) (Read/Write)
	//  Bit 4: Mode 1 V-Blank Interrupt     (1=Enable) (Read/Write)
	//  Bit 3: Mode 0 H-Blank Interrupt     (1=Enable) (Read/Write)
	//  Bit 2: Coincidence Flag  (0:LYC<>LY, 1:LYC=LY) (Read Only)
	//  Bit 1-0: Mode Flag       (Mode 0-3, see below) (Read Only)
	//           0: During H-Blank
	//           1: During V-Blank
	//           2: During Searching OAM-RAM
	//           3: During Transfering Data to LCD Driver
	STAT HardwareAddress = 0xFF41
	// SCY is the address of the SCY hardware register. The SCY
	// hardware register is used to control the vertical scroll
	// position of the background.
	SCY HardwareAddress = 0xFF42
	// SCX is the address of the SCX hardware register. The SCX
	// hardware register is used to control the horizontal scroll
	// position of the background.
	SCX HardwareAddress = 0xFF43
	// LY is the address of the LY hardware register. The LY
	// hardware register is the current scanline being rendered.
	// The range of values for LY is 0-153. Writing any value to
	// LY resets it to 0.
	LY HardwareAddress = 0xFF44
	// LYC is the address of the LYC hardware register. This register
	// is compared to LY. When they are the same, the coincidence flag
	// in the STAT hardware register is set, and a STAT interrupt is
	// requested if the coincidence interrupt flag is set.
	LYC HardwareAddress = 0xFF45
	// DMA is the address of the DMA hardware register. Writing a value
	// to DMA transfers 160 bytes of data from ROM or RAM to OAM.
	DMA HardwareAddress = 0xFF46
	// BGP is the address of the BGP hardware register. The BGP
	// hardware register is used to set the shade of grey to use for
	// the background palette. BGP is only used in DMG mode.
	//
	// The palette is set as follows:
	//  Bit 7-6 - Shade for Color Number 3
	//  Bit 5-4 - Shade for Color Number 2
	//  Bit 3-2 - Shade for Color Number 1
	//  Bit 1-0 - Shade for Color Number 0
	BGP HardwareAddress = 0xFF47
	// OBP0 is the address of the OBP0 hardware register. The OBP0
	// hardware register is used to set the shade of grey to use for
	// sprite palette 0. OBP0 is only used in DMG mode.
	//
	// The palette is set as follows:
	//  Bit 7-6 - Shade for Color Number 3
	//  Bit 5-4 - Shade for Color Number 2
	//  Bit 3-2 - Shade for Color Number 1
	//  Bit 1-0 - Ignored (always transparent)
	OBP0 HardwareAddress = 0xFF48
	// OBP1 is the address of the OBP1 hardware register. The OBP1
	// hardware register is used to set the shade of grey to use for
	// sprite palette 1. OBP1 is only used in DMG mode.
	//
	// The palette is set as follows:
	//  Bit 7-6 - Shade for Color Number 3
	//  Bit 5-4 - Shade for Color Number 2
	//  Bit 3-2 - Shade for Color Number 1
	//  Bit 1-0 - Ignored (always transparent)
	OBP1 HardwareAddress = 0xFF49
	// WY is the address of the WY hardware register. The WY
	// hardware register is used to set the Y position of the window.
	// The window is visible when (if enabled) when both WY and WX
	// are in the ranges WX=0..166, WY=0..143 respectively. Values
	// WX=7 and WY=0 locates the window at the top left of the LCD.
	WY HardwareAddress = 0xFF4A
	// WX is the address of the WX hardware register. The WX
	// hardware register is used to set the X position of the window.
	// The window is visible when (if enabled) when both WY and WX
	// are in the ranges WX=0..166, WY=0..143 respectively. Values
	// WX=7 and WY=0 locates the window at the top left of the LCD.
	WX HardwareAddress = 0xFF4B
	// VBK is the address of the VBK hardware register. The VBK
	// hardware register is used to select the current VRAM bank.
	// VBK is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit 0   - VRAM Bank (0=Bank 0, 1=Bank 1)
	VBK HardwareAddress = 0xFF4F
	// BCPS is the address of the BCPS hardware register. The BCPS
	// hardware register is used to set the background palette index
	// and auto increment flag. BCPS is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit 7   - Auto Increment  (0=Off, 1=On)
	// Bit 5-0 - Background Palette Index  ($00-$3F)
	BCPS HardwareAddress = 0xFF68
	// BCPD is the address of the BCPD hardware register. The BCPD
	// hardware register is used to read and write the background
	// palette data. BCPD is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit   0 - 4 = Red Intensity   ($00-$1F)
	//  Bit   5 - 9 = Green Intensity ($00-$1F)
	//  Bit 10 - 14 = Blue Intensity  ($00-$1F)
	BCPD HardwareAddress = 0xFF69
	// OCPS is the address of the OCPS hardware register. The OCPS
	// hardware register is used to set the sprite palette index
	// and auto increment flag. OCPS is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit 7   - Auto Increment  (0=Off, 1=On)
	// Bit 5-0 - Sprite Palette Index  ($00-$3F)
	OCPS HardwareAddress = 0xFF6A
	// OCPD is the address of the OCPD hardware register. The OCPD
	// hardware register is used to read and write the sprite
	// palette data. OCPD is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit   0 - 4 = Red Intensity   ($00-$1F)
	//  Bit   5 - 9 = Green Intensity ($00-$1F)
	//  Bit 10 - 14 = Blue Intensity  ($00-$1F)
	OCPD HardwareAddress = 0xFF6B
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

	// add hardware register to global map of hardware registers
	HardwareRegisters[address] = h

	return h
}

// RegisterHardware registers a hardware register with the given
// address and options. Similar to NewHardware, but requires the
// caller to provide a pointer to the variable that will hold the
// hardware register.
func RegisterHardware(address HardwareAddress, set func(v uint8), get func() uint8, opts ...HardwareOpt) {
	h := &Hardware{
		address: address,
		// value:   value,
		set: set,
		get: get,
	}
	for _, opt := range opts {
		opt(h)
	}

	// TODO - add default read/write functions for registers that are not readable/writable
	// TODO - add global map of hardware registers for easy lookup
	// TODO - redirect MMU read/write to hardware registers if address is in range of hardware registers

	// add hardware register to global map of hardware registers
	HardwareRegisters[address] = h
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

// WithReadFunc allows the hardware register to be read, but with
// a custom read function.
func WithReadFunc(readFunc func(h *Hardware, address uint16) uint8) HardwareOpt {
	return func(h *Hardware) {
		h.read = func(address uint16) uint8 {
			return readFunc(h, address)
		}
	}
}

// WithWriteFunc allows the hardware register to be written to
// with a custom function.
func WithWriteFunc(write func(h *Hardware, address uint16, value uint8)) HardwareOpt {
	return func(h *Hardware) {
		h.write = func(address uint16, value uint8) {
			write(h, address, value)
		}
	}
}

func WithWriteHandler(writeHandler func(writeFn func())) HardwareOpt {
	return func(h *Hardware) {
		h.writeHandler = writeHandler
	}
}

type WriteHandler func(writeFn func())

type WriteFunc func(h *Hardware, address uint16, value uint8)

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
	// was the hardware register get function set?
	if h.get != nil {
		return h.get()
	}
	// was the hardware register read function set?
	if h.read != nil {
		return h.read(h.address)
	}

	// the hardware register is not readable, a panic is thrown
	panic(fmt.Sprintf("hardware: no read function for address 0x%04X", h.address))
}

func (h *Hardware) Write(value uint8) {
	// was the hardware register set function set?
	if h.set != nil {
		h.set(value)
		return
	}
	// was the hardware register write function set?
	if h.write != nil {
		if h.writeHandler != nil {
			h.writeHandler(func() {
				h.write(h.address, value)
			})
		} else {
			h.write(h.address, value)
		}
		return
	}

	// the hardware register is not writable, a panic is thrown
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
