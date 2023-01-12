package boot

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
)

type ROM struct {
	raw         []byte
	md5Checksum string
}

func NewBootROM(raw []byte, md5Checksum string) *ROM {
	// ensure correct lengths
	if len(raw) != 256 && len(raw) != 2304 {
		panic(fmt.Sprintf("boot: invalid boot rom length: %d", len(raw)))
	}

	// validate checksum
	if md5Checksum == "" {
		panic("boot: checksum is empty")
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

func (b *ROM) Read(addr uint16) byte {
	return b.raw[addr]
}

func (b *ROM) Write(addr uint16, val byte) {
	panic("boot: illegal write to boot")
}

const (
	// DMGEarlyBootRomChecksum is the checksum of the DMG early boot rom,
	// a variant that was found in very early DMG units, only ever sold
	// in Japan. This boot room is not used in the emulator, but is
	// included for completeness.
	DMGEarlyBootRomChecksum = "a8f84a0ac44da5d3f0ee19f9cea80a8c"
	// DMGBootRomChecksum is the checksum of the DMG boot rom, which is
	// the boot rom found in the original B&W Game Boy. This boot rom is
	// used in the emulator, and is the default boot rom for DMG mode.
	DMGBootRomChecksum = "32fbbd84168d3482956eb3c5051637f5"
	// MGBBootRomChecksum is the checksum of the MGB boot rom, which differs
	// only by a single byte from the DMG boot rom, used to identify the
	// MGB. This boot rom is not used in the emulator, but is included for
	// completeness.
	MGBBootRomChecksum   = "71a378e71ff30b2d8a1f02bf5c7896aa"
	CGBEarlyBiosChecksum = "7c773f3c0b01cb73bca8e83227287b7f"
	CGBBiosChecksum      = "dbfce9db9deaa2567f6a84fde55f9680"
)

// DMGBootRom is the boot rom for the original B&W Game Boy. This is
// the default boot rom for DMG mode.
var DMGBootRom = [0x100]byte{}

// CGBBootROM is the boot rom for the Colour Game Boy. This is the
// default boot rom // for CGB mode.
var CGBBootROM = [0x900]byte{}

func init() {
	// load the boot roms from local files // TODO - make this configurable

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
