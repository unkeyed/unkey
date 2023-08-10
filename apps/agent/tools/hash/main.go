package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func hash(s string) string {
	hash := sha256.New()
	hash.Write([]byte(s))

	buf := hash.Sum(nil)
	fmt.Println("buf", buf)
	return base64.StdEncoding.EncodeToString(buf)
}

// Return the hash of the key used for authentication
func main() {

	fmt.Println(hash("key_LhUjo9W6YwpVqMLotu8CpH"))

}

// key: 'key_LhUjo9W6YwpVqMLotu8CpH',
//   hash: 'iMZufy26hPf9b7fgAZw3c6aOgK4WyxK8QXDEVIBYAxM='
