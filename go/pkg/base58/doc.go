// Package base58 implements Base58 encoding for binary data.
//
// Base58 is an encoding scheme that represents binary data in a human-readable
// ASCII format while avoiding visually similar characters. This implementation
// uses the Bitcoin alphabet: "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz".
//
// DISCLAIMER: This implementation is copied from https://github.com/mr-tron/base58
// and maintained internally to avoid external dependencies in critical paths.
// The encoding logic is identical to the original implementation.
//
// # Usage
//
// Basic encoding:
//
//	data := []byte("Hello World")
//	encoded := base58.Encode(data)
//	fmt.Println(encoded) // Output: JxF12TrwUP45BMd
//
// # Performance
//
// This implementation is optimized for the common case of encoding small to medium
// sized byte arrays (up to a few hundred bytes). It uses a single-pass algorithm
// with minimal allocations and performs identically to the reference implementation.
package base58
