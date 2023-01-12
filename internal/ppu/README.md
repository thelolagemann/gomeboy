# PPU (Pixel Processing Unit) Emulation

The PPU is the main graphics processor of the Game Boy. It is responsible for
displaying the graphics on the screen. The PPU is a complex piece of hardware,
with many components. 

### Hardware Registers

The PPU has several registers mapped to memory addresses in the range `0xFF40` to
`0xFF4B`. The registers are laid out as follows:

| Address | Name | Description                    |
|---------|------|--------------------------------|
| 0xFF40  | LCDC | LCD Control                    |
| 0xFF41  | STAT | LCD Status                     |
| 0xFF42  | SCY  | Scroll Y                       |
| 0xFF43  | SCX  | Scroll X                       |
| 0xFF44  | LY   | LCDC Y-Coordinate              |
| 0xFF45  | LYC  | LY Compare                     |
| 0xFF46  | DMA  | DMA Transfer and Start Address |
| 0xFF47  | BGP  | BG Palette Data                |
| 0xFF48  | OBP0 | Object Palette 0 Data          |
| 0xFF49  | OBP1 | Object Palette 1 Data          |
| 0xFF4A  | WY   | Window Y Position              |
| 0xFF4B  | WX   | Window X Position              |


For the sake of simplicity, the PPU is broken down into several packages, each
of which is responsible for a different part of the PPU. The PPU is broken 
down into the following packages:

* `ppu/background` 
* `ppu/lcd` 
* `ppu/palette` 
* `ppu/sprite`
* `ppu/tile` 
* `ppu/window` 

<sup>More information about each of these packages and their responsibilities with 
regard to the PPU can be found in their respective README files.</sup>

With the main PPU logic broken down into several packages, the PPU
can be implemented in a modular fashion. This allows for the PPU to be
implemented in a way that is easy to understand and maintain.

## How it works

## How it's implemented

The PPU is implemented as a struct that implements the `mmu.IOBus` interface.
This allows the MMU to dispatch read and write calls to the PPU, without
knowing anything about the implementation details of the PPU. 