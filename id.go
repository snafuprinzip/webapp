package webapp

import (
	"crypto/rand"
	"fmt"
	"log"
)

// idSource contains the character base used when generating a random identifier.
const idSource = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const pwdSource = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&*()_-=+/?[]{}|<>~;:,."

// idSourceLen saves the length in a constant, so we don't look it up each time.
const idSourceLen = byte(len(idSource))
const pwdSourceLen = byte(len(pwdSource))

// GenerateID creates a prefixed random identifier.
func GenerateID(prefix string, length int) string {
	// Create an array with the correct capacity
	id := make([]byte, length)
	// Fill our array with random numbers
	_, err := rand.Read(id)
	if err != nil {
		log.Fatalf("Unable to read random numbers: %s\n", err)
	}

	// Replace each random number with an alphanumeric value
	for i, b := range id {
		id[i] = idSource[b%idSourceLen]
	}

	// Return the formatted id
	return fmt.Sprintf("%s_%s", prefix, string(id))
}

// GenerateRandomPassword is used to generate an initial  random password for the admin account
func GenerateRandomPassword(length int) string {
	// Create an array with the correct capacity
	id := make([]byte, length)
	// Fill our array with random numbers
	_, err := rand.Read(id)
	if err != nil {
		log.Fatalf("Unable to read random numbers: %s\n", err)
	}

	// Replace each random number with an alphanumeric value
	for i, b := range id {
		id[i] = pwdSource[b%pwdSourceLen]
	}

	// Return the formatted id
	return string(id)
}
