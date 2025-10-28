package handlers

import (
	myws "myapp/services/websocket"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func ServeWs(hub *myws.Hub, c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to upgrade connection to WebSocket",
		})
		return
	}

	client := &myws.Client{Hub: hub, Conn: conn, Send: make(chan myws.Message)}

	client.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
