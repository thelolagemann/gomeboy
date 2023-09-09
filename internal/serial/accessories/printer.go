package accessories

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/ppu/palette"
	"github.com/thelolagemann/gomeboy/internal/types"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
)

// CommandPosition is the position of the command in the command sequence
type CommandPosition uint8

const (
	// CommandPositionMagic1 is the first magic byte
	CommandPositionMagic1 CommandPosition = 0x00
	// CommandPositionMagic2 is the second magic byte
	CommandPositionMagic2 CommandPosition = 0x01
	// CommandPositionID is the ID of the command
	CommandPositionID CommandPosition = 0x02
	// CommandPositionCompression is the compression flag
	CommandPositionCompression CommandPosition = 0x03
	// CommandPositionLengthLow is the low byte of the length
	CommandPositionLengthLow CommandPosition = 0x04
	// CommandPositionLengthHigh is the high byte of the length
	CommandPositionLengthHigh CommandPosition = 0x05
	// CommandPositionData is the data of the command
	CommandPositionData CommandPosition = 0x06
	// CommandPositionChecksumLow is the low byte of the checksum
	CommandPositionChecksumLow CommandPosition = 0x07
	// CommandPositionChecksumHigh is the high byte of the checksum
	CommandPositionChecksumHigh CommandPosition = 0x08
	// CommandPositionKeepAlive is the keep alive flag
	CommandPositionKeepAlive CommandPosition = 0x09
	// CommandPositionStatus is the status byte
	CommandPositionStatus CommandPosition = 0x0A
)

func (c CommandPosition) String() string {
	switch c {
	case CommandPositionMagic1:
		return "Magic1"
	case CommandPositionMagic2:
		return "Magic2"
	case CommandPositionID:
		return "ID"
	case CommandPositionCompression:
		return "Compression"
	case CommandPositionLengthLow:
		return "LengthLow"
	case CommandPositionLengthHigh:
		return "LengthHigh"
	case CommandPositionData:
		return "Data"
	case CommandPositionChecksumLow:
		return "ChecksumLow"
	case CommandPositionChecksumHigh:
		return "ChecksumHigh"
	case CommandPositionKeepAlive:
		return "KeepAlive"
	case CommandPositionStatus:
		return "Status"
	default:
		return fmt.Sprintf("Unknown(%d)", c)
	}
}

// Command is a command that can be sent to the printer
type Command = uint8

const (
	// CommandInit is the command to initialize the printer
	CommandInit Command = 0x01
	// CommandStart is the command to start printing
	CommandStart Command = 0x02
	// CommandData is the command to send data to the printer
	CommandData Command = 0x04
	// CommandStatus is the command to get the status of the printer
	CommandStatus Command = 0xF
)

type Printer struct {
	byteToSend uint8
	bitToSend  bool

	byteBeingReceived uint8
	counter           uint8
	commandLength     uint16
	lengthLeft        uint16
	position          CommandPosition
	id                Command
	compression       bool
	data              [0x280]byte
	checksum          uint16
	status            uint8
	keepAlive         bool
	imageData         [160 * 144]byte
	imageOffset       int
	timeRemaining     int
	hasJob            bool
	printJob          image.Image
	totalLength       uint16

	stashedImages []color.RGBA
}

func NewPrinter() *Printer {
	return &Printer{}
}

// Send a bit from the printer
func (p *Printer) Send() bool {
	bit := p.byteToSend&types.Bit7 != 0
	p.byteToSend <<= 1
	//fmt.Println("send\t", p.position, bit)
	return bit
}

// Receive a bit to the printer
func (p *Printer) Receive(bit bool) {
	p.byteBeingReceived <<= 1
	if bit {
		p.byteBeingReceived |= types.Bit0
	}

	//fmt.Println("receive\t", p.position, bit)
	if p.counter++; p.counter == 8 {
		p.onReceive(p.byteBeingReceived)
		p.byteBeingReceived = 0
		p.counter = 0
	}
}

