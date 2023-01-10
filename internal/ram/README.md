# RAM

RAM is a simple package that provides memory backed RAM for several components
used throughout the Game Boy. It has just one exported struct, `RAM`, which
also implements the `mmu.IOBus` interface. RAM is created simply by calling
`NewRAM` with the size of the RAM to create.

## How it's implemented

The RAM is implemented as a struct that implements the `mmu.IOBus` interface. This
allows the RAM to be used as would any other IO device. The RAM is backed by a
`[]byte` slice, which is initialized to all 0s. The `Read` and `Write` methods
are implemented to read and write to the slice. Both the `Read` and `Write` methods
check to make sure that the address is within the bounds of the RAM. If it is not,
a panic is thrown. 

