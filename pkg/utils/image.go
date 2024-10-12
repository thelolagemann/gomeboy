//go:build !test

package utils

import (
	"bytes"
	"golang.design/x/clipboard"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"math"
)

func CopyImage(img image.Image) error {
	err := clipboard.Init()
	if err != nil {
		return err
	}

	// encode image to byte slice
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return err
	}

	clipboard.Write(clipboard.FmtImage, b.Bytes())
	return nil
}

// ForPixel iterates over every pixel coordinate in the range of width * height, calling f
// for each coordinate.
func ForPixel(width, height int, f func(x, y int)) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			f(x, y)
		}
	}
}

type point struct {
	X, Y, Z float64
}

// Rotate2DFrame rotates a 2D framebuffer in 3D space with perspective correction.
func Rotate2DFrame(frame *[144][160][3]uint8, angleX, angleY float64) {
	angleX /= 64
	angleY /= 64

	var rotatedFrame [144][160][3]uint8

	// Define the rotation matrices.
	rotateX := func(p point, angleX float64) point {
		sinX, cosX := math.Sin(angleX), math.Cos(angleX)
		return point{
			X: p.X,
			Y: p.Y*cosX - p.Z*sinX,
			Z: p.Y*sinX + p.Z*cosX,
		}
	}

	rotateY := func(p point, angleY float64) point {
		sinY, cosY := math.Sin(angleY), math.Cos(angleY)
		return point{
			X: p.X*cosY + p.Z*sinY,
			Y: p.Y,
			Z: -p.X*sinY + p.Z*cosY,
		}
	}

	// Define the viewer's position.
	viewer := point{X: 0, Y: 0, Z: -10}

	// Iterate over each pixel in the framebuffer.
	for y := 0; y < 144; y++ {
		for x := 0; x < 160; x++ {
			// Define the point in 3D space corresponding to the pixel.
			point := point{X: float64(x - 160/2), Y: float64(y - 144/2), Z: 0}

			// Apply rotations around X, Y, and Z axes.
			point = rotateX(point, angleX)
			point = rotateY(point, angleY)

			// Apply perspective correction.
			scale := viewer.Z / (viewer.Z + point.Z)
			projectedX := int((point.X*scale + viewer.X) + 160/2)
			projectedY := int((point.Y*scale + viewer.Y) + 144/2)

			// Check if the projected point is within the bounds of the framebuffer.
			if projectedX >= 0 && projectedX < 160 && projectedY >= 0 && projectedY < 144 {
				// Copy the color of the pixel from the original frame to the rotated frame.
				rotatedFrame[y][x] = frame[projectedY][projectedX]
			}
		}
	}

	*frame = rotatedFrame
}

// ShakeFrame shakes the given frame in place, using the offset to apply an oscillation.
func ShakeFrame(frame *[144][160][3]uint8, offset int) {
	// Create a temporary frame to store the result
	var tempFrame [144][160][3]uint8
	for y := 0; y < 144; y++ {
		for x := 0; x < 160; x++ {
			tempFrame[y][x] = frame[y][x]
		}
	}

	const amplitude, frequency = 2.0, 0
	var phase = float64(offset)

	// Calculate the offset based on sine function
	offsetX := func(t float64) int {
		return int(amplitude * math.Sin(2*math.Pi*frequency*t+phase))
	}

	// Apply the oscillating offset
	for y := 0; y < 144; y++ {
		for x := 0; x < 160; x++ {
			// Calculate the time component based on the current x position
			t := float64(x) / float64(160)

			// Calculate the offset
			offset := offsetX(t)

			// Apply the offset, ensuring it stays within bounds
			newX := x + offset
			if newX >= 0 && newX < 160 {
				tempFrame[y][newX] = frame[y][x]
			}
		}
	}

	// Copy the result back to the original frame
	*frame = tempFrame
}

// ResizeWithAspectRatio resizes the input image to fit within the target width and height while maintaining aspect ratio,
// filling the empty space with black if needed.
func ResizeWithAspectRatio(img image.Image, targetWidth, targetHeight int) image.Image {
	// Get original image dimensions
	origWidth := img.Bounds().Dx()
	origHeight := img.Bounds().Dy()

	// Calculate the scale factor to maintain aspect ratio
	widthRatio := float64(targetWidth) / float64(origWidth)
	heightRatio := float64(targetHeight) / float64(origHeight)
	scale := widthRatio
	if heightRatio < widthRatio {
		scale = heightRatio
	}

	// Calculate new dimensions based on the scale factor
	newWidth := int(float64(origWidth) * scale)
	newHeight := int(float64(origHeight) * scale)

	// Create a black-filled canvas with the target size
	output := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(output, output.Bounds(), &image.Uniform{black}, image.Point{}, draw.Src)

	// Resize the image while maintaining the aspect ratio
	resizedImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Calculate the offset to center the resized image on the canvas
	offsetX := (targetWidth - newWidth) / 2
	offsetY := (targetHeight - newHeight) / 2

	// Draw the resized image onto the black canvas
	draw.Draw(output, image.Rect(offsetX, offsetY, offsetX+newWidth, offsetY+newHeight), resizedImg, image.Point{}, draw.Over)

	return output
}