// onReceive is called when the printer receives a byte
func (p *Printer) onReceive(b byte) {
	// reset the byte to send
	p.byteToSend = 0

	// decode the position of the command
	switch p.position {
	case CommandPositionMagic1:
		if b != 0x88 && b != 0x10 { // Pokemon Pinball sometimes sends 0x10 instead of 0x88 for some reason
			return
		}
		p.commandLength = 0
		p.lengthLeft = 0
		p.totalLength = 0
		p.position = CommandPositionMagic2
	case CommandPositionMagic2:
		if b != 0x33 {
			// reset
			p.position = CommandPositionMagic1
			return
		}
		p.position = CommandPositionID
	case CommandPositionID:
		// is it a valid command?
		if b != CommandInit && b != CommandStart && b != CommandData && b != CommandStatus {
			p.position = CommandPositionMagic1
		} else {
			p.position = CommandPositionCompression
			p.id = b
		}
	case CommandPositionCompression:
		p.compression = b&types.Bit0 == types.Bit0 // TODO implement compression
		p.position = CommandPositionLengthLow
	case CommandPositionLengthLow:
		p.totalLength = uint16(b)
		p.position = CommandPositionLengthHigh
	case CommandPositionLengthHigh:
		p.totalLength |= uint16(b) << 8
		if p.totalLength == 0 {
			// don't need to receive any data
			p.position = CommandPositionChecksumLow
		} else {
			p.position = CommandPositionData
			p.lengthLeft = p.totalLength
		}
	case CommandPositionData:
		p.data[p.commandLength] = b
		p.commandLength++
		if p.lengthLeft > 0 {
			p.lengthLeft--
		} else {
			p.position = CommandPositionChecksumLow
		}
		if p.commandLength == p.totalLength {
			p.position = CommandPositionChecksumLow
		}
	case CommandPositionChecksumLow:
		p.checksum ^= uint16(b)
		p.position = CommandPositionChecksumHigh
	case CommandPositionChecksumHigh:
		p.checksum ^= uint16(b) << 8
		// TODO verify checksum
		p.byteToSend = 0x81
		p.position = CommandPositionKeepAlive
	case CommandPositionKeepAlive:
		if p.id == CommandInit {
			p.byteToSend = 0
		} else {
			if p.status == 6 {
				p.status = 4
			}
			p.byteToSend = 8
		}
		p.position = CommandPositionStatus
	case CommandPositionStatus:
		if b == 0 {
			p.runCommand(p.id)

			p.position = CommandPositionMagic1
		}
		return
	default:
		panic(fmt.Sprintf("unknown position: %x", p.position))
	}

	if p.position >= CommandPositionID && p.position < CommandPositionChecksumLow {
		p.checksum += uint16(b)
	}
}

// runCommand runs the current command
func (p *Printer) runCommand(cmd Command) {
	switch cmd {
	case CommandInit:
		// initialize printer
		p.status = 0
		p.imageOffset = 0
	case CommandStart:
		// update status
		if p.commandLength == 4 {
			p.status = byte(0x04)
			// decode the imageData
			colourData := make([]color.RGBA, p.imageOffset)
			pal := p.data[2]

			colors := palette.ColourPalettes[palette.Greyscale]

			for i := 0; i < p.imageOffset; i++ {
				colourData[i] = colors[(pal>>(p.imageData[i]<<1))&0b11]
			}

			// create the image
			img := image.NewRGBA(image.Rect(0, 0, 160, p.imageOffset/160))

			for i := 0; i < len(colourData); i++ {
				img.Pix[i*4] = colourData[i].R
				img.Pix[i*4+1] = colourData[i].G
				img.Pix[i*4+2] = colourData[i].B
				img.Pix[i*4+3] = colourData[i].A
			}
			// has job
			p.hasJob = true
			p.printJob = img
			p.stashImage(colourData)

			p.imageOffset = 0
		}

	case CommandData:
		if p.commandLength == 0x280 {
			// ready to print
			p.status = 0x8

			// p.imageOffset = 0
			currentByte := 0
			for row := 0; row < 2; row++ {
				for tileX := 0; tileX < 20; tileX++ {
					for tileY := 0; tileY < 8; tileY++ {
						for x := 0; x < 8; x++ {
							bit1 := (p.data[currentByte] >> (7 - x)) & 1
							bit2 := (p.data[currentByte+1] >> (7 - x)) & 1

							p.imageData[p.imageOffset+tileY*160+tileX*8+x] = (bit2 << 1) | bit1
						}
						currentByte += 2
					}
				}

				p.imageOffset += 160 * 8
			}

		} else if p.commandLength != 0 {
			// still receiving data
		} else {
			// p.status = 0x08
		}
	case CommandStatus:
		p.status |= 0
	default:
		panic(fmt.Sprintf("unknown command: %x", cmd))
	}
	p.commandLength = 0
	p.byteToSend = p.status
}

func (p *Printer) HasPrintJob() bool {
	return p.hasJob
}

func (p *Printer) GetPrintJob() image.Image {
	p.hasJob = false
	return p.printJob
}

func (p *Printer) stashImage(colourData []color.RGBA) {
	p.stashedImages = append(p.stashedImages, colourData...)
}

func (p *Printer) PrintStashed() {
	saveNextImage(p.stashedImages, 160, len(p.stashedImages)/160)
}

// saveNextImage saves the next imageData
func saveNextImage(c []color.RGBA, i int, u int) {
	f, err := os.Create(fmt.Sprintf("imageData-%d.png", rand.Int()))
	if err != nil {
		panic(err)
	}

	// create the imageData from the data
	img := image.NewRGBA(image.Rect(0, 0, i, u))
	for co := 0; co < len(c); co++ {
		img.Pix[co*4] = c[co].R
		img.Pix[co*4+1] = c[co].G
		img.Pix[co*4+2] = c[co].B
		img.Pix[co*4+3] = c[co].A
	}

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("saved imageData-%d.png\n", u)
}
