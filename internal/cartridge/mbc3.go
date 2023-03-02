package cartridge

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"time"
)

type RTC struct {
	Seconds                     uint8
	Minutes                     uint8
	Hours                       uint8
	DaysLower                   uint8
	DaysHigherAndControl        uint8
	LatchedSeconds              uint8
	LatchedMinutes              uint8
	LatchedHours                uint8
	LatchedDaysLower            uint8
	LatchedDaysHigherAndControl uint8

	Register       uint8
	LatchFlagValue uint8
	LastUpdate     time.Time
	CyclesCount    uint16
}

// Update updates the RTC state.
func (r *RTC) Update() {
	delta := time.Since(r.LastUpdate)

	if r.DaysHigherAndControl>>6&0x1 == 0 && (delta >= time.Second) {
		r.LastUpdate = time.Now()
		var days uint32
		deltaSeconds := int(delta.Seconds())
		r.Seconds += uint8(deltaSeconds % 60)

		if r.Seconds >= 60 {
			r.Seconds -= 60
			r.Minutes++
		}
		deltaSeconds /= 60
		r.Minutes += uint8(deltaSeconds % 60)
		if r.Minutes >= 60 {
			r.Minutes -= 60
			r.Hours++
		}
		deltaSeconds /= 60
		r.Hours += uint8(deltaSeconds % 24)
		if r.Hours >= 24 {
			r.Hours -= 24
			days++
		}
		deltaSeconds /= 24
		days += uint32(deltaSeconds)
		days += uint32(r.DaysLower)
		days += uint32(r.DaysHigherAndControl&0x1) << 8
		if days >= 512 {
			days = days % 512
			r.DaysHigherAndControl ^= 1 << 7
		}

		r.DaysLower = uint8(days & 0xFF)
		r.DaysHigherAndControl = r.DaysHigherAndControl & 0xFE
		if days >= 256 {
			r.DaysHigherAndControl |= 1
		}
	}
}

// MemoryBankedCartridge3 represents a MemoryBankedCartridge3 cartridge. This cartridge type has external RAM and
// supports switching between 2 ROM banks and 4 RAM banks, and provides a real time clock.
type MemoryBankedCartridge3 struct {
	rom     []byte
	romBank uint32

	ram        []byte
	ramBank    int32
	ramEnabled bool

	hasRTC     bool
	rtc        *RTC
	rtcEnabled bool
	latchedRTC []byte
	latched    bool
	header     *Header
}

func (m *MemoryBankedCartridge3) Load(s *types.State) {
	m.romBank = s.Read32()
	s.ReadData(m.ram)
	m.ramBank = int32(s.Read32())
	m.ramEnabled = s.ReadBool()
	m.rtcEnabled = s.ReadBool()
	s.ReadData(m.latchedRTC)
	m.latched = s.ReadBool()
}

func (m *MemoryBankedCartridge3) Save(s *types.State) {
	s.Write32(m.romBank)
	s.WriteData(m.ram)
	s.Write32(uint32(m.ramBank))
	s.WriteBool(m.ramEnabled)
	s.WriteBool(m.rtcEnabled)
	s.WriteData(m.latchedRTC)
	s.WriteBool(m.latched)
}

// NewMemoryBankedCartridge3 returns a new MemoryBankedCartridge3 cartridge.
func NewMemoryBankedCartridge3(rom []byte, header *Header) *MemoryBankedCartridge3 {
	return &MemoryBankedCartridge3{
		rom:     rom,
		romBank: 1,
		ram:     make([]byte, header.RAMSize),
		hasRTC:  header.CartridgeType == MBC3TIMERBATT || header.CartridgeType == MBC3TIMERRAMBATT, // MBC3 + RTC or MBC3 + RAM + RTC
		rtc: &RTC{
			LastUpdate: time.Now(),
		},
		latchedRTC: make([]byte, 0x10),
		header:     header,
	}
}

