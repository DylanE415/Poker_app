package main

import (
	crand "crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"net/http"
)

// the server containing all rooms
type Server struct {
	room1           *Room
	room2           *Room
	usersByUsername map[string]User // you already have this
	usersByID       map[string]User
	sessionsMu      sync.RWMutex
	sessions        map[string]Session // sessionID -> session data
}

// Session keeps who is logged in and until when.
type Session struct {
	UserID    string
	ExpiresAt time.Time
}

// initalize the sessions map
func (s *Server) initSessions() {
	if s.sessions == nil {
		s.sessions = make(map[string]Session)
	}
}

// returns the room object for a particular room number
func (s *Server) getRoom(q string) *Room {
	if q == "2" {
		return s.room2
	} else if q == "1" {
		return s.room1
	}

	return nil
}

// newSessionID returns a 32-byte random token
func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// createSessionFor stores a new session and sets the cookie on the response.
func (s *Server) createSessionFor(w http.ResponseWriter, userID string) error {
	sid, err := newSessionID()
	if err != nil {
		return err
	}

	// lock so when we run in parallel we don't overwrite the map
	s.sessionsMu.Lock()
	s.sessions[sid] = Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(6 * time.Hour), // session lifetime
	}
	s.sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,                 // JS can't read it
		Secure:   true,                 // require HTTPS on the internet
		SameSite: http.SameSiteLaxMode, // or Strict if you prefer
		MaxAge:   6 * 60 * 60,          // seconds
	})
	return nil
}

// userIDFromRequest fetches the logged-in user ID from the session cookie.
func (s *Server) userIDFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session")
	if err != nil || cookie.Value == "" {
		return "", false
	}
	s.sessionsMu.RLock()
	sess, ok := s.sessions[cookie.Value]
	s.sessionsMu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return "", false
	}
	return sess.UserID, true
}

// requireAuth wraps handlers so only logged-in users can access them.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.userIDFromRequest(r); !ok {
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
