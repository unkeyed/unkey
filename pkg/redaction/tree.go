package redaction

import (
	"bytes"
	"encoding/json"
	"strings"
)

// node is one step in the path tree: children for object members, elem for array
// elements, redact marking a path that ends here.
type node struct {
	children map[string]*node
	elem     *node
	redact   bool
}

// redactAll stands in for a match found while the scan is desynchronized, where
// only the member name is trustworthy.
var redactAll = &node{children: nil, elem: nil, redact: true}

// add inserts one path into the tree.
func (r *Redactor) add(path string) {
	cur := r.root
	name := ""

	for _, segment := range strings.Split(path, ".") {
		label, arrays := splitSegment(segment)

		// A segment can be array steps alone, as in the leading "[]" of
		// "[].key", where the body itself is an array.
		if label != "" {
			if cur.children == nil {
				cur.children = make(map[string]*node)
			}
			next, ok := cur.children[label]
			if !ok {
				next = &node{children: nil, elem: nil, redact: false}
				cur.children[label] = next
			}
			cur, name = next, label
		}

		for range arrays {
			if cur.elem == nil {
				cur.elem = &node{children: nil, elem: nil, redact: false}
			}
			cur = cur.elem
		}
	}

	if cur == r.root {
		return
	}
	cur.redact = true
	r.paths = append(r.paths, path)
	r.names[name] = struct{}{}
}

// splitSegment separates a path segment's member name from its trailing array
// markers: "variables[][]" is the name "variables" followed by two steps into
// elements.
func splitSegment(segment string) (string, int) {
	arrays := 0
	for strings.HasSuffix(segment, "[]") {
		segment = segment[:len(segment)-2]
		arrays++
	}

	return segment, arrays
}

// resolve looks up the member name spanning [nameStart, nameEnd) against the
// current position.
//
// A name may be escaped: `{"key":"..."}` is the member `key` to every JSON
// parser, including the validator that accepts the request and the handler that
// reads it, so comparing raw bytes would let a caller rename a field past the
// path tree and have its own credential logged. Escapes are vanishingly rare in
// practice, so they take a slower path that decodes first, and a name that will
// not decode at all is treated as sensitive rather than guessed at.
func (r *Redactor) resolve(frames []frame, depth, untracked int, desynced bool, in []byte, nameStart, nameEnd int) *node {
	raw := in[nameStart+1 : nameEnd-1]
	if bytes.IndexByte(raw, '\\') < 0 {
		return r.lookup(frames, depth, untracked, desynced, raw)
	}

	decoded, ok := decodeName(in[nameStart:nameEnd])
	if !ok {
		return redactAll
	}

	return r.lookupString(frames, depth, untracked, desynced, decoded)
}

// decodeName unescapes a complete JSON string token, quotes included.
func decodeName(token []byte) (string, bool) {
	var name string
	if err := json.Unmarshal(token, &name); err != nil {
		return "", false
	}

	return name, true
}

// lookup resolves an unescaped member name. While the scan is synchronized this
// is an exact path match; once desynchronized it degrades to matching the name
// alone. Taking []byte keeps the map lookup allocation-free.
func (r *Redactor) lookup(frames []frame, depth, untracked int, desynced bool, name []byte) *node {
	if desynced {
		if _, sensitive := r.names[string(name)]; sensitive {
			return redactAll
		}

		return nil
	}

	children := r.childrenAt(frames, depth, untracked)
	if children == nil {
		return nil
	}

	return children[string(name)]
}

// lookupString is [Redactor.lookup] for a name that had to be decoded.
func (r *Redactor) lookupString(frames []frame, depth, untracked int, desynced bool, name string) *node {
	if desynced {
		if _, sensitive := r.names[name]; sensitive {
			return redactAll
		}

		return nil
	}

	children := r.childrenAt(frames, depth, untracked)
	if children == nil {
		return nil
	}

	return children[name]
}

// childrenAt returns the member names reachable at the current position, or nil
// when nothing below here can match.
func (r *Redactor) childrenAt(frames []frame, depth, untracked int) map[string]*node {
	if depth == 0 || untracked > 0 {
		return nil
	}

	current := frames[depth-1]
	if current.isArray || current.node == nil {
		return nil
	}

	return current.node.children
}
