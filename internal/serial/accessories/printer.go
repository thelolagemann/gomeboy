package accessories

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types"
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
	imageData         [160 * 200]byte
	imageOffset       int
	timeRemaining     int
	packetSize        uint
	hasJob            bool
	printJob          image.Image
}

func NewPrinter() *Printer {
	return &Printer{}
}

// Send a bit from the printer
func (p *Printer) Send() bool {
	bit := p.byteToSend&types.Bit7 != 0
	p.byteToSend <<= 1

	return bit
}

// Receive a bit to the printer
func (p *Printer) Receive(bit bool) {
	p.byteBeingReceived <<= 1
	if bit {
		p.byteBeingReceived |= types.Bit0
	}

	if p.counter++; p.counter == 8 {
		p.onReceive(p.byteBeingReceived)
		p.byteBeingReceived = 0
		p.counter = 0
	}
}

// onReceive is called when the printer receives a byte
func (p *Printer) onReceive(b byte) {
	if p.position != CommandPositionData {
		fmt.Printf("received %x at %s (%d)\n", b, p.position, p.commandLength)
	}

	// decode the position of the command
	switch p.position {
	case CommandPositionMagic1:
		if b != 0x88 {
			return
		}
		p.status = 0
		p.commandLength = 0
		p.checksum = 0

	case CommandPositionMagic2:
		if b != 0x33 {
			if b != 0x88 {
				// reset
				p.position = CommandPositionMagic1
			}
			return
		}
		p.byteToSend = 0
	case CommandPositionID:
		p.id = b
		p.packetSize++

	case CommandPositionCompression:
		p.compression = b&types.Bit0 == types.Bit0
	case CommandPositionLengthLow:
		p.lengthLeft = uint16(b)
	case CommandPositionLengthHigh:
		p.lengthLeft |= uint16(b&3) << 8
		if p.lengthLeft == 0 {
			p.position++
		}
	case CommandPositionData:
		p.data[p.commandLength] = b
		p.commandLength++
		if p.lengthLeft > 0 {
			p.lengthLeft--
		}
	case CommandPositionChecksumLow:
		p.checksum ^= uint16(b)
	case CommandPositionChecksumHigh:
		p.checksum ^= uint16(b) << 8
		// TODO verify checksum
		if p.checksum != 0 {
			// checksum error
			p.status |= 1
			p.position = CommandPositionMagic1

			fmt.Printf("checksum error: %x", p.checksum)
			return
		}

		p.byteToSend = 0x81
	case CommandPositionKeepAlive:
		if p.id == CommandInit {
			p.byteToSend = 0
		} else {
			if p.status == 6 && p.timeRemaining == 0 {
				p.status = 4 // ready
			}
			p.byteToSend = p.status
		}
		p.keepAlive = b&types.Bit0 == types.Bit0
	case CommandPositionStatus:
		if b == 0 {
			// GB Printer expects 2 0x0s.
			p.packetSize++
			// Send back 0x81 to GB on 1st 0x0
			if p.packetSize == 1 {
				p.byteToSend = 0x81
			} else if p.packetSize == 2 {
				p.runCommand(p.id)
				p.byteToSend = p.status
				p.packetSize = 0
				p.position = CommandPositionMagic1
			}
		}
		return
	default:
		panic(fmt.Sprintf("unknown position: %x", p.position))
	}

	if p.position >= CommandPositionID && p.position < CommandPositionChecksumLow {
		p.checksum += uint16(b)
	}

	if p.position != CommandPositionData {
		p.position++
	}
	if p.position == CommandPositionData && p.lengthLeft == 0 {
		p.position++
	}

	fmt.Printf("sending %x at %s (%d)\n", p.byteToSend, p.position, p.commandLength)
}

// runCommand runs the current command
func (p *Printer) runCommand(cmd Command) {
	//fmt.Printf("running command cmd=%d, length=%d, compression=%t status=%d, keepAlive=%t, timeRemaining=%d\n", cmd, p.commandLength, p.compression, p.status, p.keepAlive, p.timeRemaining)
	switch cmd {
	case CommandInit:
		// initialize printer
		p.status = 0
		p.imageOffset = 0
	case CommandStart:
		if p.commandLength == 4 {
			// update status
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
		}

	case CommandData:
		if p.commandLength == 0x280 {
			// ready to print
			p.status = 0x8

			// p.imageOffset = 0
			currentByte := 0
			for row := 0; row < 2; row++ {
				for col := 0; col < 20; col++ {
					for y := 0; y < 8; y++ {
						for x := 0; x < 8; x++ {
							bit1 := (p.data[(col*8+y)*2] >> 7) & 0x01
							bit2 := (p.data[(col*8+y)*2+1] >> 6) & 0x02

							p.imageData[int(p.imageOffset)+(col*8)+(y*160)+x] = bit1 | bit2

							p.data[(col*8+y)*2] <<= 1
							p.data[(col*8+y)*2+1] <<= 1
						}

						currentByte++
					}
				}

				p.imageOffset += 160 * 8
			}

		} else if p.commandLength != 0 {
			// still receiving data
			// p.status = 0x06
			fmt.Println("still receiving data")
		}
	case CommandStatus:
		p.status |= 0
	default:
		panic(fmt.Sprintf("unknown command: %x", cmd))
	}
}

func (p *Printer) HasPrintJob() bool {
	return p.hasJob
}

func (p *Printer) GetPrintJob() image.Image {
	p.hasJob = false
	return p.printJob
}

// saveNextImage saves the next imageData
func saveNextImage(c []color.RGBA, i int, u int) {
	f, err := os.Create(fmt.Sprintf("imageData-%d.png", rand.Int()))
	if err != nil {
		panic(err)
	}

	// create the imageData from the data
	img := image.NewRGBA(image.Rect(0, 0, i, int(u)))
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
