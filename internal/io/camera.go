package io

import (
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"image"
	"image/draw"
	"math/rand/v2"
	"unsafe"
)

const (
	CameraShoot               = iota
	CameraNVH                 // output gain and edge operation mode
	CameraC0                  // exposure MSB
	CameraC1                  // exposure LSB
	CameraEVI                 // output voltage reference, edge enhancement ratio & invert
	CameraDitherContrastStart = 6
	CameraDitherContrastEnd   = 0x35

	CameraSensorW = 128
	CameraSensorH = 120
	CameraW       = 128
	CameraH       = 112

	CameraTileSize = 14 << 8

	Filter1D         = 0
	Filter1DAndHoriz = 2
	Filter2D         = 14
)

// Camera implements the functionality of the [POCKETCAMERA] cartridge, allowing
// for any [image.Image] to be used as the camera's source.
type Camera struct {
	CameraShooter

	Filter1D, Filter1DAndHoriz, Filter2D bool // disable/enable filtering modes

	ShotImage       image.Image                       // the image that comes in from CameraShooter
	SensedImage     [CameraSensorW][CameraSensorH]int // the image that the sensor "sees"
	Registers       [0x36]uint8
	registersMapped bool
	sensorImage     *image.Gray
}

// CameraShooter mimics a "camera" attached to the emulator, by simply returning
// an [image.Image]. This allows for any image source to be used as a camera input.
type CameraShooter func() image.Image

// NoiseShooter is a simple [CameraShooter] that returns random noise.
func NoiseShooter(w, h int) func() image.Image {
	f := image.NewGray(image.Rect(0, 0, w, h))
	return func() image.Image {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				f.Pix[y*w+x] = rand.N(uint8(255))
			}
		}
		return f
	}
}

func (c *Cartridge) readCameraRAM(addr uint16) uint8 {
	if c.Camera.registersMapped {
		if addr&0x7f == CameraShoot {
			return c.Camera.Registers[CameraShoot]
		}
		return 0 // only CameraShoot register is readable
	}

	if c.Camera.Registers[CameraShoot]&1 == 1 {
		return 0 // ram is unreadable whilst the camera is shooting
	}

	return c.RAM[c.ramOffset+uint32(addr)&0x1fff]
}

