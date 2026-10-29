package streamschema

import (
	"context"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	bufferBytesMax = 32 << 10
	// readsEmptyMax is how many consecutive (0, nil) reads are tolerated
	// before the reader is treated as stuck, matching bufio.
	readsEmptyMax = 100
)

var plainStringByte = func() (table [256]bool) {
	for b := 0x20; b < 0x80; b++ {
		table[b] = b != '"' && b != '\\'
	}
	return table
}()

// numberByte marks bytes that can appear in a number token. The grammar is
// checked by [parseDecimal]; this only finds where the token ends.
var numberByte = func() (table [256]bool) {
	for _, b := range []byte("0123456789+-.eE") {
		table[b] = true
	}
	return table
}()

type scanner struct {
	ctx context.Context
	r   io.Reader
	buf []byte
	// buf[start:end] holds bytes read but not yet consumed.
	start int
	end   int
	// read counts every byte read from r.
	read          int64
	inputBytesMax int64
	eof           bool
}

// newScanner returns a scanner that reads at most inputBytesMax bytes from r
// before failing with [ReasonLimit].
func newScanner(ctx context.Context, r io.Reader, inputBytesMax int64) scanner {
	return scanner{
		ctx:           ctx,
		r:             r,
		buf:           make([]byte, 512),
		start:         0,
		end:           0,
		read:          0,
		inputBytesMax: inputBytesMax,
		eof:           false,
	}
}

func (s *scanner) offset() int64 {
	return s.read - int64(s.end-s.start)
}

// fill moves unconsumed bytes to the front of the buffer and reads more. It
// returns after reading at least one byte or reaching EOF. The read is capped
// so that at most inputBytesMax+1 bytes are ever taken from r.
func (s *scanner) fill() error {
	if err := s.ctx.Err(); err != nil {
		return fault.Wrap(err, fault.Internal("validation canceled"))
	}

	if s.end == len(s.buf) && len(s.buf) < bufferBytesMax {
		grown := make([]byte, min(len(s.buf)*2, bufferBytesMax))
		s.end = copy(grown, s.buf[s.start:s.end])
		s.start = 0
		s.buf = grown
	}
	if s.start > 0 {
		s.end = copy(s.buf, s.buf[s.start:s.end])
		s.start = 0
	}
	p := s.buf[s.end:]
	if remaining := s.inputBytesMax - s.read + 1; remaining < int64(len(p)) {
		p = p[:remaining]
	}

	for range readsEmptyMax {
		n, err := s.r.Read(p)
		if n < 0 || n > len(p) {
			return fault.New("reader returned an invalid byte count")
		}
		s.end += n
		s.read += int64(n)
		if s.read > s.inputBytesMax {
			return limitError(s.inputBytesMax, "input exceeds InputBytesMax")
		}
		if err == io.EOF {
			s.eof = true
			return nil
		}
		if err != nil {
			return fault.Wrap(err, fault.Internal("failed to read input"))
		}
		if n > 0 {
			return nil
		}
	}
	return fault.Wrap(io.ErrNoProgress, fault.Internal("reader returned no data"))
}

// ensure reads until at least n bytes are buffered or the input ends, and
// returns how many are buffered. n must not exceed 12, the longest lookahead a
// token needs.
func (s *scanner) ensure(n int) (int, error) {
	for s.end-s.start < n && !s.eof {
		if err := s.fill(); err != nil {
			return 0, err
		}
	}
	return s.end - s.start, nil
}

func (s *scanner) next() (byte, error) {
	if err := s.skipSpace(); err != nil {
		return 0, err
	}
	if s.start == s.end {
		return 0, syntaxError(s.offset(), "unexpected end of input")
	}
	return s.buf[s.start], nil
}

func (s *scanner) skipSpace() error {
	for {
		for s.start < s.end {
			switch s.buf[s.start] {
			case ' ', '\t', '\n', '\r':
				s.start++
			default:
				return nil
			}
		}
		if s.eof {
			return nil
		}
		if err := s.fill(); err != nil {
			return err
		}
	}
}

func (s *scanner) finish() error {
	if err := s.skipSpace(); err != nil {
		return err
	}
	if s.start != s.end {
		return syntaxError(s.offset(), "unexpected data after the top-level value")
	}
	return nil
}

func (s *scanner) literal(text string) error {
	offset := s.offset()
	available, err := s.ensure(len(text))
	if err != nil {
		return err
	}
	if available < len(text) || string(s.buf[s.start:s.start+len(text)]) != text {
		return syntaxError(offset, "invalid literal")
	}
	s.start += len(text)
	return nil
}

func (s *scanner) stringRunes() (int64, error) {
	_, runes, err := s.scanString(nil, false, 0)
	return runes, err
}

func (s *scanner) name(dst []byte, bytesMax int) ([]byte, error) {
	dst, _, err := s.scanString(dst, true, bytesMax)
	return dst, err
}

