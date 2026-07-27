package redaction

import (
	"bytes"
	"sort"
	"unsafe"
)

// Placeholder is written in place of every redacted value, including the
// surrounding quotes, so the result stays valid JSON regardless of the original
// value's type.
const Placeholder = `"[REDACTED]"`

// maxDepth bounds the container stack. JSON nested deeper than this cannot match
// any path a spec declares, so deeper levels are tracked only well enough to
// unwind correctly.
const maxDepth = 64

// Redactor rewrites JSON bodies, replacing the values of members whose path is
// configured. A Redactor is immutable after construction and safe for concurrent
// use.
type Redactor struct {
	root  *node
	paths []string

	// names holds the final member name of every path, used as a fallback when
	// the structure is too malformed to track position. Over-redacting is the
	// right way to be wrong there.
	names map[string]struct{}
}

// New builds a Redactor covering the given paths.
//
// A path is member names joined by dots, with `[]` marking a step into array
// elements: "key", "data.variables[].value", "a[][]". Paths are anchored at the
// root of the body, so "value" never matches a nested value.
//
// A Redactor with no paths, and a nil *Redactor, both pass bodies through
// untouched.
func New(paths []string) *Redactor {
	r := &Redactor{
		root:  &node{children: nil, elem: nil, redact: false},
		paths: make([]string, 0, len(paths)),
		names: make(map[string]struct{}, len(paths)),
	}
	for _, p := range paths {
		r.add(p)
	}
	sort.Strings(r.paths)

	return r
}

// Paths returns the configured paths, sorted. Intended for logging the active
// configuration at startup.
func (r *Redactor) Paths() []string {
	if r == nil {
		return nil
	}

	return append([]string(nil), r.paths...)
}

// frame is one open container on the scanner's stack. node is the tree position
// whose children apply to this object's members, or whose elem applies to this
// array's elements; nil means nothing below here can match.
type frame struct {
	node    *node
	isArray bool
}

// Redact returns in with the value of every configured path replaced by
// [Placeholder].
//
// The input is never mutated. When nothing matches, in is returned as-is, so the
// common case allocates nothing; a match allocates one buffer.
//
// Truncated input fails closed: an unterminated string, object, or array in a
// redacted position consumes the rest of the body, so a secret cut off mid-write
// cannot survive in the tail.
//
// Structurally broken input gives up on position and falls back to matching the
// final member name of each path, which over-redacts rather than leaking. That
// covers stray quotes shifting every following token, colons chained after a
// value, and containers closing that never opened.
//
// What it covers is well-formed JSON and truncated prefixes of it, which is what
// a client can send and all a handler can emit. One gap remains, unchanged from
// matching on names: a JSON document encoded inside a string value is opaque,
// since to the parser it is a single string token.
func (r *Redactor) Redact(in []byte) []byte {
	if r == nil || r.root == nil || len(in) == 0 {
		return in
	}

	var out []byte
	flushed := 0 // how much of in has been written to out

	var frames [maxDepth]frame
	depth := 0     // containers being tracked
	untracked := 0 // containers open beyond maxDepth
	desynced := false

	for i := 0; i < len(in); {
		switch in[i] {
		case '"':
			nameStart := i
			nameEnd, ok := stringEnd(in, nameStart)
			if !ok {
				// Unterminated string: everything after it belongs to that
				// string, so no further member can be identified.
				i = len(in)

				continue
			}

			after := skipSpace(in, nameEnd)
			if after >= len(in) {
				i = len(in)

				continue
			}

			if in[after] != ':' {
				if isStructural(in[after]) {
					// A value or an array element.
					i = nameEnd

					continue
				}
				// The scan is misaligned: an odd number of quotes earlier made
				// this token start inside what is really a name. Resynchronize
				// one byte in, and stop trusting position.
				desynced = true
				i = nameStart + 1

				continue
			}

			// A member outside any container means the opening brace was never
			// seen, so position cannot be trusted from here on.
			if depth == 0 && untracked == 0 {
				desynced = true
			}

			target := r.resolve(frames[:], depth, untracked, desynced, in, nameStart, nameEnd)
			valueStart := skipSpace(in, after+1)

			if target != nil && target.redact {
				end := sensitiveValueEnd(in, valueStart)

				if out == nil {
					out = make([]byte, 0, len(in)+len(Placeholder))
				}
				out = append(out, in[flushed:valueStart]...)
				out = append(out, Placeholder...)
				flushed = end
				i = end

				continue
			}

			// Descend into a container value here, while the member name that
			// governs it is still known.
			if valueStart < len(in) && (in[valueStart] == '{' || in[valueStart] == '[') {
				if depth < maxDepth {
					frames[depth] = frame{node: target, isArray: in[valueStart] == '['}
					depth++
				} else {
					untracked++
				}
				i = valueStart + 1

				continue
			}

			i = after + 1

		case '{', '[':
			// A container in value position: either the root body or an element
			// of the array we are inside. Anywhere else, the structure is not
			// what it claims to be.
			var n *node
			switch {
			case depth == 0 && untracked == 0:
				n = r.root
			case depth > 0 && frames[depth-1].isArray:
				if parent := frames[depth-1].node; parent != nil {
					n = parent.elem
				}
			case untracked > 0:
				// Past maxDepth, position is unavailable rather than wrong, so
				// nothing below can match but the scan stays trustworthy.
			default:
				desynced = true
			}

			if depth < maxDepth {
				frames[depth] = frame{node: n, isArray: in[i] == '['}
				depth++
			} else {
				untracked++
			}
			i++

		case '}', ']':
			switch {
			case untracked > 0:
				untracked--
			case depth > 0:
				depth--
			default:
				// Closing a container that never opened.
				desynced = true
			}
			i++

		default:
			if desynced {
				// Position is already lost, so containers no longer mean
				// anything and only string tokens can still match. Jump to the
				// next one instead of walking the bytes between, which keeps a
				// megabyte of garbage from costing a megabyte of crawling.
				rel := bytes.IndexByte(in[i:], '"')
				if rel < 0 {
					i = len(in)
				} else {
					i += rel
				}

				continue
			}
			i++
		}
	}

	if out == nil {
		return in
	}

	return append(out, in[flushed:]...)
}

// RedactString is [Redact] for callers that store the result as a string, such
// as a log row. Converting the result with string(...) would copy the whole body
// a second time, on top of the buffer Redact already allocated; this hands back
// the bytes directly, so a body with nothing to redact costs no allocation at
// all and a body with something to redact costs exactly one.
//
// The returned string aliases its argument whenever nothing was redacted, so the
// caller must treat in as immutable for as long as the string is reachable. For
// request logs that means until the batch holding the row has been flushed.
// Never pass a buffer that gets written to again, including anything pooled and
// refilled in place.
func (r *Redactor) RedactString(in []byte) string {
	out := r.Redact(in)
	if len(out) == 0 {
		return ""
	}

	return unsafe.String(unsafe.SliceData(out), len(out))
}

// sensitiveValueEnd returns the index just past everything a sensitive member
// governs, starting at the value itself.
//
// That is usually one value, but a colon after it means what was just consumed is
// itself a name, as in `"key":"a":"secret"`, and everything down such a chain is
// governed by the same sensitive path. Valid JSON never puts a colon after a
// value, so this only extends on malformed input, and each step moves forward.
func sensitiveValueEnd(in []byte, valueStart int) int {
	end := valueEnd(in, valueStart)

	for {
		next := skipSpace(in, end)
		if next >= len(in) || in[next] != ':' {
			return end
		}
		end = valueEnd(in, skipSpace(in, next+1))
	}
}
