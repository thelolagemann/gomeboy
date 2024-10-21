package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
	"strings"
)

type IO struct {
	widget.BaseWidget

	*io.Bus

	refreshers []func()
}

func NewIO(b *io.Bus) *IO {
	i := &IO{Bus: b}
	i.ExtendBaseWidget(i)
	return i
}

func (i *IO) CreateRenderer() fyne.WidgetRenderer {
	joypadRegs, joypadR := createIOSectionList("Joypad", []uint16{types.P1}, i.Bus)
	serialRegs, serialR := createIOSectionList("Serial", []uint16{types.SB, types.SC}, i.Bus)
	timerRegs, timerR := createIOSectionList("Timer", []uint16{types.DIV, types.TIMA, types.TMA, types.TAC}, i.Bus)
	ppuRegs, ppuR := createIOSectionList("PPU", []uint16{types.LCDC, types.STAT, types.SCY, types.SCX, types.LY, types.LYC, types.DMA, types.BGP, types.OBP0, types.OBP1, types.WY, types.WX}, i.Bus)
	gbcRegs, gbcR := createIOSectionList("GBC", []uint16{types.KEY0, types.KEY1, types.VBK, types.RP, types.SVBK}, i.Bus)
	gbcPCMRegs, gbcPCMR := createIOSectionList("GBC PCM", []uint16{types.PCM12, types.PCM34}, i.Bus)
	gbcPPURegs, gbcPR := createIOSectionList("GBC PPU", []uint16{types.HDMA1, types.HDMA2, types.HDMA3, types.HDMA4, types.HDMA5, types.BCPS, types.BCPD, types.OCPS, types.OCPD, types.OPRI}, i.Bus)
	intRegs, intR := createIOSectionList("Interrupts", []uint16{types.IF, types.IE}, i.Bus)

	ioRegs := container.NewGridWithColumns(2, container.NewVBox(joypadRegs, serialRegs, timerRegs, intRegs, gbcRegs, gbcPCMRegs), container.NewVBox(ppuRegs, gbcPPURegs))

	apu1, apu1R := createIOSectionList("APU Channel 1 (Square+Sweep)", []uint16{types.NR10, types.NR11, types.NR12, types.NR13, types.NR14}, i.Bus)
	apu2, apu2R := createIOSectionList("APU Channel 2 (Square)", []uint16{types.NR21, types.NR22, types.NR23, types.NR24}, i.Bus)
	apu3, apu3R := createIOSectionList("APU Channel 3 (Wave)", []uint16{types.NR30, types.NR31, types.NR32, types.NR33, types.NR34}, i.Bus)
	apu4, apu4R := createIOSectionList("APU Channel 4 (Noise)", []uint16{types.NR41, types.NR42, types.NR43, types.NR44}, i.Bus)
	apuC, apuCR := createIOSectionList("APU Control", []uint16{types.NR50, types.NR51, types.NR52}, i.Bus)
	apuWave, apuWaveCR := createIOSectionList("APU Wave RAM (0xFF30-0xFF37)", []uint16{0xff30, 0xff31, 0xff32, 0xff33, 0xff34, 0xff35, 0xff36, 0xff37}, i.Bus)
	apuWave2, apuWave2CR := createIOSectionList("APU Wave RAM (0xFF38-0xFF3F)", []uint16{0xff38, 0xff39, 0xff3a, 0xff3b, 0xff3c, 0xff3d, 0xff3e, 0xff3f}, i.Bus)
	apuRegs := container.NewGridWithColumns(2, container.NewVBox(apu1, apu2, apuC), container.NewVBox(apu3, apu4, apuWave, apuWave2))

	i.refreshers = append(i.refreshers, joypadR, serialR, timerR, ppuR, gbcR, gbcPCMR, gbcPR, intR, apu1R, apu2R, apu3R, apu4R, apuCR, apuWaveCR, apuWave2CR)
	return widget.NewSimpleRenderer(container.NewVBox(ioRegs, apuRegs))
}

func (i *IO) Refresh() {
	for _, refresher := range i.refreshers {
		refresher()
	}
}

