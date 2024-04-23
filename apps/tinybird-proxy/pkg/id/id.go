package id

import "math/rand"



func New(length ...int) string {
		if len(length) == 0 {
				length = append(length, 8)
		}
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, length[0])
		for i := range b {
				b[i] = charset[rand.Intn(len(charset))]
		}
		return string(b)
}