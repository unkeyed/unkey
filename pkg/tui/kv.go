package tui

import "strings"

// KV accumulates key-value pairs and prints them with values aligned to the
// widest key. Keys render dim; values carry whatever styling the caller
// applied.
type KV struct {
	r      *Renderer
	pairs  [][2]string
	indent int
}

// KV starts a key-value block.
func (r *Renderer) KV() *KV {
	return &KV{r: r, pairs: nil, indent: 0}
}

// Indent shifts the whole block right by n spaces.
func (kv *KV) Indent(n int) *KV {
	kv.indent = n
	return kv
}

// Add appends a pair.
func (kv *KV) Add(key, value string) *KV {
	kv.pairs = append(kv.pairs, [2]string{key, value})
	return kv
}

// AddIf appends a pair only when the value is non-empty, for optional fields.
func (kv *KV) AddIf(key, value string) *KV {
	if value == "" {
		return kv
	}
	return kv.Add(key, value)
}

// Print writes the block in a single write. An empty block prints nothing.
func (kv *KV) Print() {
	if len(kv.pairs) == 0 {
		return
	}

	width := 0
	for _, pair := range kv.pairs {
		if w := visibleWidth(pair[0]); w > width {
			width = w
		}
	}

	var b strings.Builder
	for _, pair := range kv.pairs {
		writeSpaces(&b, kv.indent)
		// Pad inside the Dim so the escape codes wrap the whole column once;
		// trailing spaces render identically dim or plain.
		var key strings.Builder
		writePadded(&key, pair[0], width)
		b.WriteString(kv.r.Dim(key.String()))
		b.WriteString("  ")
		b.WriteString(pair[1])
		b.WriteByte('\n')
	}
	kv.r.write(b.String())
}
