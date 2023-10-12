package ppu

type dirtyCause int

const (
	lcdc dirtyCause = iota
	scy
	scx
	lyc
	bgp
	obp0
	obp1
	wy
	wx
	bcps
	bcpd
	ocps
	ocpd
	tile
	tileMap
	tileAttr
)

type dirtyEvent struct {
	cause   dirtyCause
	atCycle uint64
}

func (p *PPU) dirtyBackground(cause dirtyCause) {
	p.dirtiedLog[p.lastDirty] = dirtyEvent{
		cause: cause,
	}
	p.lastDirty++
	p.backgroundDirty = true
	// fmt.Println("dirty background", cause, p.CurrentCycle)
}
