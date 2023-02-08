package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
)

const ScanlineSize = 63
const TileSizeInBytes = 3

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
//	Bit 6 - 4 Color Palette Number  **CGB Mode Only**    (OBP0-7)
//	Bit 3 - OBJ Palette Number *Non CGB Mode Only** (0=OBP0, 1=OBP1)
//	Bit 2 - Background/Sprite Pixel (0=Background, 1=Sprite)
//	Bit 1 - 0 Color number  (0-3)
//
// New Format 3 bytes per 8 pixels
//
//	Bit 23 - 20 Color Palette Number  **CGB Mode Only**    (OBP0-7)
//	Bit 19 - OBJ Palette Number *Non CGB Mode Only** (0=OBP0, 1=OBP1)
//	Bit 18 - Background/Sprite Pixel (0=Background, 1=Sprite)
//	Bit 15 - 0 Color Number for pixels 1 - 8 (00-03)
type Pixel = uint8

// RenderScanline renders the given pixel data into a format that can be
// displayed on the screen. The pixel data is a slice of bytes
// that represent the pixels on the screen.
//
// TODO merge common functionality with RenderScanlineCGB
func RenderScanline(jobs <-chan RenderJob, output chan<- *RenderOutput) {
	scanline := &RenderOutput{}
	for job := range jobs {
		spriteXPerScreen := [ScreenWidth]uint8{}
		scanline.Line = job.Line
		tileOffset := job.XStart % 8
		b1 := job.Scanline[0]
		b2 := job.Scanline[1]
		//tileInfo := job.Scanline[2]

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

			scanline.Scanline[x] = job.palettes.GetColour(uint8(low + high))
		}

		// draw sprites
		for i := 0; i < (len(job.Sprites))/3; i++ {
			// where does the sprite start on the screen?
			startX := job.SpritePositions[i]

			for x := uint8(0); x < 8; x++ {
				// get the pixel data
				low := 0
				high := 0

				// which bit of the byte do we need to read?
				bitIndex := uint8(1 << (7 - x))

				if job.Sprites[i*3]&bitIndex != 0 {
					low = 1
				}

				if job.Sprites[i*3+1]&bitIndex != 0 {
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

				if !(job.Sprites[i*3+2]&0x01 == 1 && !job.TilePriority[startX+x]) {
					// we need to determine which palette number that the background tile is using
					// we can do this by looking at the tile info byte
					// first we need to determine which of the 20/21 tiles the pixel is in

					if scanline.Scanline[startX+x] != job.palettes.GetColour(0) {
						continue
					}
				}

				// skip if occupied by sprite with lower x coordinate
				if spriteXPerScreen[startX+x] != 0 && spriteXPerScreen[startX+x] <= startX {
					continue
				}

				scanline.Scanline[startX+x] = job.objPalette[job.Sprites[i*3+2]>>3&0x1].GetColour(colourNum)

				// mark the pixel as occupied by a sprite
				spriteXPerScreen[startX+x] = startX
			}
		}
		output <- scanline
	}

}

