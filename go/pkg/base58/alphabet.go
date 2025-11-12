package base58

// alphabet defines the Bitcoin Base58 character set.
//
// This alphabet excludes visually similar characters (0, O, I, l) to reduce
// transcription errors. It's the standard alphabet used by Bitcoin and
// many other cryptocurrency applications.
const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