func createIOSectionList(name string, regs []uint16, b *io.Bus) (fyne.CanvasObject, func()) {
	background := canvas.NewRectangle(disabled)
	background.CornerRadius = 5
	sectionLabel := canvas.NewText(name, orange)
	sectionLabel.TextStyle.Monospace = true

	labelBackground := canvas.NewRectangle(bg)
	labelBackground.Resize(sectionLabel.MinSize())
	labelBackground.CornerRadius = 3

	content := container.NewVBox(container.NewStack(labelBackground, container.NewPadded(sectionLabel)))

	if strings.Contains(name, "Wave RAM") {
		row := container.NewHBox()
		for range regs {
			hexValue := canvas.NewText("00", white)
			hexValue.TextStyle.Monospace = true

			row.Add(hexValue)
		}
		content.Add(container.NewPadded(row))
	} else {
		for _, reg := range regs {
			addressText := canvas.NewText(fmt.Sprintf("0x%04X", reg), boolColor)
			addressText.TextStyle.Monospace = true

			nameText := canvas.NewText(fmt.Sprintf("%-5s", hardwareAddressMap[reg]), boolColorNumber)
			nameText.TextStyle.Monospace = true

			hexValue := canvas.NewText("0x00", white)
			hexValue.TextStyle.Monospace = true

			binaryValue := canvas.NewText("(0000 0000)", white)
			binaryValue.TextStyle.Monospace = true

			content.Add(container.NewPadded(container.NewHBox(addressText, nameText, hexValue, binaryValue)))
		}

	}

	/*intFlags := []string{"VBL", "STAT", "TIMER", "SERIAL", "JOYPAD"}
	if name == "Interrupts" {
		for _, f := range intFlags {
			nameLabel := canvas.NewText(fmt.Sprintf("%-8s", f), boolColor)
			status := canvas.NewText("OFF", color.RGBA{0x7f, 0x7f, 0x7f, 255})

			ifLabel := canvas.NewText("IF:", boolColorNumber)
			ifValue := canvas.NewText("0", color.RGBA{0x7f, 0x7f, 0x7f, 255})

			ieLabel := canvas.NewText("IE:", boolColorNumber)
			ieValue := canvas.NewText("0", color.RGBA{0x7f, 0x7f, 0x7f, 255})

			nameLabel.TextStyle.Monospace = true
			status.TextStyle.Monospace = true
			ifLabel.TextStyle.Monospace = true
			ifValue.TextStyle.Monospace = true
			ieLabel.TextStyle.Monospace = true
			ieValue.TextStyle.Monospace = true
			ieLabel.Alignment = fyne.TextAlignTrailing
			ieValue.Alignment = fyne.TextAlignTrailing

			content.Add(container.NewPadded(container.NewHBox(nameLabel, status, ifLabel, ifValue, ieLabel, ieValue)))
		}
	}*/

	background.Resize(content.MinSize())
	return container.NewStack(background, content), func() {
		if strings.Contains(name, "Wave RAM") {
			for i, reg := range regs {
				hexValue := content.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container).Objects[i].(*canvas.Text)
				hexValue.Text = fmt.Sprintf("%02X", b.LazyRead(reg))
				hexValue.Refresh()
			}
		} else {
			for i, reg := range regs {
				regBox := content.Objects[i+1].(*fyne.Container).Objects[0].(*fyne.Container)
				v := b.LazyRead(reg)
				if regBox.Objects[2].(*canvas.Text).Text != fmt.Sprintf("0x%02X", v) {
					regBox.Objects[2].(*canvas.Text).Text = fmt.Sprintf("0x%02X", v)
				}
				binaryText := fmt.Sprintf("%08b", v)
				if mask, ok := hardwareMask[reg]; ok {
					binaryText = formatBinaryWithMask(v, mask)
					regBox.Objects[2].(*canvas.Text).Text = fmt.Sprintf("0x%02X", v&mask)
				}
				regBox.Objects[2].Refresh()

				if regBox.Objects[3].(*canvas.Text).Text != "("+binaryText[:4]+" "+binaryText[4:]+")" {
					regBox.Objects[3].(*canvas.Text).Text = "(" + binaryText[:4] + " " + binaryText[4:] + ")"
					regBox.Objects[3].Refresh()
				}
			}
			/*if name == "Interrupts" {
				intF := b.LazyRead(types.IF)
				intE := b.LazyRead(types.IE)
				for r := range intFlags {
					f := (intF >> r) & 1
					e := (intE >> r) & 1

					flagBox := content.Objects[len(regs)+1+r].(*fyne.Container).Objects[0].(*fyne.Container)
					flagBox.Objects[3].(*canvas.Text).Text = fmt.Sprintf("%1b", f)
					if f == 0 {
						flagBox.Objects[3].(*canvas.Text).Color = color.RGBA{0x7f, 0x7f, 0x7f, 255}
					} else {
						flagBox.Objects[3].(*canvas.Text).Color = white
					}
					flagBox.Objects[3].Refresh()

					flagBox.Objects[5].(*canvas.Text).Text = fmt.Sprintf("%1b", e)
					if e == 0 {
						flagBox.Objects[5].(*canvas.Text).Color = color.RGBA{0x7f, 0x7f, 0x7f, 255}
					} else {
						flagBox.Objects[5].(*canvas.Text).Color = white
					}
					flagBox.Objects[5].Refresh()

					if f == 1 && e == 1 {
						flagBox.Objects[1].(*canvas.Text).Text = " ON"
						flagBox.Objects[1].(*canvas.Text).Color = success
					} else {
						flagBox.Objects[1].(*canvas.Text).Text = "OFF"
						flagBox.Objects[1].(*canvas.Text).Color = color.RGBA{0x7f, 0x7f, 0x7f, 255}
					}
					flagBox.Objects[5].Refresh()
				}
			}*/
		}
	}
}

