package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// generateSignature computes the HMAC-SHA256 signature of the payload
// and returns it in the format expected by GitHub: "sha256=<hex>"
func generateSignature(payload []byte, secret string) string {
	mac := hmacSHA256([]byte(secret), payload)
	return fmt.Sprintf("sha256=%x", mac)
}

// hmacSHA256 computes the HMAC-SHA256 of data using the provided key
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
