package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	genkey "github.com/waterdragen/akl-ws/genkey"

	gin "github.com/gin-gonic/gin"
	uuid "github.com/google/uuid"
	websocket "github.com/gorilla/websocket"
)

// Require Go 1.21^
// install all dependencies: `go mod download`

var connUsersData *ConnUsers = NewConnUsers()

func main() {
	gin.SetMode(gin.ReleaseMode)
	const PORT string = ":9002"
	log.Println("Running server on http://localhost" + PORT)

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

	err := router.Run("localhost" + PORT)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func genkeyWebsocket(conn *websocket.Conn) {
	connID := generateConnID()

	defer func() {
		connUsersData.Pop(connID)
		conn.Close()
	}()

	for {
		genkeyProcess(conn, connID)
	}
}

func genkeyProcess(conn *websocket.Conn, connID uint32) {
	defer func() {
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack())
			stackTrace = strings.Join(strings.Split(stackTrace, "\n")[0:5], "\n")
			errMsg := fmt.Sprintf("%s\n%s", r, stackTrace)
			conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
			log.Println(errMsg)
		}
	}()

	// Read message from the websock et client
	_, message, err := conn.ReadMessage()
	if err != nil {
		// Disconnected
		return
	}

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
	userData = genkeyMain.GetUserData()

	if userData.Interactive.InInteractive {
		genkeyInteractive := genkey.NewGenkeyInteractive(conn, userData)
		genkeyInteractive.InteractiveSubsequent(string(message))
	} else {
		genkeyMain.Run(string(message))
	}

	if userData.Interactive.InInteractive {
		genkeyMain.SendMessage("[HOLD]")
	} else {
		genkeyMain.SendMessage("[DONE]")
	}

	// Store UserData to sync map
	connUsersData.Add(connID, genkeyMain.GetUserData())
}

func generateConnID() uint32 {
	return uuid.New().ID()
}
