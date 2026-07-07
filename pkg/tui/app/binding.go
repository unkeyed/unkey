package app

// Binding maps trigger keys to a hint label, so a component declares a key and
// its help text once and uses it for both dispatch and the hint bar.
//
// Keys are matched against KeyMsg.String(). An empty Keys makes it a hint-only
// entry (rendered, never matched) for keys handled elsewhere (e.g. arrow-key
// navigation owned by a list). Display overrides the key text shown in the hint
// bar (e.g. "1-5" for a range, "↑/↓" for navigation); it defaults to Keys[0].
type Binding struct {
	Keys    []string
	Display string
	Help    string
}

// Matches reports whether k triggers this binding.
func (b Binding) Matches(k KeyMsg) bool {
	s := k.String()
	for _, key := range b.Keys {
		if key == s {
			return true
		}
	}
	return false
}

// DisplayKey is the key text to show in a hint bar.
func (b Binding) DisplayKey() string {
	if b.Display != "" {
		return b.Display
	}
	if len(b.Keys) > 0 {
		return b.Keys[0]
	}
	return ""
}
