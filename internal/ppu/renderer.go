package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

const ScanlineSize = 63
const TileSizeInBytes = 3
const SpriteSizeInBytes = 4

// Pixel represents a single pixel on the screen. The PPU
// renders the screen with 2bit color depth, so that a single byte
// can represent 4 pixels. With the custom rendering logic in place,
// the pixels are encoded with additional information about the pixel
// being rendered, such as if it is a sprite or a background pixel.
// Therefore, a pixel of the Game Boy screen is represented as a
// single byte. A Renderer is used to convert the pixel data into
// a format that can be displayed on the screen.
//
// The pixel data is stored in the following format:
//
//	Bit 6 - 4 Color Palette Number  **CGB mode Only**    (OBP0-7)
//	Bit 3 - OBJ Palette Number *Non CGB mode Only** (0=OBP0, 1=OBP1)
//	Bit 2 - Background/Sprite Pixel (0=Background, 1=Sprite)
//	Bit 1 - 0 Color number  (0-3)
//
// New Format 3 bytes per 8 pixels
//
//	Bit 23 - 20 Color Palette Number  **CGB mode Only**    (OBP0-7)
//	Bit 19 - OBJ Palette Number *Non CGB mode Only** (0=OBP0, 1=OBP1)
//	Bit 18 - Background/Sprite Pixel (0=Background, 1=Sprite)
//	Bit 15 - 0 Color Number for pixels 1 - 8 (00-03)
type Pixel = uint8

// RenderScanlineDMG renders the given pixel data into a format that can be
// displayed on the screen. The pixel data is a slice of bytes
// that represent the pixels on the screen.
func RenderScanlineDMG(job RenderJob, scanline *[160][3]uint8) {
	spriteXPerScreen := [ScreenWidth]uint8{}
	tileOffset := job.XStart % 8
	b1 := job.Scanline[0]
	b2 := job.Scanline[1]

	currentTile := 0

	// iterate over the pixels in the scanline
	for x := uint8(0); x < ScreenWidth; x++ {
		// have we reached the end of the current tile?
		if x != 0 && (x+tileOffset)%8 == 0 {
			currentTile++
			// load the next tile
			b1 = job.Scanline[currentTile*TileSizeInBytes]
			b2 = job.Scanline[currentTile*TileSizeInBytes+1]
			//tileInfo = job.Scanline[currentTile*TileSizeInBytes+2]
		}

		// get the pixel data (bit x of b1 and b2)
		low := 0
		high := 0

		// which bit of the byte do we need to read?
		bitIndex := uint8(1 << (7 - ((x + tileOffset) % 8)))
		if b1&bitIndex != 0 {
			low = 1
		}
		if b2&bitIndex != 0 {
			high = 2
		}

		scanline[x] = job.palettes[0].GetColour(uint8(low + high))
	}

	// draw sprites
	for i := 0; i < (len(job.Sprites))/SpriteSizeInBytes; i++ {
		// where does the sprite start on the screen?
		startX := job.Sprites[i*SpriteSizeInBytes+3]

		for x := uint8(0); x < 8; x++ {
			// get the pixel data
			low := 0
			high := 0

			// which bit of the byte do we need to read?
			bitIndex := uint8(1 << (7 - x))

			// are we flipping the sprite?
			if job.Sprites[i*SpriteSizeInBytes+2]&types.Bit5 == types.Bit5 {
				bitIndex = uint8(1 << x)
			}

			if job.Sprites[i*SpriteSizeInBytes]&bitIndex != 0 {
				low = 1
			}

			if job.Sprites[i*SpriteSizeInBytes+1]&bitIndex != 0 {
				high = 2
			}
			colourNum := uint8(low + high)

			if colourNum == 0 {
				continue // transparent
			}

			// is the sprite out of bounds?
			if startX+x >= ScreenWidth {
				break
			}

			if !(job.Sprites[i*SpriteSizeInBytes+2]&types.Bit7 == 0 && !(job.Scanline[((tileOffset+startX+x)/8)*TileSizeInBytes+2]&types.Bit7 == types.Bit7)) {
				// we need to determine which palette number that the background tile is using
				// we can do this by looking at the tile info byte
				// first we need to determine which of the 20/21 tiles the pixel is in

				if scanline[startX+x] != job.palettes[0].GetColour(0) {
					continue
				}
			}

			// skip if occupied by sprite with lower x coordinate
			if spriteXPerScreen[startX+x] != 0 && spriteXPerScreen[startX+x] <= startX {
				continue
			}

			scanline[startX+x] = job.objPalette[job.Sprites[i*SpriteSizeInBytes+2]>>4&0x1].GetColour(colourNum)

			// mark the pixel as occupied by a sprite
			spriteXPerScreen[startX+x] = startX
		}
	}
}

