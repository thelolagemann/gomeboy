package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"strconv"
)

// Cartridge displays information about the currently loaded [io.Cartridge]
type Cartridge struct {
	widget.BaseWidget
	*io.Cartridge
}

func NewCartridge(cart *io.Cartridge) *Cartridge {
	c := &Cartridge{Cartridge: cart}
	c.ExtendBaseWidget(c)
	return c
}

func (c *Cartridge) CreateRenderer() fyne.WidgetRenderer {
	// create cartridge info container
	cartridgeInfo := container.NewVBox()

	// create textgrid for cartridge info
	cartridgeInfoGrid := widget.NewTextGrid()
	cartridgeInfoGrid.SetText(`Title				` + c.Cartridge.Title + `
Manufacturer		` + c.ManufacturerCode + `
CGB Support			` + strconv.FormatBool(c.IsCGBCartridge()) + `
SGB Support			` + strconv.FormatBool(c.SGBFlag) + `
Cartridge Type		` + c.CartridgeType.String() + `
ROM Size			` + utils.HumanReadable(c.ROMSize) + `
RAM Size			` + utils.HumanReadable(c.RAMSize) + `
Destination			` + c.Destination() + `
Licensee			` + c.Licensee() + `
ROM Version			` + strconv.Itoa(int(c.MaskROMVersion)) + `
Header Checksum		` + fmt.Sprintf("0x%02X", c.HeaderChecksum) + `
Global Checksum		` + fmt.Sprintf("0x%04X", c.GlobalChecksum))
	cartridgeInfo.Add(newCard("Cartridge Information", container.NewPadded(cartridgeInfoGrid)))

	// create features checklist
	type feature struct {
		name  string
		value bool
	}
	features := []feature{{"Accelerometer", c.Features.Accelerometer}, {"Battery", c.Features.Battery}, {"RAM", c.Features.RAM}, {"RTC", c.Features.RTC}, {"Rumble", c.Features.Rumble}}
	checkList := container.NewVBox()
	for _, k := range features {
		checkList.Add(newStaticCheckbox(k.name, k.value))
	}
	cartridgeInfo.Add(newCard("Features", checkList))

	return widget.NewSimpleRenderer(cartridgeInfo)
}
