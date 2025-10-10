// room number reqest to int
package main

import (
	"fmt"
	"strconv"
)

func room_request_to_int(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("missing required room id")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid room id: %s", s)
	}
	if n != 1 && n != 2 {
		return 0, fmt.Errorf("room id must be 1 or 2")
	}
	return n, nil
}

// check if a string is in a slice
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// get player by id from hand
func FindPlayerIndex(H *Hand, id string) int {
	for i, p := range H.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}
