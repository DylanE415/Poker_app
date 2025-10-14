package main

//This file is for loading the users hashes into memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

type usersFile struct {
	Users []User `json:"users"`
}

// loadUsersJSON reads users.json, validates hashes, and returns maps:
// - byUsername (lowercased username -> User)
// - byID (id -> User)
func loadUsersJSON(path string) (map[string]User, map[string]User, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var uf usersFile
	if err := json.NewDecoder(f).Decode(&uf); err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(uf.Users) == 0 {
		return nil, nil, errors.New("no users in users.json")
	}

	byUsername := make(map[string]User, len(uf.Users))
	byID := make(map[string]User, len(uf.Users))

	for _, u := range uf.Users {
		if u.ID == "" || u.Username == "" || u.PasswordHash == "" {
			return nil, nil, fmt.Errorf("user entry missing fields: %+v", u)
		}
		// validate bcrypt hash format (also extracts cost)
		if _, err := bcrypt.Cost([]byte(u.PasswordHash)); err != nil {
			return nil, nil, fmt.Errorf("bad bcrypt hash for user %q: %w", u.Username, err)
		}
		key := strings.ToLower(strings.TrimSpace(u.Username))
		if _, dup := byUsername[key]; dup {
			return nil, nil, fmt.Errorf("duplicate username %q", u.Username)
		}
		if _, dup := byID[u.ID]; dup {
			return nil, nil, fmt.Errorf("duplicate id %q", u.ID)
		}
		byUsername[key] = u
		byID[u.ID] = u
	}

	return byUsername, byID, nil
}
