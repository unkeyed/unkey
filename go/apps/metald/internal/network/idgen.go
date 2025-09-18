package network

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
)

// IDGenerator generates short, unique IDs for network devices
// Network interface names in Linux are limited to 15 characters,

// Generates an for the bridge, it's not guaranteed unique until it's saved to the DB
func GenerateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	id := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)[:12]

	return fmt.Sprintf("br-%s", id)
}
