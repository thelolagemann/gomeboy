# boot

A boot ROM package for the Game Boy. This package is not necessary in order to emulate the boot ROM, as it's simply
read-only memory that can be implemented with a simple `[]byte` slice. However, it provides a convenient way to
construct a boot ROM from a file and a known-good checksum, as well as providing several known boot ROM checksums.

## Overview

The package provides a ROM struct which represents a boot ROM for the Game Boy. When the Game Boy first powers on, the 
boot ROM is mapped to memory addresses `0x0000` - `0x00FF`. The boot ROM performs a series of tasks, such as initializing 
the hardware, setting the stack pointer, scrolling the Nintendo logo, etc. Once the boot ROM has completed its tasks,
it is unmapped from memory, and the cartridge is mapped over the boot ROM, thus starting the cartridge execution, 
and preventing the boot ROM from being executed again until the Game Boy is reset.

## How it works

To create a new boot ROM, use the `NewBootROM` function, it accepts the following arguments:

- `raw`: a byte slice containing the boot ROM data
- `md5Checksum`: a string containing the expected MD5 checksum of the raw data

The function will then validate the provided boot ROM data by checking its length (which should be 256 or 2304 bytes) 
and comparing its checksum against the provided `md5Checksum`. If the checksum does not match, or the length is not valid 
the function will panic.

Once the boot ROM data has been validated, the NewBootROM function creates a new ROM struct and assigns the raw data to 
its raw field.

The created ROM struct can then be used as an implementation of the `mmu.IOBus` interface, providing a `Read` method that 
returns the byte at a given address, and a `Write` method that panics as the boot ROM is read-only.

## Example

```go
package main

import (
	"github.com/thelolagemann/gomeboy/boot"
)

func main() {
	// loading the DMG boot rom
	bootRom := boot.NewBootROM(boot.DMGBootROM[:], boot.DMGBootROMChecksum)
	
	// loading the CGB boot rom
	cgbBootRom := boot.NewBootROM(boot.CGBBootROM[:], boot.CGBBootROMChecksum)
}
```

In this example, the package is imported, and NewBootROM function is called twice, first to load the DMG boot ROM and 
then to load the CGB boot ROM. The function takes the DMGBootROM and CGBBootROM constants as the raw arguments and the 
corresponding DMGBootROMChecksum and CGBBootROMChecksum constants as the md5Checksum arguments, respectively. If the 
validation checks pass, the function returns a new ROM struct that can be used as an implementation of the mmu.IOBus 
interface.