package web

import (
	"bytes"
	"encoding/binary"
	"github.com/cespare/xxhash"
	"github.com/google/brotli/go/cbrotli"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"sync"
)

func init() {
	h := &hub{
		clients: make(map[*Client]bool),
		player1: nil,
		player2: nil,

		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),

		compression:      true,
		compressionLevel: 2,
		framePatching:    true,
		frameSkipping:    true,
		framePatchRatio:  1,
	}
	p1 := &Player{
		hub:           h,
		clientClose:   make(chan struct{}, 10),
		clientConnect: make(chan *Client, 10),
		clientSync:    make(chan *Client, 10),
		patchCache:    newCache(16384),
		frameCache:    newCache(1024),
		currentFrame:  make([]byte, 92160),
	}
	p2 := &Player{
		hub:           h,
		clientClose:   make(chan struct{}),
		clientConnect: make(chan *Client, 10),
		clientSync:    make(chan *Client, 10),
		patchCache:    newCache(16384),
		frameCache:    newCache(1024),
		currentFrame:  make([]byte, 92160),
	}

	display.Install("web", p1, []display.DriverOption{})
	h.player1 = p1
	h.player2 = p2

	go func() {
		err := h.run()
		if err != nil {
			panic(err)
		}
	}()
}

type Player struct {
	c             *Client
	hub           *hub
	clientClose   chan struct{}
	clientConnect chan *Client
	clientSync    chan *Client

	gb                     display.Emulator
	pressed, release       chan<- io.Button
	patchCache, frameCache *cache
	currentFrame           []byte

	playerByte byte

	mu sync.Mutex
}

func (p *Player) Initialize(emu display.Emulator) {}

func (p *Player) Attach(gb *gameboy.GameBoy) {
	p.gb = gb
}

func (p *Player) Start(fb <-chan []byte, events <-chan event.Event, pressed, released chan<- io.Button) error {
	// setup keys
	p.pressed = pressed
	p.release = released

	// handle events
	go func() {
		for {
			<-events // TODO handle events
		}
	}()

	// determine which player byte to use
	var playerByte byte = 0
	if p.hub.player1 == p {
		playerByte = 1
	} else if p.hub.player2 == p {
		playerByte = 2
	}
	p.playerByte = playerByte

	// setup vars
	var dirtied = false
	var dirtiedPixelCount, framesSkipped = 0, 0
	dirtiedPixels := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*4)
	emptyDirtiedPixels := make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*4)

	var frameSkipBuf = make([]byte, 4)
	var cacheBuf = make([]byte, 2)
	var e = Frame
	var buffer, output []byte

	for {
		select {
		case f := <-fb:
			// process incoming framebuffer
			for i := 0; i < ppu.ScreenWidth*ppu.ScreenHeight; i++ {
				r, g, b := f[i*3], f[i*3+1], f[i*3+2]
				if p.currentFrame[i*4] != r || p.currentFrame[i*4+1] != g || p.currentFrame[i*4+2] != b {
					dirtied = true

					dirtiedPixels[i*4] = r
					dirtiedPixels[i*4+1] = g
					dirtiedPixels[i*4+2] = b
					dirtiedPixels[i*4+3] = 255

					dirtiedPixelCount++
				}

				p.currentFrame[i*4] = r
				p.currentFrame[i*4+1] = g
				p.currentFrame[i*4+2] = b
				p.currentFrame[i*4+3] = 255
			}

			// did the framebuffer get dirtied (or has the hub disabled frameSkipping)
			if dirtied || !p.hub.frameSkipping {
				// handle frame skips
				if framesSkipped > 0 && p.hub.frameSkipping {
					binary.LittleEndian.PutUint32(frameSkipBuf, uint32(framesSkipped))

					// send frames skipped to clients
					p.hub.broadcast <- p.createMessage(FrameSkip, bytes.TrimRight(frameSkipBuf, "\x00"))
				}

				// can we patch the framebuffer?
				if dirtiedPixelCount < (p.hub.framePatchRatio*4608) && p.hub.framePatching {
					buffer = dirtiedPixels
					e = FramePatch
				} else {
					buffer = p.currentFrame
				}

				// handle compression (if enabled)
				if p.hub.compression {
					var err error
					output, err = cbrotli.Encode(buffer, cbrotli.WriterOptions{
						Quality: p.hub.compressionLevel,
					})
					if err != nil {
						// TODO handle error
						panic(err)
					}
				} else {
					output = buffer
				}

				// calculate the hash of the data to see if it exists in cache
				hash := xxhash.Sum64(output)

				// should we be looking in frame of patch cache
				if e == FramePatch {
					p.patchCache.Lock()

					if idx := p.patchCache.index(hash); idx != -1 { // found in cache
						binary.LittleEndian.PutUint16(cacheBuf, uint16(idx))
						p.hub.broadcast <- p.createMessage(PatchCache, bytes.TrimRight(cacheBuf, "\x00"))
					} else { // not found in cache
						p.patchCache.add(hash, output)
						binary.LittleEndian.PutUint16(cacheBuf, uint16(p.patchCache.index(hash)))
						p.hub.broadcast <- p.createMessage(FramePatch, append(cacheBuf, output...))
					}

					p.patchCache.Unlock()
				} else { // full frame
					p.frameCache.Lock()

					if idx := p.frameCache.index(hash); idx != -1 { // found in cache
						binary.LittleEndian.PutUint16(cacheBuf, uint16(idx))
						p.hub.broadcast <- p.createMessage(FrameCache, bytes.TrimRight(cacheBuf, "\x00"))
					} else { // not found in cache
						p.frameCache.add(hash, output)
						binary.LittleEndian.PutUint16(cacheBuf, uint16(p.frameCache.index(hash)))
						p.hub.broadcast <- p.createMessage(Frame, append(cacheBuf, output...))
					}

					p.frameCache.Unlock()
				}
			} else if p.hub.frameSkipping { // if not dirtied, but frame skipping is enabled, increment count
				framesSkipped++
			}

			// reset various flags
			dirtied = false
			dirtiedPixelCount = 0
			e = Frame
			copy(dirtiedPixels, emptyDirtiedPixels)
		case <-p.clientClose:
			p.c = nil
		case c := <-p.clientConnect:
			// is there is already a client attached, or the client
			// is the one connecting, then ignore
			if p.c != nil || c == p.c {
				continue
			}

			p.c = c
			c.Send <- p.createMessage(PlayerIdentify, []byte{p.playerByte})
			go p.ReadPump(c.player)
		case c := <-p.clientSync:
			p.Sync(c)
		}
	}
}

