package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/cartridge"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"strconv"
)

type Cartridge struct {
	C *cartridge.Cartridge
}

func (c *Cartridge) Title() string {
	return "Cartridge"
}

func (c *Cartridge) Run(window fyne.Window, events <-chan display.Event) error {
	// create main container
	main := container.NewVBox()

	// create cartridge info container
	cartridgeInfo := container.NewVBox()

	// add cartridge info container to main container
	main.Add(cartridgeInfo)

	// create textgrid for cartridge info
	cartridgeInfoGrid := widget.NewTextGrid()
	cartridgeInfoGrid.SetText(`Title			` + c.C.Title() + `
Manufacturer	` + c.C.Header().ManufacturerCode + `
CGB Support		` + strconv.FormatBool(c.C.Header().CartridgeGBMode == cartridge.FlagOnlyCGB) + `
SGB Support		` + strconv.FormatBool(c.C.Header().SGBFlag) + `
Cartridge Type	` + c.C.Header().CartridgeType.String() + `
ROM Size		` + humanReadable(c.C.Header().ROMSize) + `
RAM Size		` + humanReadable(c.C.Header().RAMSize) + `
Destination		` + c.C.Header().Destination() + `
Licensee		` + c.C.Header().Licensee() + `
ROM Version		` + strconv.Itoa(int(c.C.Header().MaskROMVersion)) + `
Header Checksum	` + strconv.Itoa(int(c.C.Header().HeaderChecksum)) + `
Global Checksum	` + strconv.Itoa(int(c.C.Header().GlobalChecksum)))

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
