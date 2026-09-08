package paseto

import "encoding/binary"

// preAuthEncode implements PASETO Pre-Authentication Encoding. It has no
// decoder because PAE only separates fields before authentication.
func preAuthEncode(pieces ...[]byte) []byte {
	encoded := make([]byte, 0)
	encoded = appendLittleEndian64(encoded, len(pieces))
	for _, piece := range pieces {
		encoded = appendLittleEndian64(encoded, len(piece))
		encoded = append(encoded, piece...)
	}
	return encoded
}

func appendLittleEndian64(destination []byte, value int) []byte {
	var encoded [8]byte
	// A Go slice cannot reach 2^63 bytes. Converting its length therefore keeps
	// the most significant bit clear as required by the PASETO specification.
	binary.LittleEndian.PutUint64(encoded[:], uint64(value))
	return append(destination, encoded[:]...)
}
