package ppu

type Sprite interface {
	UpdateSprite(address uint16, value uint8)
	TileID(no int) int
	Attributes() *SpriteAttributes
	PushScanlines(fromScanline, amount int)
	PopScanline() (int, int)
	IsScanlineEmpty() bool
	ResetScanlines()
}

// DefaultSprite is the 8x8 default sprite.
type DefaultSprite struct {
	*SpriteAttributes
	tileID            int
	ScanlineDrawQueue *Queue
	CurrentTileLine   int
}

func NewDefaultSprite() *DefaultSprite {
	var sprite DefaultSprite
	sprite.SpriteAttributes = &SpriteAttributes{}
	sprite.ScanlineDrawQueue = NewQueue(8)
	return &sprite
}

func (s *DefaultSprite) UpdateSprite(address uint16, value uint8) {
	var attrId = int(address) % 4
	if attrId == 2 {
		s.tileID = int(value)
	} else {
		s.SpriteAttributes.Update(attrId, value)
	}
}

func (s *DefaultSprite) TileID(no int) int {
	if no > 0 {
		panic("8x8 sprite has only one tile")
	}
	return s.tileID
}

func (s *DefaultSprite) Attributes() *SpriteAttributes {
	return s.SpriteAttributes
}

func (s *DefaultSprite) PushScanlines(fromScanline, amount int) {
	for i := 0; i < amount; i++ {
		if s.ScanlineDrawQueue.len == 8 {
			break
		}
		s.ScanlineDrawQueue.Push(fromScanline + i)
	}
}

func (s *DefaultSprite) PopScanline() (int, int) {
	if s.ScanlineDrawQueue.len <= 0 {
		panic("no scanline to draw")
	}
	value := s.ScanlineDrawQueue.Pop()
	oldTileLine := s.CurrentTileLine

	if s.CurrentTileLine < 7 {
		s.CurrentTileLine++
	}

	return value, oldTileLine
}

func (s *DefaultSprite) IsScanlineEmpty() bool {
	return s.ScanlineDrawQueue.len <= 0
}

func (s *DefaultSprite) ResetScanlines() {
	s.ScanlineDrawQueue.Reset()
	s.CurrentTileLine = 0
}

// LargeSprite is the 8x16 large sprite.
type LargeSprite struct {
	*SpriteAttributes
	TileIDs           [2]int
	ScanlineDrawQueue *Queue
	CurrentTileLine   int
}

func NewLargeSprite() *LargeSprite {
	var sprite LargeSprite
	sprite.SpriteAttributes = &SpriteAttributes{}
	sprite.ScanlineDrawQueue = NewQueue(16)
	return &sprite
}

func (s *LargeSprite) UpdateSprite(address uint16, value uint8) {
	var attrId = int(address) % 4
	if attrId == 2 {
		s.TileIDs[0] = int(value)
		s.TileIDs[1] = int(value + 1)
	} else {
		s.SpriteAttributes.Update(attrId, value)
	}
}

func (s *LargeSprite) TileID(no int) int {
	if no > 1 {
		panic("8x16 sprite has only two tiles")
	}
	return s.TileIDs[no]
}

func (s *LargeSprite) Attributes() *SpriteAttributes {
	return s.SpriteAttributes
}

func (s *LargeSprite) PushScanlines(fromScanline, amount int) {
	for i := 0; i < amount; i++ {
		if s.ScanlineDrawQueue.len >= 16 {
			break
		}
		s.ScanlineDrawQueue.Push(fromScanline + i)
	}
}

func (s *LargeSprite) PopScanline() (int, int) {
	if s.ScanlineDrawQueue.len <= 0 {
		panic("no scanline to draw")
	}
	value := s.ScanlineDrawQueue.Pop()
	oldTileLine := s.CurrentTileLine

	if s.CurrentTileLine < 15 {
		s.CurrentTileLine++
	}

	return value, oldTileLine
}

func (s *LargeSprite) IsScanlineEmpty() bool {
	return s.ScanlineDrawQueue.len <= 0
}

func (s *LargeSprite) ResetScanlines() {
	s.ScanlineDrawQueue.Reset()
	s.CurrentTileLine = 0
}

// SpriteAttributes represents the attributes of a sprite.
type SpriteAttributes struct {
	X int
	Y int
	// Bit 7 - OBJ-to-BG Priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3)
	// (Used for both BG and Window. BG color 0 is always behind OBJ)
	Priority bool
	// Bit 6 - Y flip          (0=Normal, 1=Vertically mirrored)
	FlipY bool
	// Bit 5 - X flip          (0=Normal, 1=Horizontally mirrored)
	FlipX bool
	// Bit 4 - Palette number  **Non CGB Mode Only** (0=OBP0, 1=OBP1)
	UseSecondPalette uint8
	// Bit 3 - Tile VRAM-Bank  **CGB Mode Only**     (0=Bank 0, 1=Bank 1)
	VRAMBank uint8
	// Bit 0-2 - Palette number  **CGB Mode Only**     (OBP0-7)
	CGBPalette uint8
}

func (s *SpriteAttributes) Update(attribute int, value uint8) {
	switch attribute {
	case 0:
		s.Y = int(value)
	case 1:
		s.X = int(value)
	case 3:
		s.Priority = value&0x80 != 0
		s.FlipY = value&0x40 != 0
		s.FlipX = value&0x20 != 0
		if value&0x10 != 0 {
			s.UseSecondPalette = 1
		} else {
			s.UseSecondPalette = 0
		}
		s.VRAMBank = value & 0x08
		s.CGBPalette = value & 0x07
	}
}

type Queue struct {
	content      []int
	maxQueueSize int
	readHead     int
	writeHead    int
	len          int
}

func NewQueue(maxQueueSize int) *Queue {
	var queue Queue
	queue.content = make([]int, maxQueueSize)
	queue.maxQueueSize = maxQueueSize
	return &queue
}

func (q *Queue) Push(value int) {
	if q.len >= q.maxQueueSize {
		return
	}

	q.content[q.writeHead] = value
	q.writeHead = (q.writeHead + 1) % q.maxQueueSize
	q.len++
}

func (q *Queue) Pop() int {
	if q.len <= 0 {
		return -1
	}
	result := q.content[q.readHead]
	q.content[q.readHead] = -1
	q.readHead = (q.readHead + 1) % q.maxQueueSize
	q.len--
	return result
}

func (q *Queue) Reset() {
	q.readHead = 0
	q.writeHead = 0
	q.len = 0
}
