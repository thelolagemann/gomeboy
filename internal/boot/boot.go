package boot

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
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
	raw         []byte
	md5Checksum string
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
		raw:         raw,
		md5Checksum: md5Checksum,
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

// known boot rom checksums
const (
	// DMGEarlyBootRomChecksum is the checksum of the DMG early boot rom,
	// a variant that was found in very early DMG units, only ever sold
	// in Japan.
	DMGEarlyBootRomChecksum = "a8f84a0ac44da5d3f0ee19f9cea80a8c"
	// DMGBootRomChecksum is the checksum of the DMG boot rom, which is
	// the boot rom found in the original B&W Game Boy.
	DMGBootRomChecksum = "32fbbd84168d3482956eb3c5051637f5"
	// MGBBootRomChecksum is the checksum of the MGB boot rom, which differs
	// only by a single byte from the DMG boot rom, used to identify the
	// MGB.
	MGBBootRomChecksum = "71a378e71ff30b2d8a1f02bf5c7896aa"
	// CGBEarlyBootRomChecksum is the checksum of the CGB early boot rom,
	// a variant that was found in very early CGB units, only ever sold
	// in Japan.
	CGBEarlyBootRomChecksum = "7c773f3c0b01cb73bca8e83227287b7f"
	// CGBBootRomChecksum is the checksum of the CGB boot rom, which is
	// the boot rom found in the original CGB.
	CGBBootRomChecksum = "dbfce9db9deaa2567f6a84fde55f9680"
)

// DMGBootRom is a boot rom for the DMG. This boot rom
// should be 256 bytes in length, and will be used to
// initialize the DMG hardware.
var DMGBootRom = [0x100]byte{}

// CGBBootROM is a boot rom for the CGB. This boot rom
// should be 2304 bytes in length, and will be used to
// initialize the CGB hardware.
var CGBBootROM = [0x900]byte{}

func init() {
	// load the boot roms from local files // TODO - make this configurable/ load from sameboy repo

	dmgBootRom, err := os.ReadFile("boot/dmg_boot.bin")
	if err != nil {
		panic(err)
	}
	copy(DMGBootRom[:], dmgBootRom)

	cgbBootRom, err := os.ReadFile("boot/cgb_boot.bin")
	if err != nil {
		panic(err)
	}
	copy(CGBBootROM[:], cgbBootRom)
}
