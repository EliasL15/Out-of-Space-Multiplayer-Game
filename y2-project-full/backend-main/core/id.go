package core

import (
	_ "embed"
	"math/rand"
	"strings"
)

// randomDigit returns a random ASCII digit (0-10) as a rune.
func randomDigit() rune {
	return rune('0' + (rand.Uint32() % 10))
}

// randomLobbyCode generates a random four-digit number for use as a lobby ID.
func randomLobbyCode() string {
	return string([]rune{randomDigit(), randomDigit(), randomDigit(), randomDigit()})
}

//go:embed usernames.txt
var usernamesRaw string
var usernamesSplit []string = nil

// usernames returns a slice of allowed player names.
func usernames() []string {
	if usernamesSplit == nil {
		usernamesSplit = strings.Split(strings.TrimSpace(usernamesRaw), "\n")
	}

	return usernamesSplit
}

// randomUsername generates a random username from our list of appropriate names.
func randomUsername() string {
	usernames := usernames()

	return usernames[int(rand.Uint32())%len(usernames)]
}
