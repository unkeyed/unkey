package base58

// Encode converts a byte slice to a Base58 encoded string using the Bitcoin alphabet.
//
// This function handles leading zeros correctly by encoding them as '1' characters.
// The algorithm uses a single-pass approach with optimized buffer sizing to minimize
// memory allocations for typical input sizes.
//
// The input can be any byte slice, including empty slices and slices containing
// only zero bytes. Empty input returns an empty string, while zero bytes are
// encoded as '1' characters.
//
// This implementation is vendored from https://github.com/mr-tron/base58 but
// maintained internally to avoid external dependencies.
func Encode(buf []byte) string {
	size := len(buf)

	zcount := 0
	for zcount < size && buf[zcount] == 0 {
		zcount++
	}

	// It is crucial to make this as short as possible, especially for
	// the usual case of bitcoin addrs
	size = zcount +
		// This is an integer simplification of
		// ceil(log(256)/log(58))
		(size-zcount)*555/406 + 1

	out := make([]byte, size)

	var i, high int
	var carry uint32

	high = size - 1
	for _, b := range buf {
		i = size - 1
		for carry = uint32(b); i > high || carry != 0; i-- {
			carry = carry + 256*uint32(out[i])
			out[i] = byte(carry % 58)
			carry /= 58
		}
		high = i
	}

	// Determine the additional "zero-gap" in the buffer (aside from zcount)
	for i = zcount; i < size && out[i] == 0; i++ {
	}

	// Now encode the values with actual alphabet in-place
	val := out[i-zcount:]
	size = len(val)
	for i = 0; i < size; i++ {
		out[i] = alphabet[val[i]]
	}

	return string(out[:size])
}
