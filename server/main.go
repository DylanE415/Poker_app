package main

import (
	"log"
	"net/http"
)

type Player struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Stack float64 `json:"stack"`
}

/* === main === */

func main() {
	// room takes in id, minStack
	s := &Server{
		room1: newRoom(1, 30.0, 100.0),
		room2: newRoom(2, 30.0, 100.0),
	}
	// launch goroutines
	go s.room1.run()
	go s.room2.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/join", s.joinHandler)
	mux.HandleFunc("/leave", s.leaveHandler)
	mux.HandleFunc("/players", s.playersHandler)
	mux.HandleFunc("/state", s.stateHandler)
	mux.HandleFunc("/action", s.setActionHandler)

	log.Println("Server on :8080 | POST /join  POST /leave  GET /players  (use ?room=2 for room 2)")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
