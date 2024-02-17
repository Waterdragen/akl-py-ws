package main

import (
	"log"
	"net/http"
	"strings"
	"unsafe"

	genkey "github.com/waterdragen/akl-ws/genkey"

	gin "github.com/gin-gonic/gin"
	websocket "github.com/gorilla/websocket"
)

// Require Go 1.21^
// install all dependencies: `go mod download`

var connUsersData *ConnUsers = NewConnUsers()

func main() {
	const PORT string = ":9002"

	router := gin.Default()
	router.LoadHTMLGlob("../templates/*")

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		},
	}

	handleRoute := func(c *gin.Context) {
		tail := strings.TrimSuffix(c.Param("tail"), "/")

		if tail == "genkey" {
			var h http.Header = http.Header{}
			h.Set("title", "text/html")
			conn, err := upgrader.Upgrade(c.Writer, c.Request, h)
			if err != nil {
				c.HTML(http.StatusBadRequest, "bad_request.html", nil)
				return
			}

			go genkeyWebsocket(conn)

		} else {
			c.HTML(http.StatusBadRequest, "bad_request.html", nil)
		}
	}

	router.GET("/go/:tail/", handleRoute)
	router.GET("/go/:tail", handleRoute)

	// Default service
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusBadRequest, "bad_request.html", nil)
	})

	err := router.Run(PORT)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func genkeyWebsocket(conn *websocket.Conn) {
	defer func() {
		connUsersData.Pop(generateConnID(conn))
		conn.Close()
	}()

	for {
		// Read message from the websock et client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Failed to read message from Websocket:", err)
			break
		}

		connID := generateConnID(conn)

		// Check if any cached data exists for this user
		var userData *genkey.UserData

		userDataObj, hasUserData := connUsersData.Get(connID)
		if !hasUserData {
			userData = nil
		} else {
			userData = userDataObj
		}

		// Run genkey
		genkeyMain := genkey.NewGenkeyMain(conn, userData)
		genkeyMain.Run()

		// Store UserData to sync map
		connUsersData.Add(connID, genkeyMain.GetUserData())

		log.Printf("Received message: %s", message)
	}
}

func generateConnID(conn *websocket.Conn) uint64 {
	return uint64(uintptr((unsafe.Pointer(conn))))
}
