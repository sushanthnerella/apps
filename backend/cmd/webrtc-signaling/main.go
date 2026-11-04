package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/purohit-platform/backend/internal/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin for development & mobile clients
	},
}

type Client struct {
	ID     string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type RoomHub struct {
	// roomID -> clientID -> *Client
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
	fmt.Printf("Client %s joined room %s (Total peers: %d)\n", c.ID, c.RoomID, len(h.rooms[c.RoomID]))
}

func (h *RoomHub) Unregister(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if room, exists := h.rooms[c.RoomID]; exists {
		delete(room, c.ID)
		close(c.Send)
		if len(room) == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	fmt.Printf("Client %s left room %s\n", c.ID, c.RoomID)
}

func (h *RoomHub) BroadcastToRoom(senderID string, roomID string, msgBytes []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	roomID := r.URL.Query().Get("roomId")
	clientID := r.URL.Query().Get("clientId")

	if roomID == "" || clientID == "" {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"missing roomId or clientId"}`))
		conn.Close()
		return
	}

	client := &Client{
		ID:     clientID,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	hub.Register(client)

	// Writer pump
	go func() {
		defer conn.Close()
		for msg := range client.Send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}()

	// Reader pump
	defer func() {
		hub.Unregister(client)
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var signal models.WebRTCMessage
		if err := json.Unmarshal(message, &signal); err == nil {
			signal.SenderID = clientID
			signal.RoomID = roomID
			broadcastData, _ := json.Marshal(signal)
			hub.BroadcastToRoom(clientID, roomID, broadcastData)
		}
	}
}

func main() {
	http.HandleFunc("/ws/webrtc", handleWebSocket)

	port := ":8082"
	fmt.Printf("📹 Puja Vaahini WebRTC Signaling Server listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
