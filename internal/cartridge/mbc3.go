package cartridge

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
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

// MemoryBankedCartridge3 represents a MemoryBankedCartridge3 cartridge.
type MemoryBankedCartridge3 struct {
	*memoryBankedCartridge

	hasRTC     bool
	rtc        *RTC
	rtcEnabled bool
	latchedRTC []byte
	latched    bool
}

func (m *MemoryBankedCartridge3) Load(s *types.State) {
	m.rtcEnabled = s.ReadBool()
	s.ReadData(m.latchedRTC)
	m.latched = s.ReadBool()
}

func (m *MemoryBankedCartridge3) Save(s *types.State) {
	s.WriteBool(m.rtcEnabled)
	s.WriteData(m.latchedRTC)
	s.WriteBool(m.latched)
}

// NewMemoryBankedCartridge3 returns a new MemoryBankedCartridge3 cartridge.
func NewMemoryBankedCartridge3(rom []byte, header *Header) *MemoryBankedCartridge3 {
	header.b.Lock(io.RAM)
	return &MemoryBankedCartridge3{
		memoryBankedCartridge: newMemoryBankedCartridge(rom, header),
		hasRTC:                header.CartridgeType == MBC3TIMERBATT || header.CartridgeType == MBC3TIMERRAMBATT, // MBC3 + RTC or MBC3 + RAM + RTC
		rtc: &RTC{
			LastUpdate: time.Now(),
		},
		latchedRTC: make([]byte, 0x10),
	}
}

// TODO reimplement RTC

// Write attempts to switch the ROM or RAM bank.
func (m *MemoryBankedCartridge3) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		switch m.CartridgeType {
		case MBC3RAM, MBC3RAMBATT:
			m.ramEnabled = (value & 0xF) == 0xA
		case MBC3TIMERBATT:
			m.rtcEnabled = (value & 0xF) == 0xA
		case MBC3TIMERRAMBATT:
			m.ramEnabled = (value & 0xF) == 0xA
			m.rtcEnabled = (value & 0xF) == 0xA
		}
		if m.ramEnabled && m.ramBank != 0xff {
			m.b.Unlock(io.RAM)
		} else {
			m.b.Lock(io.RAM)
		}
	case address < 0x4000:
		m.setROMBank(uint16(value), false)
	case address < 0x6000:
		if value >= 0x08 && value <= 0x0C {
			if m.hasRTC && m.rtcEnabled {
				m.rtc.Register = value
				m.ramBank = 0xff
			}
		} else if value <= 0x03 && m.ramEnabled {
			m.ramBank = value & 0x03
			if int(m.ramBank)*0x2000 >= len(m.ram) {
				m.ramBank = uint8(int(m.ramBank) % (len(m.ram) / 0x2000))
			}

			if m.ramEnabled {
				// copy new ram bank to bus
				m.b.CopyTo(0xA000, 0xC000, m.ram[int(m.ramBank)*0x2000:])
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
		if m.ramBank != 0xff {
			if m.ramEnabled {
				m.b.Set(address, value)
				m.ram[int(m.ramBank)*0x2000+int(address&0x1fff)] = value
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
