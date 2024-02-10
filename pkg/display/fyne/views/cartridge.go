package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"strconv"
)

type Cartridge struct {
	C *io.Cartridge
}

func (c *Cartridge) Title() string {
	return "Cartridge"
}

func (c *Cartridge) Run(window fyne.Window, events <-chan event.Event) error {
	// create main container
	main := container.NewVBox()

	// create cartridge info container
	cartridgeInfo := container.NewVBox()

	// add cartridge info container to main container
	main.Add(cartridgeInfo)

	// create textgrid for cartridge info
	cartridgeInfoGrid := widget.NewTextGrid()
	cartridgeInfoGrid.SetText(`Title			` + c.C.Title + `
Manufacturer	` + c.C.ManufacturerCode + `
CGB Support		` + strconv.FormatBool(c.C.IsCGBCartridge()) + `
SGB Support		` + strconv.FormatBool(c.C.SGBFlag) + `
Cartridge Type	` + c.C.CartridgeType.String() + `
ROM Size		` + humanReadable(uint(c.C.ROMSize)) + `
RAM Size		` + humanReadable(uint(c.C.RAMSize)) + `
Destination		` + c.C.Destination() + `
Licensee		` + c.C.Licensee() + `
ROM Version		` + strconv.Itoa(int(c.C.MaskROMVersion)) + `
Header Checksum	` + strconv.Itoa(int(c.C.HeaderChecksum)) + `
Global Checksum	` + strconv.Itoa(int(c.C.GlobalChecksum)))

	// add cartridge info textgrid to cartridge info container
	cartridgeInfo.Add(cartridgeInfoGrid)

	// create cartridge rom container
	cartridgeROM := container.NewVBox()

	// add cartridge rom container to main container
	main.Add(cartridgeROM)

	// create textgrid for cartridge rom

	window.SetContent(main)

	runUntilQuit(events, func() {

	})

	return nil
}
