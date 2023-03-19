package types

// Address represents a memory address in the Game Boy's memory,
// which can be read from or written to. It is used to abstract
// away the actual memory addresses, and instead use a more
// readable and understandable interface.
type Address struct {
	// Read is a function that is called when the CPU reads from
	// the address.
	Read func(address uint16) uint8
	// Write is a function that is called when the CPU writes to
	// the address.
	Write func(address uint16, value uint8)
}

// HardwareAddress represents the address of a hardware
// register of the Game Boy. The hardware IO are mapped
// to memory addresses 0xFF00 - 0xFF7F & 0xFFFF.
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
	// TODO document the NRXX IO
	NR10 HardwareAddress = 0xFF10
	NR11 HardwareAddress = 0xFF11
	NR12 HardwareAddress = 0xFF12
	NR13 HardwareAddress = 0xFF13
	NR14 HardwareAddress = 0xFF14
	NR21 HardwareAddress = 0xFF16
	NR22 HardwareAddress = 0xFF17
	NR23 HardwareAddress = 0xFF18
	NR24 HardwareAddress = 0xFF19
	NR30 HardwareAddress = 0xFF1A
	NR31 HardwareAddress = 0xFF1B
	NR32 HardwareAddress = 0xFF1C
	NR33 HardwareAddress = 0xFF1D
	NR34 HardwareAddress = 0xFF1E
	NR41 HardwareAddress = 0xFF20
	NR42 HardwareAddress = 0xFF21
	NR43 HardwareAddress = 0xFF22
	NR44 HardwareAddress = 0xFF23
	NR50 HardwareAddress = 0xFF24
	NR51 HardwareAddress = 0xFF25
	NR52 HardwareAddress = 0xFF26
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
	//  Bit 5: mode 2 OAM Interrupt         (1=Enable) (Read/Write)
	//  Bit 4: mode 1 V-Blank Interrupt     (1=Enable) (Read/Write)
	//  Bit 3: mode 0 H-Blank Interrupt     (1=Enable) (Read/Write)
	//  Bit 2: Coincidence Flag  (0:LYC<>LY, 1:LYC=LY) (Read Only)
	//  Bit 1-0: mode Flag       (mode 0-3, see below) (Read Only)
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
	// The window is visible when (if enabled) both WY and WX
	// are in the ranges WX=0..166, WY=0..143 respectively. Values
	// WX=7 and WY=0 locates the window at the top left of the LCD.
	WY HardwareAddress = 0xFF4A
	// WX is the address of the WX hardware register. The WX
	// hardware register is used to set the X position of the window.
	// The window is visible when (if enabled) both WY and WX
	// are in the ranges WX=0..166, WY=0..143 respectively. Values
	// WX=7 and WY=0 locates the window at the top left of the LCD.
	WX HardwareAddress = 0xFF4B
	// KEY0 is the address of the KEY0 hardware register. The KEY0
	// hardware register is written to indicate the CGB compatibility
	// mode. KEY0 is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bits 7 - 4 - Not used
	//  Bit 3 - Disable some CGB functions (0=Normal, 1=Disable)
	//  Bit 2 - Disable all CGB functions (0=Normal, 1=Disable)
	//  Bit 1 - Unused
	//  Bit 0 - Unknown
	KEY0 HardwareAddress = 0xFF4C
	// KEY1 is the address of the KEY1 hardware register. The KEY1
	// hardware register is written to indicate the CGB speed mode.
	// KEY1 is only used in CGB mode.
	KEY1 HardwareAddress = 0xFF4D
	// VBK is the address of the VBK hardware register. The VBK
	// hardware register is used to select the current VRAM bank.
	// VBK is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit 0   - VRAM Bank (0=Bank 0, 1=Bank 1)
	VBK HardwareAddress = 0xFF4F
	// BDIS is the address of the BDIS hardware register. The BDIS
	// hardware register is used only to disable the boot ROM.
	//
	// The register is set as follows:
	//  Bit 0   - Disable boot ROM (0=Enable, 1=Disable)
	BDIS  HardwareAddress = 0xFF50
	HDMA1 HardwareAddress = 0xFF51
	HDMA2 HardwareAddress = 0xFF52
	HDMA3 HardwareAddress = 0xFF53
	HDMA4 HardwareAddress = 0xFF54
	HDMA5 HardwareAddress = 0xFF55
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
	// OPRI is the address of the OPRI hardware register. The OPRI
	// hardware register is used to set the sprite priority. OPRI
	// is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit 0 - Sprite priority (0=OBJ priority, 1=Coordinates priority)
	OPRI HardwareAddress = 0xFF6C
	// SVBK is the address of the SVBK hardware register. The SVBK
	// hardware register is used to select the current WRAM bank.
	// SVBK is only used in CGB mode.
	//
	// The register is set as follows:
	//  Bit 0 - 2 = WRAM Bank ($00-$07)
	//
	// Note: Writing a value of 00h selects WRAM Bank 01h.
	SVBK HardwareAddress = 0xFF70
	// RP is the address of the RP hardware register. The RP
	// hardware register is used control the IR port.
	//
	// The register is set as follows:
	//  Bit 7 - IR Port Enable (0=Disable, 1=Enable)
	//  Bit 6 - IR Port Input  (0=Low, 1=High)
	RP HardwareAddress = 0xFF56
	// IE is the address of the IE hardware register. The IE
	// hardware register is used to Enable interrupts. Writing a 1
	// to a bit in IE Enables the corresponding interrupt, and writing
	// a 0 disables the interrupt.
	IE HardwareAddress = 0xFFFF
)
