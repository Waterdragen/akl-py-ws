package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	PORT := ":9002"

	router := gin.Default()
	router.LoadHTMLGlob("../templates/*")

	handleRoute := func(c *gin.Context) {
		tail := strings.TrimSuffix(c.Param("tail"), "/")

		if tail == "main" {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				panic(err)
			}

			go handleWebsocket(conn)

		} else {
			c.HTML(http.StatusBadRequest, "bad_request.html", gin.H{})
		}
	}

	router.GET("/go/:tail/", handleRoute)
	router.GET("/go/:tail", handleRoute)

	// Default service
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusBadRequest, "bad_request.html", gin.H{})
	})

	err := router.Run(PORT)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func handleWebsocket(conn *websocket.Conn) {
	defer conn.Close()

	for {
		// Read message from the websock et client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Failed to read message from Websocket:", err)
			break
		}
		log.Printf("Received message: %s", message)

		// Send the message back to the Websocket client
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("Failed to write message to Websocket:", err)
			break
		}
	}
}