func RenderScanlineCGB(job RenderJob, scanline *[160][3]uint8) {
	spriteXPerScreen := [ScreenWidth]uint8{}
	tileOffset := job.XStart % 8

	// load the inital data
	b1 := job.Scanline[0]
	b2 := job.Scanline[1]
	tileInfo := job.Scanline[2]
	currentTile := 0

	// iterate over the pixels in the scanline
	for x := uint8(0); x < ScreenWidth; x++ {
		// have we reached the end of the tile?
		if x != 0 && (x+tileOffset)%8 == 0 {
			currentTile++
			// load the next tile
			b1 = job.Scanline[currentTile*TileSizeInBytes]
			b2 = job.Scanline[currentTile*TileSizeInBytes+1]
			tileInfo = job.Scanline[currentTile*TileSizeInBytes+2]
		}
		// get the pixel data (bit x of b1 and b2)
		low := 0
		high := 0

		// determine the bit to get according to the tile offset
		bitIndex := uint8(1 << (7 - ((x + tileOffset) % 8)))

		// are we flipping the tile horizontally?
		if tileInfo&types.Bit5 != 0 {
			bitIndex = uint8(1 << ((x + tileOffset) % 8))
		}

		if b1&bitIndex != 0 {
			low = 1
		}

		if b2&bitIndex != 0 {
			high = 2
		}
		colorNum := low + high

		scanline[x] = job.palettes[(tileInfo&0x07)+1].GetColour(uint8(colorNum))
	}
	// get palette index of the background tile
	// draw sprites
	for i := 0; i < (len(job.Sprites))/SpriteSizeInBytes; i++ {
		// where to start drawing the sprite
		startX := job.Sprites[i*SpriteSizeInBytes+3]

		for x := uint8(0); x < 8; x++ {
			// get the pixel data (bit x of b1 and b2)
			low := 0
			high := 0

			// determine the bit to get according to the tile offset
			bitIndex := uint8(1 << (7 - x))

			// are we flipping the tile horizontally?
			if job.Sprites[i*SpriteSizeInBytes+2]&types.Bit5 != 0 {
				bitIndex = uint8(1 << x)
			}

			if job.Sprites[i*SpriteSizeInBytes]&bitIndex != 0 {
				low = 1
			}

			if job.Sprites[i*SpriteSizeInBytes+1]&bitIndex != 0 {
				high = 2
			}

			colorNum := low + high
			if colorNum == 0 {
				continue
			}

			// is the sprite out of bounds?
			if startX+x >= ScreenWidth {
				break
			}

			// determine priority
			if job.BackgroundEnabled {
				if !(job.Sprites[i*SpriteSizeInBytes+2]&types.Bit7 == 0 && !(job.Scanline[((tileOffset+startX+x)/8)*TileSizeInBytes+2]&types.Bit7 == types.Bit7)) {
					// we need to determine which palette number that the background tile is using
					// we can do this by looking at the tile info byte
					// first we need to determine which of the 20/21 tiles the pixel is in
					tileNum := (tileOffset + startX + x) / 8 // 0-21

					// then we need to get the tile info byte
					eTileInfo := job.Scanline[tileNum*TileSizeInBytes+2]

					if scanline[startX+x] != job.palettes[(eTileInfo&0x07)+1].GetColour(0) {
						continue
					}
				}
			}

			// is the pixel occupied by a sprite already?
			if spriteXPerScreen[startX+x] != 0 {
				continue
			}

			palNumber := job.Sprites[i*SpriteSizeInBytes+2] & 0x07
			scanline[startX+x] = job.objPalette[palNumber+2].GetColour(uint8(colorNum))

			// mark pixel as occupied by sprite
			spriteXPerScreen[startX+x] = startX
		}
	}
}

type RenderJob struct {
	XStart            uint8
	Sprites           [40]uint8
	Scanline          [ScanlineSize]Pixel
	Line              uint8
	palettes          [9]palette.Palette
	objPalette        [10]palette.Palette
	BackgroundEnabled bool
}

type RenderOutput struct {
	Scanline [ScreenWidth][3]uint8
	Line     uint8
}

// Renderer is used to render the pixel data into a format that can be
// displayed on the screen.
type Renderer struct {
	// jobs is a channel that receives jobs from the PPU.
	jobs   chan RenderJob
	output chan<- *RenderOutput

	renderMode bool
}

// NewRenderer creates a new Renderer.
func NewRenderer(jobs chan RenderJob, output chan<- *RenderOutput) *Renderer {
	r := &Renderer{
		jobs:   jobs,
		output: output,
	}

	// start a few goroutines to render the scanlines
	for i := 0; i < 16; i++ {
		go r.RenderScanline(jobs, output)
	}

	return r
}

func (r *Renderer) RenderScanline(jobs <-chan RenderJob, output chan<- *RenderOutput) {

}

func (r *Renderer) AddJob(job RenderJob) {
	r.jobs <- job
}