// startShooting is called when a value with [types.Bit0] set is written to [CameraShoot]. It
// processes the image from the embedded [CameraShooter] and converts it into the gb tile format,
// as well as calculating how many clocks the shoot would take on real hardware according to the
// formula provided at https://gbdev.io/pandocs/Gameboy_Camera.html#sample-code-for-emulators
//
//	(32446 + (n*512) + (exposure*16)) * 4
func (c *Cartridge) startShooting() {
	c.Camera.ShotImage = c.Camera.CameraShooter()

	tempImg := utils.ResizeWithAspectRatio(c.Camera.ShotImage, CameraSensorW, CameraSensorH)
	draw.Draw(c.Camera.sensorImage, c.Camera.sensorImage.Bounds(), tempImg, tempImg.Bounds().Min, draw.Src)

	c.Camera.SensedImage = [CameraSensorW][CameraSensorH]int{}
	var p, m uint8
	switch (c.Camera.Registers[CameraShoot] >> 1) & 3 {
	case 0:
		m = 1
	case 1:
		p = 1
	case 2, 3:
		p = 1
		m = 2
	}

	// register 1
	n := (c.Camera.Registers[CameraNVH] & types.Bit7) >> 7
	vh := c.Camera.Registers[CameraNVH] & 0x60 >> 5

	// register 2 & 3
	exposure := uint16(c.Camera.Registers[CameraC1]) | uint16(c.Camera.Registers[CameraC0])<<8

	// register 4
	edge := []float64{0.50, 0.75, 1.00, 1.25, 2.00, 3.00, 4.00, 5.00}[(c.Camera.Registers[CameraEVI]&0x70)>>4]
	e3 := c.Camera.Registers[CameraEVI] >> 7
	i := c.Camera.Registers[CameraEVI]&types.Bit3 > 0

	// calculate how long the shoot should take
	c.b.s.ScheduleEvent(scheduler.CameraShoot, 2<<(32446+uint64(n)<<9+uint64(exposure)<<4))

	utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) {
		value := int(c.Camera.sensorImage.Pix[y*c.Camera.sensorImage.Stride+x])
		value = (value * int(exposure)) / 0x0300

		value = 128 + (((value - 128) * 1) / 8)
		c.Camera.SensedImage[x][y] = value
	})

	// handle inverting
	if i {
		utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) { c.Camera.SensedImage[x][y] = 255 - c.Camera.SensedImage[x][y] })
	}

	utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) { c.Camera.SensedImage[x][y] = c.Camera.SensedImage[x][y] - 128 })

	tempBuf := [CameraSensorW][CameraSensorH]int{}
	switch (n << 3) | (vh << 1) | e3 {
	case Filter1D: // 1D filtering
		if !c.Camera.Filter1D {
			break
		}
		copy(tempBuf[:], c.Camera.SensedImage[:])

		utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) {
			ms := tempBuf[x][min(y+1, CameraSensorH-1)]
			px := tempBuf[x][y]

			value := 0
			if p&types.Bit0 > 0 {
				value += px
			}
			if p&types.Bit1 > 0 {
				value += ms
			}
			if m&types.Bit0 > 0 {
				value -= px
			}
			if m&types.Bit1 > 0 {
				value -= ms
			}

			c.Camera.SensedImage[x][y] = utils.Clamp(-128, value, 127)
		})
	case 1:
		c.Camera.SensedImage = [CameraSensorW][CameraSensorH]int{}
	case Filter1DAndHoriz: // 1D filtering + Horiz. Enhancement
		if !c.Camera.Filter1DAndHoriz {
			break
		}
		utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) {
			mw := c.Camera.SensedImage[max(0, x-1)][y]
			me := c.Camera.SensedImage[min(x+1, CameraSensorW-1)][y]
			px := c.Camera.SensedImage[x][y]
			tempBuf[x][y] = utils.Clamp(0, px+(int(float64(2*px-mw-me)*edge)), 255)
		})
		utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) {
			ms := tempBuf[x][min(y+1, CameraSensorH-1)]
			px := c.Camera.SensedImage[x][y]

			value := 0
			if p&types.Bit0 > 0 {
				value += px
			}
			if p&types.Bit1 > 0 {
				value += ms
			}
			if m&types.Bit0 > 0 {
				value -= px
			}
			if m&types.Bit1 > 0 {
				value -= ms
			}
			c.Camera.SensedImage[x][y] = utils.Clamp(-128, value, 127)
		})
	case Filter2D: // 2D enhancement
		if !c.Camera.Filter2D {
			break
		}
		utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) {
			ms := c.Camera.SensedImage[x][min(y+1, CameraSensorH-1)]
			mn := c.Camera.SensedImage[x][max(0, y-1)]
			mw := c.Camera.SensedImage[max(0, x-1)][y]
			me := c.Camera.SensedImage[min(x+1, CameraSensorW-1)][y]
			px := c.Camera.SensedImage[x][y]

			tempBuf[x][y] = utils.Clamp(-128, px+(int(float64(4*px-mw-me-mn-ms)*edge)), 127)
		})
		copy(c.Camera.SensedImage[:], tempBuf[:])
	}

	utils.ForPixel(CameraSensorW, CameraSensorH, func(x, y int) { c.Camera.SensedImage[x][y] = c.Camera.SensedImage[x][y] + 128 })

	colorBuffer := [CameraW][CameraH]int{}
	utils.ForPixel(CameraW, CameraH, func(x, y int) { colorBuffer[x][y] = c.Camera.processCameraMatrix(c.Camera.SensedImage[x][y+4], x, y) })

	finalBuffer := [14][16][16]uint8{}
	utils.ForPixel(CameraW, CameraH, func(x, y int) {
		outColor := 3 - (colorBuffer[x][y] >> 6)

		tileBaseIndex := (y & 7) << 1
		tileBase := finalBuffer[y>>3][x>>3][tileBaseIndex:]

		if outColor&1 > 0 {
			tileBase[0] |= 1 << (7 - 7&x)
		}
		if outColor&2 > 0 {
			tileBase[1] |= 1 << (7 - 7&x)
		}

		copy(finalBuffer[y>>3][x>>3][tileBaseIndex:], tileBase)
	})

	copy(c.RAM[0x0100:], (*[CameraTileSize]uint8)(unsafe.Pointer(&finalBuffer[0][0][0]))[:CameraTileSize:CameraTileSize])
	copy(c.b.data[0xa100:], (*[CameraTileSize]uint8)(unsafe.Pointer(&finalBuffer[0][0][0]))[:CameraTileSize:CameraTileSize])
}

// processCameraMatrix applies dithering and contrast settings from the registers
// [CameraDitherContrastStart] to [CameraDitherContrastEnd]
func (c *Camera) processCameraMatrix(value, x, y int) int {
	x &= 3
	y &= 3

	base := 6 + (y*4+x)*3

	switch {
	case value < int(c.Registers[base]):
		return 0
	case value < int(c.Registers[base+1]):
		return 0x40
	case value < int(c.Registers[base+2]):
		return 0x80
	default:
		return 0xc0
	}
}