// scanString consumes the string token whose opening quote is at the cursor,
// validating UTF-8 and escapes and counting code points. With keep it appends
// the decoded bytes to dst, bounded by bytesMax.
func (s *scanner) scanString(dst []byte, keep bool, bytesMax int) ([]byte, int64, error) {
	tokenOffset := s.offset()
	s.start++

	runes := int64(0)
	for {
		if s.start == s.end {
			if s.eof {
				return dst, runes, syntaxError(tokenOffset, "unterminated string")
			}
			if err := s.fill(); err != nil {
				return dst, runes, err
			}
			continue
		}

		plainEnd := s.start
		for plainEnd < s.end && plainStringByte[s.buf[plainEnd]] {
			plainEnd++
		}
		if plainEnd > s.start {
			if keep {
				if plainEnd-s.start > bytesMax-len(dst) {
					return dst, runes, limitError(tokenOffset, "member name exceeds KeyBytesMax")
				}
				dst = append(dst, s.buf[s.start:plainEnd]...)
			}
			runes += int64(plainEnd - s.start)
			s.start = plainEnd
			continue
		}

		c := s.buf[s.start]
		switch {
		case c == '"':
			s.start++
			return dst, runes, nil
		case c == '\\':
			r, err := s.escape()
			if err != nil {
				return dst, runes, err
			}
			if keep {
				if utf8.RuneLen(r) > bytesMax-len(dst) {
					return dst, runes, limitError(tokenOffset, "member name exceeds KeyBytesMax")
				}
				dst = utf8.AppendRune(dst, r)
			}
		case c < 0x20:
			return dst, runes, syntaxError(s.offset(), "unescaped control character in string")
		default:
			if _, err := s.ensure(utf8.UTFMax); err != nil {
				return dst, runes, err
			}
			r, size := utf8.DecodeRune(s.buf[s.start:s.end])
			if r == utf8.RuneError && size <= 1 {
				return dst, runes, syntaxError(s.offset(), "invalid UTF-8 in string")
			}
			if keep {
				if size > bytesMax-len(dst) {
					return dst, runes, limitError(tokenOffset, "member name exceeds KeyBytesMax")
				}
				dst = append(dst, s.buf[s.start:s.start+size]...)
			}
			s.start += size
		}
		runes++
	}
}

// escape consumes the escape sequence at the cursor and returns the code point
// it denotes. A \u escape for a high surrogate must be followed by one for a
// low surrogate; unpaired surrogates are rejected.
func (s *scanner) escape() (rune, error) {
	offset := s.offset()
	available, err := s.ensure(2)
	if err != nil {
		return 0, err
	}
	if available < 2 {
		return 0, syntaxError(offset, "unterminated escape sequence")
	}

	simple := rune(0)
	switch s.buf[s.start+1] {
	case '"':
		simple = '"'
	case '\\':
		simple = '\\'
	case '/':
		simple = '/'
	case 'b':
		simple = '\b'
	case 'f':
		simple = '\f'
	case 'n':
		simple = '\n'
	case 'r':
		simple = '\r'
	case 't':
		simple = '\t'
	case 'u':
		return s.unicodeEscape(offset)
	default:
		return 0, syntaxError(offset, "invalid escape sequence")
	}
	s.start += 2
	return simple, nil
}

func (s *scanner) unicodeEscape(offset int64) (rune, error) {
	available, err := s.ensure(6)
	if err != nil {
		return 0, err
	}
	if available < 6 {
		return 0, syntaxError(offset, "unterminated \\u escape")
	}
	first, ok := hex4(s.buf[s.start+2 : s.start+6])
	if !ok {
		return 0, syntaxError(offset, "invalid \\u escape")
	}
	if !utf16.IsSurrogate(first) {
		s.start += 6
		return first, nil
	}
	if first >= 0xdc00 {
		return 0, syntaxError(offset, "unpaired surrogate in \\u escape")
	}

	available, err = s.ensure(12)
	if err != nil {
		return 0, err
	}
	if available < 12 || s.buf[s.start+6] != '\\' || s.buf[s.start+7] != 'u' {
		return 0, syntaxError(offset, "unpaired surrogate in \\u escape")
	}
	second, ok := hex4(s.buf[s.start+8 : s.start+12])
	if !ok || second < 0xdc00 || second > 0xdfff {
		return 0, syntaxError(offset, "unpaired surrogate in \\u escape")
	}
	s.start += 12
	return utf16.DecodeRune(first, second), nil
}

// number consumes the number token at the cursor into dst, using scratch to
// hold the token. It fails with [ReasonLimit] before the token would exceed
// bytesMax bytes or when its exponent exceeds exponentAbsMax.
func (s *scanner) number(scratch []byte, bytesMax int, dst *decimal) ([]byte, error) {
	tokenOffset := s.offset()
	scratch = scratch[:0]
	for {
		if s.start == s.end {
			if s.eof {
				break
			}
			if err := s.fill(); err != nil {
				return scratch, err
			}
			continue
		}
		c := s.buf[s.start]
		if !numberByte[c] {
			break
		}
		if len(scratch) >= bytesMax {
			return scratch, limitError(tokenOffset, "number exceeds NumberBytesMax")
		}
		scratch = append(scratch, c)
		s.start++
	}

	switch parseDecimal(scratch, dst) {
	case decimalOK:
		return scratch, nil
	case decimalSyntax:
		return scratch, syntaxError(tokenOffset, "invalid number")
	case decimalExponentLimit:
		return scratch, limitError(tokenOffset, "number exponent exceeds 1e9 in magnitude")
	default:
		return scratch, fault.New("unknown decimal status")
	}
}

func hex4(b []byte) (rune, bool) {
	value := rune(0)
	for _, c := range b {
		var digit byte
		switch {
		case c >= '0' && c <= '9':
			digit = c - '0'
		case c >= 'a' && c <= 'f':
			digit = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			digit = c - 'A' + 10
		default:
			return 0, false
		}
		value = value<<4 | rune(digit)
	}
	return value, true
}

func syntaxError(offset int64, message string) error {
	return &ValidationError{
		Reason:           ReasonSyntax,
		InstanceLocation: "",
		KeywordLocation:  "",
		Offset:           offset,
		Message:          message,
	}
}

func limitError(offset int64, message string) error {
	return &ValidationError{
		Reason:           ReasonLimit,
		InstanceLocation: "",
		KeywordLocation:  "",
		Offset:           offset,
		Message:          message,
	}
}
