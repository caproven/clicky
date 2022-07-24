package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/caproven/bytesize"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var pool = connPool{
	conns: make(map[*websocket.Conn]bool),
	m:     &sync.Mutex{},
}

var count uint64 = 0

type connPool struct {
	conns map[*websocket.Conn]bool // stored as map to simplify removal
	m     *sync.Mutex
}

func (p connPool) addClient(c *websocket.Conn) {
	p.m.Lock()
	p.conns[c] = true
	log.Printf("added client: %p\n", c)
	p.m.Unlock()
}

func (p connPool) removeClient(c *websocket.Conn) {
	p.m.Lock()
	delete(p.conns, c)
	log.Printf("removed client: %p\n", c)
	p.m.Unlock()
}

func (p connPool) broadcast(mt int, data []byte) {
	p.m.Lock()
	for c := range p.conns {
		c.WriteMessage(mt, data)
	}
	p.m.Unlock()
}

func main() {
	http.HandleFunc("/clicky", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}
		conn.SetReadLimit(3 * bytesize.Kilobyte)
		defer conn.Close()
		pool.addClient(conn)
		defer pool.removeClient(conn)

		// sync client with initial state
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprint(count)))

		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
				}
				break
			}
			input := string(message)
			log.Println("got message:", input)

			atomic.AddUint64(&count, 1)

			pool.broadcast(mt, []byte(fmt.Sprint(count)))
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "site.html")
	})

	http.ListenAndServe(":8080", nil)
}
