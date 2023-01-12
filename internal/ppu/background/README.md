# Background

The background package provides an implementation of the background layer for the Game Boy's PPU. 

### Hardware registers

The background rendering is controlled by the following registers:

| Address | Name | Description             |
|---------|------|-------------------------|
| 0xFF42  | SCY  | Scroll Y                |
| 0xFF43  | SCX  | Scroll X                |
| 0xFF47  | BGP  | Background palette data |

#### SCY (Scroll Y)

The SCY register controls the vertical scroll position of the background. 

#### SCX (Scroll X)

The SCX register controls the horizontal scroll position of the background. 

#### BGP (Background palette data)

The BGP register controls the background palette. The background palette is a 2-bit value for each of the 4 colors in 
the palette. The value is stored in the following format:

| Bit 1 | Bit 0 | Color      |
|-------|-------|------------|
| 0     | 0     | White      |
| 0     | 1     | Light gray |
| 1     | 0     | Dark gray  |
| 1     | 1     | Black      |

<sup>The colours are just a suggestion, the actual colours depend on how the emulator chooses to implement them. See the
[Palette](../palette/README.md) package for more information.</sup>

### How it works



### How it's implemented