package main

import (
	"log"
	"net/http"
)

/* wrapper for CORS */

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
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
	mux.HandleFunc("/sitInOrOut", s.sitInOrOutHandler)

	log.Println("Server on :8080 | POST /join  POST /leave  GET /players  (use ?room=2 for room 2)")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