func formatBinaryWithMask(value uint8, mask uint8) string {
	binaryStr := ""
	for i := 7; i >= 0; i-- {
		// Check if this bit is part of the mask
		if (mask & (1 << i)) != 0 {
			// Show the actual bit value
			bit := (value >> i) & 1
			binaryStr += fmt.Sprintf("%d", bit)
		} else {
			// Show an underscore for unused bits
			binaryStr += "_"
		}

	}
	return binaryStr
}

var hardwareMask = map[uint16]uint8{
	0xFF00: 0b0011_1111,
	0xFF02: 0b1000_0001,
	0xFF07: 0b0000_0111,
	0xFF0F: 0b0001_1111,
	0xFF41: 0b0111_1111,
	0xFF4D: 0b1000_0001,
	0xFF4F: 0b0000_0001,
	0xFF70: 0b0000_0111,
}

var hardwareAddressMap = map[uint16]string{
	0xFF00: "P1",    // Joypad select/input
	0xFF01: "SB",    // Serial transfer data
	0xFF02: "SC",    // Controls the serial port
	0xFF04: "DIV",   // Divider register (upper 8 bits of SYS_CLK)
	0xFF05: "TIMA",  // Timer register
	0xFF06: "TMA",   // Loaded into TIMA when TIMA overflows
	0xFF07: "TAC",   // Controls the timer
	0xFF0F: "IF",    // IF is used to request interrupts
	0xFF10: "NR10",  // Channel 1 Sweep
	0xFF11: "NR11",  // Channel 1 length timer & duty cycle
	0xFF12: "NR12",  // Channel 1 volume & envelope
	0xFF13: "NR13",  // Channel 1 period low
	0xFF14: "NR14",  // Channel 1 period high & control
	0xFF16: "NR21",  // Channel 2 length timer & duty cycle
	0xFF17: "NR22",  // Channel 2 volume & envelope
	0xFF18: "NR23",  // Channel 2 period low
	0xFF19: "NR24",  // Channel 2 period high & control
	0xFF1A: "NR30",  // Channel 3 DAC enable
	0xFF1B: "NR31",  // Channel 3 length timer
	0xFF1C: "NR32",  // Channel 3 output level
	0xFF1D: "NR33",  // Channel 3 period low
	0xFF1E: "NR34",  // Channel 3 period high & control
	0xFF20: "NR41",  // Channel 4 length timer
	0xFF21: "NR42",  // Channel 4 volume & envelope
	0xFF22: "NR43",  // Channel 4 frequency & randomness
	0xFF23: "NR44",  // Channel 4 control
	0xFF24: "NR50",  // Master volume & VIN panning
	0xFF25: "NR51",  // Sound panning
	0xFF26: "NR52",  // Sound On/Off
	0xFF40: "LCDC",  // LCDC is used to control the LCD
	0xFF41: "STAT",  // Reports the LCD status
	0xFF42: "SCY",   // Vertical scroll position of background
	0xFF43: "SCX",   // Horizontal scroll position of the background
	0xFF44: "LY",    // LY is the current scanline
	0xFF45: "LYC",   // Compared to LY and sets the LYC coincidence flag
	0xFF46: "DMA",   // DMA Source address & control
	0xFF47: "BGP",   // Background palette data
	0xFF48: "OBP0",  // Object palette 0 data
	0xFF49: "OBP1",  // Object palette 1 data
	0xFF4A: "WY",    // Window Y position
	0xFF4B: "WX",    // Window X position + 7
	0xFF4C: "KEY0",  // Indicates CGB compatibility mode
	0xFF4D: "KEY1",  // Used for speed switching in CGB mode
	0xFF4F: "VBK",   // VRAM Bank (1-bit)
	0xFF50: "BDIS",  // Boot ROM disable
	0xFF51: "HDMA1", // VRAM DMA source (high)
	0xFF52: "HDMA2", // VRAM DMA source (low)
	0xFF53: "HDMA3", // VRAM DMA destination (high)
	0xFF54: "HDMA4", // VRAM DMA destination (low)
	0xFF55: "HDMA5", // VRAM DMA length/mode/start
	0xFF68: "BCPS",  // Background colour palette specification
	0xFF69: "BCPD",  // Background colour palette data
	0xFF6A: "OCPS",  // Object colour palette specification
	0xFF6B: "OCPD",  // Object colour palette data
	0xFF6C: "OPRI",  // Sets sprite priority in CGB mode
	0xFF70: "SVBK",  // WRAM Bank 01h-07h
	0xFF56: "RP",    // Controls the IR port
	0xFF72: "FF72",  // Undocumented
	0xFF73: "FF73",  // Undocumented
	0xFF74: "FF74",  // Undocumented
	0xFF75: "FF75",  // Undocumented
	0xFF76: "PCM12", // Channel 1/2 PCM data
	0xFF77: "PCM34", // Channel 3/4 PCM Data
	0xFFFF: "IE",    // Controls which interrupts are enabled
}
