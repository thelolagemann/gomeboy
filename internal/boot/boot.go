package boot

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

// ROM is a boot rom for the Game Boy. When the Game Boy first
// powers on, the boot rom is mapped to memory addresses 0x0000 -
// 0x00FF. The boot rom performs a series of tasks, such as
// initializing the hardware, setting the stack pointer, scrolling
// the Nintendo logo, etc. Once the boot rom has completed its
// tasks, it is unmapped from memory, and the cartridge is mapped
// over the boot rom, thus starting the cartridge execution, and
// preventing the boot rom from being executed again.
type ROM struct {
	raw []byte

	// the md5 checksum of the boot rom
	checksum string // TODO make this a byte array
}

// NewBootROM returns a new boot rom for the Game
// Boy. The boot rom should be provided as a byte slice,
// and will be compared against the provided md5 checksum,
// to ensure the boot rom is valid. There are multiple
// situations where a panic will occur:
//
//  1. The boot rom is not a valid length (256 or 2304 bytes)
//  2. The boot rom does not match the provided MD5 checksum
//  3. The provided MD5 checksum is not a valid MD5 checksum (32 characters)
//
// Several checksums for various boot roms are provided in the
// boot package for convenience, and can be used to verify the
// provided boot rom. There are two boot roms provided in the
// boot package, one for the DMG/MGB/SGB, and one for the CGB.
//
//	 Example:
//
//		// loading the DMG boot rom
//		boot.NewBootROM(boot.DMGBootROM[:], boot.DMGBootROMChecksum)
//		// loading the CGB boot rom
//		boot.NewBootROM(boot.CGBBootROM[:], boot.CGBBootROMChecksum)
func NewBootROM(raw []byte, md5Checksum string) *ROM {
	// ensure correct lengths
	if len(raw) != 256 && len(raw) != 2304 { // 256 bytes for DMG/MGB/SGB, 2304 bytes for CGB
		panic(fmt.Sprintf("boot: invalid boot rom length: %d", len(raw)))
	}

	// validate checksum
	if len(md5Checksum) != 32 {
		panic(fmt.Sprintf("boot: invalid md5 checksum: %s", md5Checksum))
	}
	bootChecksum := md5.Sum(raw)
	if md5Checksum != hex.EncodeToString(bootChecksum[:]) {
		panic(fmt.Sprintf("boot: invalid checksum expected %s got %s", md5Checksum, hex.EncodeToString(bootChecksum[:])))
	}

	return &ROM{
		raw: raw,
	}
}

func LoadBootROM(raw []byte) *ROM {
	// ensure correct lengths
	if len(raw) != 256 && len(raw) != 2304 { // 256 bytes for DMG/MGB/SGB, 2304 bytes for CGB
		panic(fmt.Sprintf("boot: invalid boot rom length: %d", len(raw)))
	}

	// calculate checksum
	bootChecksum := md5.Sum(raw)

	return &ROM{
		raw:      raw,
		checksum: hex.EncodeToString(bootChecksum[:]),
	}
}

// Read implements the mmu.IOBus interface. This method
// simply returns the byte at the provided address.
func (b *ROM) Read(addr uint16) byte {
	return b.raw[addr]
}

// Write implements the mmu.IOBus interface. The boot rom
// is read only, and will panic if a write is attempted.
func (b *ROM) Write(addr uint16, val byte) {
	panic("boot: illegal write to boot")
}

// Checksum returns the md5 checksum of the boot rom.
func (b *ROM) Checksum() string {
	if b == nil {
		return ""
	}
	return b.checksum
}

// Model returns the model of the boot rom. The model
// is determined by the checksum of the boot rom.
func (b *ROM) Model() string {
	if b == nil {
		return "none"
	}
	for model, checksum := range knownBootROMChecksums {
		if checksum == b.checksum {
			return model
		}
	}
	return "unknown"
}

// knownBootROMChecksums is a map of known boot rom checksums,
// with the key being the model, and the value being the checksum.
var knownBootROMChecksums = map[string]string{
	"DMG0": DMGEarlyBootROMChecksum,
	"DMG":  DMBBootROMChecksum,
	"MGB":  MGBBootROMChecksum,
	"CGB0": CGBEarlyBootROMChecksum,
	"CGB":  CGBBootROMChecksum,
}

// known boot rom checksums
const (
	// DMGEarlyBootROMChecksum is the checksum of the DMG early boot rom,
	// a variant that was found in very early DMG units, only ever sold
	// in Japan.
	DMGEarlyBootROMChecksum = "a8f84a0ac44da5d3f0ee19f9cea80a8c"
	// DMBBootROMChecksum is the checksum of the DMG boot rom, which is
	// the boot rom found in the original B&W Game Boy.
	DMBBootROMChecksum = "32fbbd84168d3482956eb3c5051637f5"
	// MGBBootROMChecksum is the checksum of the MGB boot rom, which differs
	// only by a single byte from the DMG boot rom, used to identify the
	// MGB.
	MGBBootROMChecksum = "71a378e71ff30b2d8a1f02bf5c7896aa"
	// CGBEarlyBootROMChecksum is the checksum of the CGB early boot rom,
	// a variant that was found in very early CGB units, only ever sold
	// in Japan.
	CGBEarlyBootROMChecksum = "7c773f3c0b01cb73bca8e83227287b7f"
	// CGBBootROMChecksum is the checksum of the CGB boot rom, which is
	// the boot rom found in the original CGB.
	CGBBootROMChecksum = "dbfce9db9deaa2567f6a84fde55f9680"
)

// DMGBootROM is a boot rom for the DMG. This boot rom
// should be 256 bytes in length, and will be used to
// initialize the DMG hardware.
var DMGBootROM = [0x100]byte{}

// CGBBootROM is a boot rom for the CGB. This boot rom
// should be 2304 bytes in length, and will be used to
// initialize the CGB hardware.
var CGBBootROM = [0x900]byte{}