func (p *Player) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (p *Player) ReadPump(from <-chan []byte) {
	for {
		select {
		case message, ok := <-from:
			// check if the client has been closed
			if !ok {
				return
			}

			// handle special case of pause/play
			if len(message) == 1 {
				if message[0] == 0 {
					p.gb.SendCommand(display.Pause)
					p.hub.sendAllButClient(p.c, p.createMessage(PlayerInfo, []byte{PausePlay, 0}))
				} else {
					p.gb.SendCommand(display.Resume)
					p.hub.sendAllButClient(p.c, p.createMessage(PlayerInfo, []byte{PausePlay, 1}))
				}

				continue // skip further processing
			}

			switch message[0] {
			case 9: // PPU related control
				// assert gb (todo find better solution)
				gb := p.gb.(*gameboy.GameBoy)
				switch message[1] {
				case 0: // background
					gb.PPU.Debug.BackgroundDisabled = message[2] == 0
					p.hub.sendAllButClient(p.c, []byte{PlayerInfo, BackgroundDisabled, message[2]})
				case 1: // window
					gb.PPU.Debug.WindowDisabled = message[2] == 0
					p.hub.sendAllButClient(p.c, []byte{PlayerInfo, WindowDisabled, message[2]})
				case 2: // sprites
					gb.PPU.Debug.SpritesDisabled = message[2] == 0
					p.hub.sendAllButClient(p.c, []byte{PlayerInfo, SpritesDisabled, message[2]})
				}

				continue // skip further processing
			default:
				button := message[0]
				state := message[1]

				if state == 0 {
					p.release <- button
				} else {
					p.pressed <- button
				}
			}
		}
	}
}

// Sync sends the current state of the Game Boy and various
// Player information to the provided client.
func (p *Player) Sync(c *Client) {
	if p.c == nil {
		// sync
		p.clientConnect <- c
	}

	frameData, err := cbrotli.Encode(p.currentFrame, cbrotli.WriterOptions{
		Quality: 9,
	})

	if err != nil {
		// TODO handle err
		return
	}

	c.Send <- p.createMessage(FrameSync, frameData)

	// send caches
	var data []byte
	for i, c := range p.patchCache.cache {
		// this shouldn't happen in practice, but will throw a panic if it does
		if len(c.data) == 0 {
			continue
		}

		// calculate length of cache item and index
		var length, idx = make([]byte, 2), make([]byte, 2)
		binary.LittleEndian.PutUint16(length, uint16(len(c.data)))
		binary.LittleEndian.PutUint16(idx, uint16(i))

		data = append(data, append(append(length, idx...), c.data...)...)
	}

	c.Send <- p.createMessage(PatchCacheSync, data)

	data = []byte{}
	for i, c := range p.frameCache.cache {
		// this shouldn't happen in practice, but will throw a panic if it does
		if len(c.data) == 0 {
			continue
		}

		length := make([]byte, 2)
		idx := make([]byte, 2)
		binary.LittleEndian.PutUint16(length, uint16(len(c.data)))
		binary.LittleEndian.PutUint16(idx, uint16(i))

		data = append(data, append(append(length, idx...), c.data...)...)
	}

	c.Send <- p.createMessage(FrameCacheSync, data)
}

func (p *Player) createMessage(messageType Type, data []byte) []byte {
	return append([]byte{messageType, p.playerByte}, data...)
}