// Read returns the value from the cartridges ROM or RAM, depending on the bank
// selected.
func (m *MemoryBankedCartridge3) Read(address uint16) uint8 {
	switch {
	case address < 0x4000:
		return m.rom[address]
	case address < 0x8000:
		return m.rom[uint32(address-0x4000)+m.romBank*0x4000]
	case address >= 0xA000 && address < 0xC000:
		if m.ramBank >= 0 {
			if m.ramEnabled {
				return m.ram[uint32(m.ramBank)*0x2000+uint32(address&0x1FFF)]
			} else {
				return 0xFF
			}
		} else if m.hasRTC && m.rtcEnabled {
			switch m.rtc.Register {
			case 0x8:
				return m.rtc.LatchedSeconds
			case 0x9:
				return m.rtc.LatchedMinutes
			case 0xA:
				return m.rtc.LatchedHours
			case 0xB:
				return m.rtc.LatchedDaysLower
			case 0xC:
				return m.rtc.LatchedDaysHigherAndControl
			default:
				return 0xFF
			}
		} else {
			return 0xFF
		}
	}

	panic(fmt.Sprintf("mbc3: illegal read from address: %X", address))
}

// Write attempts to switch the ROM or RAM bank.
func (m *MemoryBankedCartridge3) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		switch m.header.CartridgeType {
		case MBC3RAM, MBC3RAMBATT:
			m.ramEnabled = (value & 0xF) == 0xA
		case MBC3TIMERBATT:
			m.rtcEnabled = (value & 0xF) == 0xA
		case MBC3TIMERRAMBATT:
			m.ramEnabled = (value & 0xF) == 0xA
			m.rtcEnabled = (value & 0xF) == 0xA
		}
	case address < 0x4000:
		m.romBank = uint32(value)
		if int(m.romBank)*0x4000 >= len(m.rom) {
			m.romBank = uint32(int(m.romBank) % (len(m.rom) / 0x4000))
		}
		if m.romBank == 0 {
			m.romBank = 1
		}
	case address < 0x6000:
		if value >= 0x08 && value <= 0x0C {
			if m.hasRTC && m.rtcEnabled {
				m.rtc.Register = value
				m.ramBank = -1
			}
		} else if value <= 0x03 && m.ramEnabled {
			m.ramBank = int32(value & 0x03)
			if len(m.ram) <= 0 {
				m.ramBank = 0
			} else if int(m.ramBank)*0x2000 >= len(m.ram) {
				m.ramBank = int32(int(m.ramBank) % (len(m.ram) / 0x2000))
			}
		}
	case address < 0x8000:
		if m.hasRTC {
			if m.rtc.LatchFlagValue == 0x00 && value == 0x01 {
				m.rtc.Update()
				m.rtc.LatchedSeconds = m.rtc.Seconds
				m.rtc.LatchedMinutes = m.rtc.Minutes
				m.rtc.LatchedHours = m.rtc.Hours
				m.rtc.LatchedDaysLower = m.rtc.DaysLower
				m.rtc.LatchedDaysHigherAndControl = m.rtc.DaysHigherAndControl
			}
			m.rtc.LatchFlagValue = value
		}
	case address >= 0xA000 && address < 0xC000:
		if m.ramBank >= 0 {
			if m.ramEnabled {
				m.ram[m.ramBank*0x2000+int32(address&0x1FFF)] = value
			} else if m.hasRTC && m.rtcEnabled {
				switch m.rtc.Register {
				case 0x8:
					m.rtc.Seconds = value & 0x3F
				case 0x9:
					m.rtc.Minutes = value & 0x3F
				case 0xA:
					m.rtc.Hours = value & 0x1F
				case 0xB:
					m.rtc.DaysLower = value
				case 0xC:
					m.rtc.DaysHigherAndControl = value & 0xC1
				}
			}
		}

	}
}

// Load loads the cartridge's RAM from the given data.
func (m *MemoryBankedCartridge3) LoadRAM(data []byte) {
	copy(m.ram, data)
}

// Save saves the cartridge's RAM to the given data.
func (m *MemoryBankedCartridge3) SaveRAM() []byte {
	data := make([]byte, len(m.ram))
	copy(data, m.ram)
	return data
}
