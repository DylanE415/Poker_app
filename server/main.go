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
	// 1) load users.json
	uByName, _, err := loadUsersJSON("../users/users.json")
	if err != nil {
		log.Fatal(err)
	}

	// 2) make server with rooms + users, and init sessions
	s := &Server{
		room1:           newRoom(1, 30.0, 100.0),
		room2:           newRoom(2, 30.0, 100.0),
		usersByUsername: uByName,
	}
	s.initSessions()

	// launch room loops
	go s.room1.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.loginHandler)

	// wrapped in require auth to make sure the user is logged in(has a session cookie)
	mux.Handle("/join", s.requireAuth(http.HandlerFunc(s.joinHandler)))
	mux.Handle("/leave", s.requireAuth(http.HandlerFunc(s.leaveHandler)))
	mux.Handle("/players", s.requireAuth(http.HandlerFunc(s.playersHandler)))
	mux.Handle("/state", s.requireAuth(http.HandlerFunc(s.stateHandler)))
	mux.Handle("/action", s.requireAuth(http.HandlerFunc(s.setActionHandler)))
	mux.Handle("/sitInOrOut", s.requireAuth(http.HandlerFunc(s.sitInOrOutHandler)))

	log.Println("Server on :8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
