package io

import (
	"bytes"
	"github.com/nfnt/resize"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/vladimirvivien/go4vl/device"
	"image"
	"image/color"
	"image/jpeg"
	"unsafe"
)

const (
	CAMERA_SHOOT = iota
	CAMERA_NVH
	CAMERA_C0
	CAMERA_C1
	CAMERA_EVI

	CAMERA_SENSOR_W = 128
	CAMERA_SENSOR_H = 120
	CAMERA_W        = 128
	CAMERA_H        = 112

	CAMERA_TILE_SIZE = 14 * 16 * 16
)

type Camera struct {
	CameraShooter

	sensorImage     [CAMERA_SENSOR_W][CAMERA_SENSOR_H]int
	registers       [0x36]uint8
	registersMapped bool
}

type CameraShooter func() image.Image

func clamp(min, value, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (c *Cartridge) readCameraRAM(addr uint16) uint8 {
	if c.camera.registersMapped {
		if addr&0x7f == CAMERA_SHOOT {
			return c.camera.registers[CAMERA_SHOOT]
		}
		return 0
	}

	if c.camera.registers[CAMERA_SHOOT]&1 == 1 {
		return 0
	}

	return c.RAM[c.ramOffset+uint32(addr)&0x1fff]

}

func (c *Cartridge) writeCameraRAM(addr uint16, value uint8) {
	if c.camera.registersMapped {
		addr &= 0x7f
		if addr == CAMERA_SHOOT {
			value &= 7 // only 3 bits

			if value&1 > 0 && c.camera.registers[CAMERA_SHOOT]&1 == 0 {
				c.startShooting()
			}

			if value&1 == 0 && c.camera.registers[CAMERA_EVI]&1 == 1 {
				value |= 1
				panic("dont support cancelling")
			}
			c.camera.registers[CAMERA_SHOOT] = value
		} else {
			if addr >= 0x36 {
				return // invalid
			}
			c.camera.registers[addr] = value
		}

		return
	}

	if c.ramEnabled {
		c.RAM[c.ramOffset+uint32(addr)&0x1fff] = value
	}
}

func (c *Cartridge) updateCamera() {
	c.camera.registers[CAMERA_SHOOT] &^= 1
}

func (c *Cartridge) startShooting() {
	f := <-dev.GetOutput()
	img, err := jpeg.Decode(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
	c.camera.image = resize.Resize(CAMERA_SENSOR_W, CAMERA_SENSOR_H, convertToGray(img), resize.NearestNeighbor)

	c.camera.sensorImage = [CAMERA_SENSOR_W][CAMERA_SENSOR_H]int{}
	var p, m uint8
	switch (c.camera.registers[CAMERA_SHOOT] >> 1) & 3 {
	case 0:
		m = 0
	case 1:
		p = 1
	case 2, 3:
		p = 1
		m = 2
	}

	// register 1
	n := (c.camera.registers[CAMERA_NVH] & types.Bit7) >> 7
	vh := c.camera.registers[CAMERA_NVH] & 0x60 >> 5

	// register 2 & 3
	exposure := uint16(c.camera.registers[CAMERA_C1]) | uint16(c.camera.registers[CAMERA_C0])<<8

	// register 4
	edge := []float64{0.50, 0.75, 1.00, 1.25, 2.00, 3.00, 4.00, 5.00}[(c.camera.registers[CAMERA_EVI]&0x70)>>4]
	e3 := c.camera.registers[CAMERA_EVI] & types.Bit7 >> 7
	i := c.camera.registers[CAMERA_EVI]&types.Bit3 > 0

	// calculate how long the shoot should take
	cameraClocks := uint64(4 * (32446 + uint64(n)*512 + uint64(exposure)*16))
	c.b.s.ScheduleEvent(scheduler.CameraShoot, cameraClocks)

	// copy webcam image to sensor buffer
	for j := 0; j < CAMERA_SENSOR_W; j++ {
		for k := 0; k < CAMERA_SENSOR_H; k++ {
			value := int(c.camera.image.(*image.Gray).Pix[k*c.camera.image.(*image.Gray).Stride+j])
			value = (value * int(exposure)) / 0x0300

			value = 128 + (((value - 128) * 1) / 8)
			c.camera.sensorImage[j][k] = int(value)
		}
	}

	// handle inverting
	if i {
		for j := 0; j < CAMERA_SENSOR_W; j++ {
			for k := 0; k < CAMERA_SENSOR_H; k++ {
				c.camera.sensorImage[j][k] = 255 - c.camera.sensorImage[j][k]
			}
		}
	}

	// make signed
	for j := 0; j < CAMERA_SENSOR_W; j++ {
		for k := 0; k < CAMERA_SENSOR_H; k++ {
			c.camera.sensorImage[j][k] = c.camera.sensorImage[j][k] - 128
		}
	}

	tempBuf := [CAMERA_SENSOR_W][CAMERA_SENSOR_H]int{}
	filteringMode := (n << 3) | (vh << 1) | e3
	switch filteringMode {
	case 0: // 1D filtering
		for j := 0; j < CAMERA_SENSOR_W; j++ {
			for k := 0; k < CAMERA_SENSOR_H; k++ {
				tempBuf[j][k] = c.camera.sensorImage[j][k]
			}
		}

		for j := 0; j < CAMERA_SENSOR_W; j++ {
			for k := 0; k < CAMERA_SENSOR_H; k++ {
				ms := tempBuf[j][min(k+1, CAMERA_SENSOR_H-1)]
				px := tempBuf[j][k]

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

				c.camera.sensorImage[j][k] = clamp(-128, value, 127)
			}
		}
	case 2: // 1D filtering + Horiz. Enhancement
		for j := 0; j < CAMERA_SENSOR_W; j++ {
			for k := 0; k < CAMERA_SENSOR_H; k++ {
				mw := c.camera.sensorImage[max(0, j-1)][k]
				me := c.camera.sensorImage[min(j+1, CAMERA_SENSOR_W-1)][k]
				px := c.camera.sensorImage[j][k]
				tempBuf[j][k] = clamp(0, px+(int(float64(2*px-mw-me)*edge)), 255)
			}
		}

		for j := 0; j < CAMERA_SENSOR_W; j++ {
			for k := 0; k < CAMERA_SENSOR_H; k++ {
				ms := tempBuf[j][min(k+1, CAMERA_SENSOR_H-1)]
				px := c.camera.sensorImage[j][k]

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
				c.camera.sensorImage[j][k] = clamp(-128, value, 127)
			}
		}
	}

	// make unsigned
	for j := 0; j < CAMERA_SENSOR_W; j++ {
		for k := 0; k < CAMERA_SENSOR_H; k++ {
			c.camera.sensorImage[j][k] = c.camera.sensorImage[j][k] + 128
		}
	}

	colorBuffer := [CAMERA_W][CAMERA_H]int{}
	for j := 0; j < CAMERA_W; j++ {
		for k := 0; k < CAMERA_H; k++ {
			colorBuffer[j][k] = c.processCameraMatrix(c.camera.sensorImage[j][k+4], j, k)
		}
	}

	finalBuffer := [14][16][16]uint8{}
	for j := 0; j < CAMERA_W; j++ {
		for k := 0; k < CAMERA_H; k++ {
			outColor := 3 - (colorBuffer[j][k] >> 6)

			tileBaseIndex := (k & 7) << 1
			tileBase := finalBuffer[k>>3][j>>3][tileBaseIndex:]

			if outColor&1 > 0 {
				tileBase[0] |= 1 << (7 - 7&j)
			}
			if outColor&2 > 0 {
				tileBase[1] |= 1 << (7 - 7&j)
			}

			copy(finalBuffer[k>>3][j>>3][tileBaseIndex:], tileBase)
		}
	}

	copy(c.RAM[0x0100:], (*[CAMERA_TILE_SIZE]uint8)(unsafe.Pointer(&finalBuffer[0][0][0]))[:CAMERA_TILE_SIZE:CAMERA_TILE_SIZE])
	copy(c.b.data[0xa100:], (*[CAMERA_TILE_SIZE]uint8)(unsafe.Pointer(&finalBuffer[0][0][0]))[:CAMERA_TILE_SIZE:CAMERA_TILE_SIZE])
}

func (c *Cartridge) processCameraMatrix(value, x, y int) int {
	x &= 3
	y &= 3

	base := 6 + (y*4+x)*3
	r0 := int(c.camera.registers[base])
	r1 := int(c.camera.registers[base+1])
	r2 := int(c.camera.registers[base+2])

	if value < r0 {
		return 0
	} else if value < r1 {
		return 0x40
	} else if value < r2 {
		return 0x80
	}

	return 0xc0
}

var (
	dev *device.Device
)

func convertToGray(img image.Image) *image.Gray {
	// If the image is already grayscale, return it as is
	if grayImg, ok := img.(*image.Gray); ok {
		return grayImg
	}

	// Otherwise, convert it to grayscale
	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the pixel color and convert to grayscale
			originalColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(originalColor).(color.Gray)
			gray.Set(x, y, grayColor)
		}
	}
	return gray
}
