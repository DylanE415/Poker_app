package main

import (
	"log"
	"net/http"
)

func (s *Server) handleRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/join", s.requireAuth(http.HandlerFunc(s.joinHandler)))
	mux.Handle("/leave", s.requireAuth(http.HandlerFunc(s.leaveHandler)))
	mux.Handle("/players", s.requireAuth(http.HandlerFunc(s.playersHandler)))
	mux.Handle("/state", s.requireAuth(http.HandlerFunc(s.getStateHandler)))
	mux.Handle("/action", s.requireAuth(http.HandlerFunc(s.setActionHandler)))
	mux.Handle("/sitInOrOut", s.requireAuth(http.HandlerFunc(s.sitInOrOutHandler)))

	///the static pages
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.ServeFile(w, r, "./static/login.html")
		case http.MethodPost:
			s.loginHandler(w, r)
		default:
			http.Error(w, "use GET or POST", http.StatusMethodNotAllowed)
		}
	})

	mux.Handle("/play", s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "use GET", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "./static/play.html")
	})))

	//for any URL starting with /static/, strip that prefix and serve the remaining path from the local ./static/ folder
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// redirect to login
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	return mux
}

/* === main === */
func main() {
	// load users
	uByName, uByID, err := loadUsersJSON("../users/users.json")
	if err != nil {
		log.Fatal(err)
	}

	s := &Server{
		room1:           newRoom(1, 20.0, 40),
		room2:           newRoom(2, 30.0, 100.0),
		usersByUsername: uByName,
		usersByID:       uByID, // <-- you use this in handlers; make sure it’s populated
		sessions:        make(map[string]Session),
		loginAttempts:   make(map[string]loginState),
	}
	s.initSessions()

	go s.room1.run()

	// Same-origin setup: you can serve without CORS.
	// If you *must* keep CORS, see the note below.
	log.Println("Server on :80")
	log.Fatal(http.ListenAndServe(":80", s.handleRoutes()))
}
