package handlers

import (
	"log"
	"net/http"

	"transit-backend/internal/hub"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

type WebSocketHandler struct {
	hub *hub.Hub
}

func NewWebSocketHandler(h *hub.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: h}
}

func (h *WebSocketHandler) HandleWS(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println(err)
		return err
	}

	client := &hub.Client{Hub: h.hub, Conn: conn, Send: make(chan interface{}, 256)}
	client.Hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()

	return nil
}
