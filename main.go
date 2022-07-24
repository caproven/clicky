package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/caproven/bytesize"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func main() {
	http.HandleFunc("/clicky", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()
		conn.SetReadLimit(3 * bytesize.Kilobyte)

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
			count, _ := strconv.Atoi(input)

			err = conn.WriteMessage(mt, []byte(fmt.Sprint(count+1)))
			if err != nil {
				log.Println("write failed:", err)
				break
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "site.html")
	})

	http.ListenAndServe(":8080", nil)
}
