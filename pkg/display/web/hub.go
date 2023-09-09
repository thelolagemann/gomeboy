package web

import (
	"encoding/binary"
	"github.com/gorilla/websocket"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/internal/types"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 16,
	WriteBufferSize: 1024 * 16,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	Clients    map[*Client]bool
	player1    *Player
	player2    *Player
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	activeROM  []byte

	Compression      bool
	CompressionLevel int
	FramePatching    bool
	FramePatchRatio  int // 0-20
	FrameSkipping    bool
	CurrentID        uint8

	mu sync.Mutex
}

func NewHub(opts ...HubOpt) *Hub {
	h := &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		CurrentID:  0,

		Compression:      true,
		CompressionLevel: 2,
		FramePatching:    true,
		FrameSkipping:    true,
		FramePatchRatio:  1,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

type HubOpt func(*Hub)

func WithROM(rom []byte) HubOpt {
	return func(h *Hub) {
		h.activeROM = rom
	}
}

func (h *Hub) Info() byte {
	info := uint8(0)
	if h.player1.gb.IsRunning() {
		info |= types.Bit1
	}
	if h.player2.gb.IsRunning() {
		info |= types.Bit2
	}
	if h.Compression {
		info |= types.Bit3
	}
	if h.FramePatching {
		info |= types.Bit4
	}
	if h.FrameSkipping {
		info |= types.Bit5
	}
	if h.player1.gb.IsPaused() {
		info |= types.Bit6
	}
	if h.player2.gb.IsPaused() {
		info |= types.Bit7
	}
	return info
}

// Run the hub, listening for new connections and
// broadcasting messages to connected clients.
func (h *Hub) Run() {
	// create and start the 2 players
	h.player1 = NewPlayer(h.activeROM, h)
	h.player2 = NewPlayer(h.activeROM, h)
	go h.player1.Start()

	// create http handler to listen for incoming connections
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// upgrade the connection to a websocket connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			// TODO: handle error
			return
		}

		client := h.NewClient(conn, r)

		// spawn goroutines for read/write pump
		go client.ReadPump()
		go client.WritePump()

		// send the client hub information  TODO add per client info bg, window, sprite disable cache/run status)
		client.Send <- []byte{ClientInfo, ClientStatus, h.Info(), uint8(h.CompressionLevel), uint8(h.FramePatchRatio)}

		// synchronize the players
		h.player1.clientSync <- client
		if h.player2.c != nil {
			h.player2.clientSync <- client
		}

		// synchronize connected clients
		var data []byte
		for c := range h.Clients {
			if c == client { // skip self
				continue
			}

			data = append(data, c.Metadata.RemoteAddr...)
			data = append(data, 0)
			data = append(data, c.Metadata.UserAgent...)
			data = append(data, 0)
			data = append(data, c.Metadata.Username...)
			data = append(data, 0)
			data = append(data, c.ID)
			data = append(data, byte('\n'))
		}

		if len(data) > 0 {
			data = data[:len(data)-1] // remove last newline
		}

		client.Send <- append([]byte{ClientListSync}, data...)

	})

	// setup goroutine to handle incoming web requests
	go func() {
		log.Fatal(http.ListenAndServe("192.168.1.22:8090", nil))
	}()

	// set up a goroutine to periodically Send information
	// updates to all connected clients
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				var data []byte
				for c := range h.Clients {
					latencyBuf := make([]byte, 2)
					binary.LittleEndian.PutUint16(latencyBuf, c.avgLatency)
					data = append(data, c.ID)
					data = append(data, latencyBuf...)
				}

				h.broadcast <- append([]byte{ServerInfo}, data...)
			}
		}
	}()
	for {
		select {
		case client := <-h.register:
			h.Clients[client] = true

		case client := <-h.unregister:
			h.player1.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				// check if this was one of the players
				if h.player1 != nil && client == h.player1.c {
					h.player1.clientClose <- struct{}{}
				}
				if h.player2 != nil && client == h.player2.c {
					h.player2.clientClose <- struct{}{}
				}

				id := client.Metadata.RemoteAddr
				delete(h.Clients, client)
				close(client.Send)

				// notify clients that this client is closing
				for c := range h.Clients {
					select {
					case c.Send <- append([]byte{ClientClosing}, []byte(id)...):
					default:
					}
				}

				// notify the next client that it can join
				// if there is one available
				newClient := h.nextClient()
				if newClient != nil {
					h.player1.clientConnect <- newClient
				}
			}
			h.player1.mu.Unlock()
		case message := <-h.broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

// NewClient creates a new client registered to the hub
// and returns it.
func (h *Hub) NewClient(conn *websocket.Conn, r *http.Request) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.CurrentID++

	client := &Client{
		hub:    h,
		conn:   conn,
		Send:   make(chan []byte, 256),
		ID:     h.CurrentID,
		player: make(chan []byte, 256),
		Metadata: struct {
			RemoteAddr string
			UserAgent  string
			Username   string
		}{RemoteAddr: r.RemoteAddr, UserAgent: r.Header.Get("User-Agent")},
		connectedAt: time.Now(),
	}
	h.register <- client
	return client
}

// NewPlayer upgrades a Client to a Player and returns it.
func NewPlayer(rom []byte, hub *Hub) *Player {
	// create a new Game Boy instance
	g := gameboy.NewGameBoy(rom)

	return &Player{nil, hub, make(chan struct{}, 10), make(chan *Client, 10), make(chan *Client, 10),
		g,
		make(chan joypad.Button, 10),
		make(chan joypad.Button, 10),
		newCache(16384),
		newCache(1024),
		make([]byte, 92160),
		0, sync.Mutex{},
	}
}

// SendAll sends a message to all connected clients
func (h *Hub) SendAll(message []byte) {
	h.broadcast <- message
}

func (h *Hub) send(client *Client, message []byte) {
	select {
	case client.Send <- message:
		// sent
		return
	}
}

func (h *Hub) sendAllButClient(client *Client, message []byte) {
	for c := range h.Clients {
		if c != client {
			h.send(c, message)
		}
	}
}

// nextClient returns the next client in the list awaiting
// player upgrade by comparing the value of each connectedAt
// field. Used when the original player disconnects and a new
// one is needed.
func (h *Hub) nextClient() *Client {
	var next *Client
	for c := range h.Clients {
		if next == nil || c.connectedAt.Before(next.connectedAt) {
			next = c
		}
	}

	return next
}

func tcpInfo(conn *net.TCPConn) (*unix.TCPInfo, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}

	var info *unix.TCPInfo
	ctrlErr := raw.Control(func(fd uintptr) {
		info, err = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
	})
	switch {
	case ctrlErr != nil:
		return nil, ctrlErr
	case err != nil:
		return nil, err
	}

	return info, nil
}
