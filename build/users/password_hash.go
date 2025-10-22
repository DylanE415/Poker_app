package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	print("register a new password\n")
	pw := ""
	fmt.Scanln(&pw)
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost) // cost 10
	fmt.Println(string(hash))
}
