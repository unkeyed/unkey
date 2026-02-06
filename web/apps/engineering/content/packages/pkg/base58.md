---
title: base58
description: "implements Base58 encoding for binary data"
---

Package base58 implements Base58 encoding for binary data.

Base58 is an encoding scheme that represents binary data in a human-readable ASCII format while avoiding visually similar characters. This implementation uses the Bitcoin alphabet: "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz".

DISCLAIMER: This implementation is copied from [https://github.com/mr-tron/base58](https://github.com/mr-tron/base58) and maintained internally to avoid external dependencies in critical paths. The encoding logic is identical to the original implementation.

### Usage

Basic encoding:

	data := []byte("Hello World")
	encoded := base58.Encode(data)
	fmt.Println(encoded) // Output: JxF12TrwUP45BMd

### Performance

This implementation is optimized for the common case of encoding small to medium sized byte arrays (up to a few hundred bytes). It uses a single-pass algorithm with minimal allocations and performs identically to the reference implementation.

## Constants

alphabet defines the Bitcoin Base58 character set.

This alphabet excludes visually similar characters (0, O, I, l) to reduce transcription errors. It's the standard alphabet used by Bitcoin and many other cryptocurrency applications.
```go
const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
```


## Functions

### func Encode

```go
func Encode(buf []byte) string
```

Encode converts a byte slice to a Base58 encoded string using the Bitcoin alphabet.

This function handles leading zeros correctly by encoding them as '1' characters. The algorithm uses a single-pass approach with optimized buffer sizing to minimize memory allocations for typical input sizes.

The input can be any byte slice, including empty slices and slices containing only zero bytes. Empty input returns an empty string, while zero bytes are encoded as '1' characters.

This implementation is vendored from [https://github.com/mr-tron/base58](https://github.com/mr-tron/base58) but maintained internally to avoid external dependencies.

