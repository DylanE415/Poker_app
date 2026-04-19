package main

import (
	crand "crypto/rand"
	"encoding/base64"
	"net"
	"strings"
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
	loginMu         sync.Mutex
	ipLoginState    map[string]ipLoginState //ip address -> login attempts
	singupMu        sync.Mutex
	ipSignupState   map[string]ipSignupState
	usersLock       sync.RWMutex
}

const (
	maxLoginFailures  = 10
	maxSingupAttempts = 3
	lockoutPeriod     = 2 * time.Minute
)

type ipLoginState struct {
	failures  int
	lockUntil time.Time
	lastTouch time.Time
}

type ipSignupState struct {
	attempts  int
	lockUntil time.Time
	lastTouch time.Time
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
		Secure:   false,                // require HTTPS on the internet(set false when testing)
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

func (s *Server) getIpAddress(request *http.Request) string {
	//get original ip address from X-Forwarded-For
	if xff := request.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	//if there is no X-Forwarded-For, use the real ip
	if xrip := request.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	//if there is no X-Forwarded-For or X-Real-IP, use the remote address
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr

}

/*
------------------ Login ----------------------------------------------------------------------------------------------------------------
----------------------------------------------------------------------------------------------------------------------------------------
---------------------------------------------------------------------------------------------------------------------------------------------
*/

func (s *Server) canAttemptLogin(ip string) (ok bool, wait time.Duration) {
	//locks the ipLoginState map and unlocks when func returns
	s.loginMu.Lock()
	defer s.loginMu.Unlock()

	state := s.ipLoginState[ip]
	now := time.Now()
	if now.Before(state.lockUntil) {
		return false, time.Until(state.lockUntil)
	}
	// If the lock expired, let them try again; failures remain until success (or you can reset here).
	return true, 0
}

func (s *Server) recordLoginFail(ip string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()

	now := time.Now()
	state := s.ipLoginState[ip]

	// If their lock expired earlier, we keep counting from current failures;
	// you can optionally reset st.failures=0 if st.lockUntil.Before(now).
	state.failures++
	state.lastTouch = now
	if state.failures >= maxLoginFailures {
		state.lockUntil = now.Add(lockoutPeriod)
	}
	s.ipLoginState[ip] = state
}

func (s *Server) recordLoginSuccess(ip string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	delete(s.ipLoginState, ip)
}

/*
------------------ SIGNUP ----------------------------------------------------------------------------------------------------------------
----------------------------------------------------------------------------------------------------------------------------------------
---------------------------------------------------------------------------------------------------------------------------------------------
*/
func (s *Server) canAttemptSignup(ip string) (ok bool, wait time.Duration) {
	//locks the ipLoginState map and unlocks when func returns
	s.singupMu.Lock()
	defer s.singupMu.Unlock()

	state := s.ipSignupState[ip]
	now := time.Now()
	if now.Before(state.lockUntil) {
		return false, time.Until(state.lockUntil)
	}
	// If the lock expired, let them try again; failures remain until success (or you can reset here).
	return true, 0
}

func (s *Server) recordSignupAttempt(ip string) {
	s.singupMu.Lock()
	defer s.singupMu.Unlock()

	now := time.Now()
	state := s.ipSignupState[ip]

	// If their lock expired earlier, we keep counting from current failures;
	// you can optionally reset st.failures=0 if st.lockUntil.Before(now).
	state.attempts++
	state.lastTouch = now
	if state.attempts > maxSingupAttempts {
		state.lockUntil = now.Add(lockoutPeriod)
	}
	s.ipSignupState[ip] = state
}
