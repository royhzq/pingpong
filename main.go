// Basic PING PONG websocket setup
// The purpose of PING PONG, where a PING is sent from server to client
// and a PONG is sent back to server by client om response.
// Pong response is a built-in response by most browsers. So nothing is needed to be done on client side
// Ping Pong is to determine whether a connection should be kept alive, from the client's regular PONG response
// The server should set a read deadline for the client to prevent idle connections from hogging resources
// If a PONG or other message is not received within the deadline, the connection will be automatically terminated by the websocket

package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	pongWait   = 5 * time.Second
	writeWait  = 10 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	log.Println("Start Reader")
	ws.SetReadLimit(512) // Max size in bytes from client - if breached, close conn
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		log.Println("PONG received")
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// For loop to constantly listen to messages from client
	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Println("Read Err:", err)
			return
		}
		log.Println(string(p))

	}

}

func writer(ws *websocket.Conn) {

	// Demonstrate sending PING messages to client
	// To keep connection alive
	log.Println("Start Writer")
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()

	// For loop to constantly ping client at every `writeWait` interval
	// Nested in select case to allow for more cases in the future
	// pingTicker is blocking in the for loop and will release every second
	for {
		select {
		case <-pingTicker.C:
			err := ws.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Println("Write Err:", err)
				return
			}
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			log.Println("PING")

		}
	}

}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	// Serves out home page
	tmpl := template.Must(template.ParseFiles("index.html"))
	var context = struct {
		Host string
	}{
		r.Host,
	}
	tmpl.Execute(w, &context)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	// Used to upgrade to websocket and call `writer()` `reader()` func
	// For each client connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println("Upgrader err: ", err)
		}
		return
	}
	go writer(ws)
	go reader(ws)
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", serveIndex)
	r.HandleFunc("/ws", serveWs)
	log.Println("Starting server...")
	http.ListenAndServe(":8000", r)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
