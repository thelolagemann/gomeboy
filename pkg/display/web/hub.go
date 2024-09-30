package web

import (
	"encoding/binary"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/thelolagemann/gomeboy/internal/types"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type hub struct {
	clients          map[*Client]bool
	player1, player2 *Player

	broadcast            chan []byte
	register, unregister chan *Client

	compression      bool
	compressionLevel int
	framePatching    bool
	framePatchRatio  int
	frameSkipping    bool
	currentID        uint8

	mu sync.Mutex
}

func (w *hub) run() error {
	// create http handler for client connections
	http.HandleFunc("/", func(wr http.ResponseWriter, r *http.Request) {
		wr.Header().Set("Access-Control-Allow-Origin", "*")

		// upgrade the connection to a websocket connection
		conn, err := upgrader.Upgrade(wr, r, nil)
		if err != nil {
			// TODO handle err
			return
		}

		// create new client
		c := w.newClient(conn, r)

		// spawn read/write pumps
		go c.ReadPump()
		go c.WritePump()

		// send initial data information
		c.Send <- []byte{ClientInfo, ClientStatus, w.info(), uint8(w.compressionLevel), uint8(w.framePatchRatio)}

		// inform players of the new clients
		w.player1.clientSync <- c
		if w.player2.c != nil {
			w.player2.clientSync <- c
		}

		// synchronize clients to connecting client
		var data []byte
		for cl := range w.clients {
			if c == cl {
				continue // skip self
			}

			data = append(data, c.Metadata.RemoteAddr...)
			data = append(data, 0)
			data = append(data, cl.Metadata.UserAgent...)
			data = append(data, 0)
			data = append(data, cl.Metadata.Username...)
			data = append(data, 0)
			data = append(data, cl.ID)
			data = append(data, byte('\n'))
		}

		if len(data) > 0 {
			// remove last newline to avoid issues with JS
			data = data[:len(data)-1]
		}

		c.Send <- append([]byte{ClientListSync}, data...)
	})

	// setup goroutines

	// web server
	go func() {
		log.Fatal(http.ListenAndServe(":8090", nil))
	}()

	// periodic info updates
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				// build information
				var data []byte
				for c := range w.clients {
					latencyBuf := make([]byte, 2)
					binary.LittleEndian.PutUint16(latencyBuf, c.avgLatency)
					data = append(data, c.ID)
					data = append(data, latencyBuf...)
				}

				// broadcast information
				w.broadcast <- append([]byte{ServerInfo}, data...)
			}
		}
	}()

	// handle broadcasting
	for {
		select {
		case c := <-w.register:
			w.clients[c] = true
		case c := <-w.unregister:
			w.player1.mu.Lock()
			// is this client still registered
			if _, ok := w.clients[c]; ok {
				// was it one of the players?
				if w.player1 != nil && c == w.player1.c {
					w.player1.clientClose <- struct{}{}
				}
				if w.player2 != nil && c == w.player2.c {
					w.player2.clientClose <- struct{}{}
				}

				id := c.Metadata.RemoteAddr
				delete(w.clients, c)

				// notify connected clients that this client has disconnected
				for c := range w.clients {
					select {
					case c.Send <- append([]byte{ClientClosing}, id...):
					default:
					}
				}

				// notify the next client that it can join if there is one available
				if next := w.nextPlayer(); next != nil {
					w.player1.clientConnect <- next
				}
			}
			w.player1.mu.Unlock()
		case msg := <-w.broadcast:
			for c := range w.clients {
				select {
				case c.Send <- msg:
				default:
					close(c.Send)
					delete(w.clients, c)
				}
			}
		}
	}
}

// info returns a byte of information containing the various
// hub settings. The byte is constructed as follows:
//
//	Bit 0: Running status of player 1
//	Bit 1: Running status of player 2
//	Bit 2: Compression enabled
//	Bit 3: Frame patching enabled
//	Bit 4: Frame skipping enabled
//	Bit 5: Player 1 paused
//	Bit 6: Player 2 paused
func (w *hub) info() byte {
	info := uint8(0)
	if w.player1.gb != nil {
		if !w.player1.gb.Paused() {
			info |= types.Bit0
		}
		if w.player1.gb.Paused() {
			info |= types.Bit5
		}
	}

	if w.player2.gb != nil {
		if !w.player2.gb.Paused() {
			info |= types.Bit1
		}
		if w.player2.gb.Paused() {
			info |= types.Bit6
		}
	}

	if w.compression {
		info |= types.Bit2
	}
	if w.framePatching {
		info |= types.Bit3
	}
	if w.frameSkipping {
		info |= types.Bit4
	}

	fmt.Printf("%08b\n", info)

	return info
}

// nextPlayer returns the next client in the list awaiting
// player upgrade by comparing the value of each connectedAt
// field. Used when a player disconnects and a new player
// is able to take over.
func (w *hub) nextPlayer() *Client {
	var next *Client
	for c := range w.clients {
		if next == nil || c.connectedAt.Before(next.connectedAt) {
			next = c
		}
	}

	return next
}

// newClient creates a new client and registers it to the hub
func (w *hub) newClient(conn *websocket.Conn, r *http.Request) *Client {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentID++

	c := &Client{
		hub:    w,
		conn:   conn,
		Send:   make(chan []byte, 256),
		ID:     w.currentID,
		player: make(chan []byte, 256),
		Metadata: struct {
			RemoteAddr string
			UserAgent  string
			Username   string
		}{RemoteAddr: r.RemoteAddr, UserAgent: r.Header.Get("User-Agent")},
		connectedAt: time.Now(),
	}
	w.register <- c
	return c
}

// sendAllButClient sends a message to all connected clients except
// the one specified. Used for events such as username registration,
// where the client is the one that initiated the event, so is already
// aware of the registered username.
func (w *hub) sendAllButClient(client *Client, message []byte) {
	for c := range w.clients {
		if c == client {
			continue
		}
		c.Send <- message
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 16,
	WriteBufferSize: 1024 * 16,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