func RenderScanlineCGB(jobs <-chan RenderJobCGB, output chan<- *RenderOutput) {
	scanline := &RenderOutput{}
	for {
		select {
		case job := <-jobs:
			spriteXPerScreen := [ScreenWidth]uint8{}
			scanline.Line = job.Line
			// determine tile offset

			// example:
			// offset = 3
			// start drawing from tile 0, pixel 3 (tile 0, pixel 0 is the first pixel)
			// when x = 0, draw tile 0, pixel 3
			// when x = 1, draw tile 0, pixel 4
			// when x = 2, draw tile 0, pixel 5
			// when x = 3, draw tile 0, pixel 6
			// when x = 4, draw tile 0, pixel 7
			// when x = 5, update tile, draw tile 1, pixel 0
			// when x = 6, draw tile 1, pixel 1
			// when x = 7, draw tile 1, pixel 2
			// ... and so on
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

				if b1&bitIndex != 0 {
					low = 1
				}

				if b2&bitIndex != 0 {
					high = 2
				}
				colorNum := low + high

				scanline.Scanline[x] = job.palettes.GetColour(tileInfo>>4&0x07, uint8(colorNum))
			}
			// get palette index of the background tile
			// draw sprites
			for i := 0; i < (len(job.Sprites))/3; i++ {
				// where to start drawing the sprite
				startX := job.SpritePositions[i]

				for x := uint8(0); x < 8; x++ {
					// get the pixel data (bit x of b1 and b2)
					low := 0
					high := 0

					// determine the bit to get according to the tile offset
					bitIndex := uint8(1 << (7 - x))

					if job.Sprites[i*3]&bitIndex != 0 {
						low = 1
					}

					if job.Sprites[i*3+1]&bitIndex != 0 {
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
						if !(job.Sprites[i*3+2]&0x01 == 1 && !job.TilePriority[startX+x]) {
							// we need to determine which palette number that the background tile is using
							// we can do this by looking at the tile info byte
							// first we need to determine which of the 20/21 tiles the pixel is in
							tileNum := (tileOffset + startX + x) / 8 // 0-21

							// then we need to get the tile info byte
							eTileInfo := job.Scanline[tileNum*TileSizeInBytes+2]

							if scanline.Scanline[startX+x] != job.palettes.GetColour(eTileInfo>>4&0x07, 0) {
								continue
							}
						}
					}

					// is the pixel occupied by a sprite already?
					if spriteXPerScreen[startX+x] != 0 {
						continue
					}

					scanline.Scanline[startX+x] = job.objPalette.GetColour(job.Sprites[i*3+2]>>4&0x07, uint8(colorNum))

					// mark pixel as occupied by sprite
					spriteXPerScreen[startX+x] = startX
				}
			}

			output <- scanline
		}
	}
}

type RenderJob struct {
	XStart          uint8
	Sprites         [30]uint8
	SpritePositions [10]uint8
	Scanline        [ScanlineSize]Pixel
	TilePriority    [ScreenWidth]bool
	Line            uint8
	palettes        palette.Palette
	objPalette      [2]palette.Palette
}

type RenderOutput struct {
	Scanline [ScreenWidth][3]uint8
	Line     uint8
}

type RenderJobCGB struct {
	XStart            uint8
	Scanline          [ScanlineSize]Pixel
	TilePriority      [ScreenWidth]bool
	BackgroundEnabled bool
	Sprites           [30]uint8
	SpritePositions   [10]uint8
	Line              uint8
	palettes          *palette.CGBPalette
	objPalette        *palette.CGBPalette
}

// Renderer is used to render the pixel data into a format that can be
// displayed on the screen.
type Renderer struct {

	// jobs is a channel that receives jobs from the PPU.
	jobs   chan RenderJob
	output chan<- *RenderOutput
}

// NewRenderer creates a new Renderer.
func NewRenderer(jobs chan RenderJob, output chan<- *RenderOutput) *Renderer {
	r := &Renderer{
		jobs:   jobs,
		output: output,
	}

	// start a few goroutines to render the scanlines
	for i := 0; i < ScreenHeight; i++ {
		go RenderScanline(jobs, output)
	}

	return r
}

type RendererCGB struct {
	jobs   chan RenderJobCGB
	output chan<- *RenderOutput
}

// QueueJob returns instantly and queues the given job to be rendered.
func (r *RendererCGB) QueueJob(job RenderJobCGB) {
	r.jobs <- job
}

func NewRendererCGB(jobs chan RenderJobCGB, output chan<- *RenderOutput) *RendererCGB {
	r := &RendererCGB{
		jobs:   jobs,
		output: output,
	}

	// start a few goroutines to render the scanlines
	for i := 0; i < 16; i++ {
		go RenderScanlineCGB(jobs, output)
	}

	return r
}

func (r *Renderer) AddJob(job RenderJob) {
	r.jobs <- job
}
