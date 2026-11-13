package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/purohit-platform/backend/internal/httpsecurity"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: httpsecurity.AllowedOrigin,
}

type Client struct {
	ID     string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type RoomHub struct {
	rooms map[string]map[string]*Client
	mutex sync.RWMutex
}

var hub = &RoomHub{
	rooms: make(map[string]map[string]*Client),
}

func (h *RoomHub) Register(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, exists := h.rooms[c.RoomID]; !exists {
		h.rooms[c.RoomID] = make(map[string]*Client)
	}
	h.rooms[c.RoomID][c.ID] = c
	fmt.Printf("🎥 WebRTC: Peer %s joined room %s\n", c.ID, c.RoomID)
}

func (h *RoomHub) Unregister(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if room, exists := h.rooms[c.RoomID]; exists {
		if room[c.ID] != c {
			return
		}
		delete(room, c.ID)
		close(c.Send)
		if len(room) == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	fmt.Printf("🎥 WebRTC: Peer %s left room %s\n", c.ID, c.RoomID)
}

func (h *RoomHub) BroadcastToRoom(senderID, roomID string, msgBytes []byte) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if room, exists := h.rooms[roomID]; exists {
		for clientID, client := range room {
			if clientID != senderID {
				select {
				case client.Send <- msgBytes:
				default:
					close(client.Send)
					delete(room, clientID)
				}
			}
		}
	}
}

type ChatHub struct {
	rooms map[string]map[string]*Client
	mutex sync.RWMutex
}

var chatHub = &ChatHub{
	rooms: make(map[string]map[string]*Client),
}

func (h *ChatHub) Register(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, exists := h.rooms[c.RoomID]; !exists {
		h.rooms[c.RoomID] = make(map[string]*Client)
	}
	h.rooms[c.RoomID][c.ID] = c
	fmt.Printf("💬 Chat: Peer %s joined room %s\n", c.ID, c.RoomID)
}

func (h *ChatHub) Unregister(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if room, exists := h.rooms[c.RoomID]; exists {
		if room[c.ID] != c {
			return
		}
		delete(room, c.ID)
		close(c.Send)
		if len(room) == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	fmt.Printf("💬 Chat: Peer %s left room %s\n", c.ID, c.RoomID)
}

func (h *ChatHub) BroadcastToRoom(senderID, roomID string, msgBytes []byte) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if room, exists := h.rooms[roomID]; exists {
		for clientID, client := range room {
			if clientID != senderID {
				select {
				case client.Send <- msgBytes:
				default:
					close(client.Send)
					delete(room, clientID)
				}
			}
		}
	}
}
