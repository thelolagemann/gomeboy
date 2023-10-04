package web

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"sync"
	"time"
)

type Client struct {
	mu       sync.RWMutex
	hub      *hub
	conn     *websocket.Conn
	Send     chan []byte
	ID       uint8
	Metadata struct {
		RemoteAddr string
		UserAgent  string
		Username   string
	}
	avgLatency  uint16
	connectedAt time.Time

	player chan []byte
}

func (c *Client) ReadPump() {
	// deferred function to handle unregistering client
	// and closing connection
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		c.avgLatency = 0
		// c.mu.Unlock()
	}()

	// read messages from client
read:
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return // connection closed
		}

		switch message[0] {
		case 10: // system related messages
			c.hub.mu.Lock()

			switch message[1] {
			case Compression:
				c.hub.compression = message[2] == 1
			case CompressionLevel:
				c.hub.compressionLevel = int(message[2])
			case FramePatching:
				c.hub.framePatching = message[2] == 1
			case FrameSkipping:
				c.hub.frameSkipping = message[2] == 1
			case FrameCaching:
			// TODO implement
			case RegisterUsername:
				c.Metadata.Username = string(message[2:])

				message = []byte{}
				message = append(message, c.Metadata.RemoteAddr...)
				message = append(message, 0)
				message = append(message, c.Metadata.UserAgent...)
				message = append(message, 0)
				message = append(message, c.Metadata.Username...)
				message = append(message, 0)
				message = append(message, c.ID)
				message = append(message, byte('\n'))

				c.Send <- append([]byte{ClientInfo, RegisterUsername, 0xFF}, message...)
				c.hub.sendAllButClient(c, append([]byte{ClientInfo, RegisterUsername}, message...))
				c.hub.mu.Unlock()
				continue read // special case of register username handled
			case Player2Confirmation:
				fmt.Println("recieved player 2 confirmation")
				// has a player 2 already attached to this client
				if c.hub.player2.c != nil {
					continue read // skip this message
				}

				fmt.Println("upgrading player 2")

				/* TODO reimplement this
				if !c.hub.player2.gb.IsRunning() {
					go c.hub.player2.Start()
				}*/

				c.hub.player2.clientConnect <- c
			}

			c.hub.sendAllButClient(c, append([]byte{ClientInfo, message[1]}, message[2:]...))
			c.hub.mu.Unlock()
			continue read
		case 255: // websocket client request close
			c.hub.sendAllButClient(c, append([]byte{ClientClosing}, []byte(c.Metadata.RemoteAddr)...)) // TODO send ID instead of IP
			c.hub.unregister <- c
			return
		default:
			// send through to player if not nil
			if c.player != nil {
				c.player <- message
			}
		}
	}
}

func (c *Client) WritePump() {
	// deferred function to handle unregistering client
	// and closing connection
	defer func() {
		c.hub.unregister <- c
		c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		c.mu.Unlock()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.mu.Lock()
			// connection hub closed the connection
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// try to write message to client
			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				return
			}

			// update average latency
			info, err := tcpInfo(c.conn.UnderlyingConn().(*net.TCPConn))
			if err != nil {
				return
			}
			c.avgLatency = ((c.avgLatency * 9) + uint16(info.Rtt/1000)) / 10
			c.mu.Unlock()
		}
	}
}
