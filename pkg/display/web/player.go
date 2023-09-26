package web

import (
	"bytes"
	"encoding/binary"
	"github.com/cespare/xxhash"
	"github.com/google/brotli/go/cbrotli"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"sync"
)

type Player struct {
	c             *Client
	hub           *Hub
	clientClose   chan struct{}
	clientConnect chan *Client
	clientSync    chan *Client

	gb                     *gameboy.GameBoy
	pressed, release       chan joypad.Button
	patchCache, frameCache *cache
	currentFrame           []byte

	playerByte byte

	mu sync.Mutex
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
					p.gb.Pause()
					p.hub.sendAllButClient(p.c, p.createMessage(PlayerInfo, []byte{PausePlay, 0}))
				} else {
					p.gb.Unpause()
					p.hub.sendAllButClient(p.c, p.createMessage(PlayerInfo, []byte{PausePlay, 1}))
				}

				continue // skip further processing
			}

			switch message[0] {
			case 9: // PPU related control
				switch message[1] {
				case 0: // background
					p.gb.PPU.Debug.BackgroundDisabled = message[2] == 0
					p.hub.sendAllButClient(p.c, []byte{PlayerInfo, BackgroundDisabled, message[2]})
				case 1: // window
					p.gb.PPU.Debug.WindowDisabled = message[2] == 0
					p.hub.sendAllButClient(p.c, []byte{PlayerInfo, WindowDisabled, message[2]})
				case 2: // sprites
					p.gb.PPU.Debug.SpritesDisabled = message[2] == 0
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

// Start starts the Game Boy of the Player. This
// should be called after the Player has been
// configured by the user, either via configuration
// or manual setup.
func (p *Player) Start() {
	// setup a framebuffer for the gameboy
	fb := make(chan []byte, 144)

	// setup events
	events := make(chan event.Event, 144)
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

	var dirtied = false
	var dirtiedPixelCount, framesSkipped = 0, 0
	dirtiedPixels := make([]byte, ppu.ScreenHeight*ppu.ScreenWidth*4)
	emptyDirtiedPixels := make([]byte, ppu.ScreenHeight*ppu.ScreenWidth*4)

	var frameSkipBuf = make([]byte, 4)
	var cacheBuf = make([]byte, 2)
	var e = Frame
	var buffer, output []byte

	// start gameboy in a goroutine
	go p.gb.Start(fb, events, p.pressed, p.release)

	for {
		select {
		case f := <-fb:
			// process incoming framebuffer
			for i := 0; i < ppu.ScreenHeight*ppu.ScreenWidth; i++ {
				// track dirty pixel count to determine appropriate update (patch vs full frame)
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

			// was the framebuffer dirtied (or has the hub disabled FrameSkipping)
			if dirtied || !p.hub.FrameSkipping {
				// handle frame skipping
				if framesSkipped > 0 && p.hub.FrameSkipping {
					binary.LittleEndian.PutUint32(frameSkipBuf, uint32(framesSkipped))

					// send skip update to all clients
					p.hub.SendAll(p.createMessage(FrameSkip, bytes.TrimRight(frameSkipBuf, "\x00")))
					framesSkipped = 0
				}

				// determine if we should patch the framebuffer
				if dirtiedPixelCount < (p.hub.FramePatchRatio*4608) && p.hub.FramePatching {
					buffer = dirtiedPixels
					e = FramePatch
				} else {
					buffer = p.currentFrame
				}

				// handle compression (if enabled)
				if p.hub.Compression {
					var err error
					output, err = cbrotli.Encode(buffer, cbrotli.WriterOptions{
						Quality: 7,
					})
					if err != nil {
						// TODO: handle error
						continue
					}
				} else {
					output = buffer
				}

				// calculate the hash of the data
				hash := xxhash.Sum64(output)

				// is this patch?
				if e == FramePatch {
					p.patchCache.Lock()
					// does this patch exist in the cache?
					if idx := p.patchCache.index(hash); idx != -1 { // yes
						binary.LittleEndian.PutUint16(cacheBuf, uint16(idx))
						p.hub.SendAll(p.createMessage(PatchCache, bytes.TrimRight(cacheBuf, "\x00")))
					} else { // no
						p.patchCache.add(hash, output)
						binary.LittleEndian.PutUint16(cacheBuf, uint16(p.patchCache.index(hash)))
						p.hub.SendAll(p.createMessage(FramePatch, append(cacheBuf, output...)))
					}
					p.patchCache.Unlock()
				} else { // full frame
					p.frameCache.Lock()
					// does this frame exist in the cache?
					if idx := p.frameCache.index(hash); idx != -1 { // yes
						binary.LittleEndian.PutUint16(cacheBuf, uint16(idx))
						p.hub.SendAll(p.createMessage(FrameCache, bytes.TrimRight(cacheBuf, "\x00")))
					} else { // no
						p.frameCache.add(hash, output)
						binary.LittleEndian.PutUint16(cacheBuf, uint16(p.frameCache.index(hash)))
						p.hub.SendAll(p.createMessage(Frame, append(cacheBuf, output...)))
					}
					p.frameCache.Unlock()
				} // end patch check
			} else if p.hub.FrameSkipping { // if FrameSkipping is enabled however, update the frames skipped
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
			// if there is already a client attached or the
			// client is the one connecting, continue
			if p.c != nil || p.c == c {
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

func (p *Player) isPlayer1() bool {
	return p == p.c.hub.player1
}

func (p *Player) isPlayer2() bool {
	return p == p.c.hub.player2
}

func (p *Player) createMessage(messageType Type, data []byte) []byte {
	return append([]byte{messageType, p.playerByte}, data...)
}
