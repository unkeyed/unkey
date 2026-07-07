package app

// KeyType classifies a key event. Printable keys carry KeyRunes with the rune
// in Runes; everything else is a named type.
type KeyType int

const (
	KeyRunes KeyType = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
	KeyTab
	KeyShiftTab
	KeyCtrlC
	KeyCtrlP
	KeyCtrlN
	KeyBackspace
)

// KeyMsg is a single key event. Callers read Type and Runes; Alt/Paste are
// carried for completeness.
type KeyMsg struct {
	Type  KeyType
	Runes []rune
	Alt   bool
	Paste bool
}

// String renders the key the way callers match on it: printable keys as their
// literal characters, named keys as words ("up", "enter", "ctrl+c", ...).
func (k KeyMsg) String() string {
	switch k.Type {
	case KeyRunes:
		return string(k.Runes)
	case KeyUp:
		return "up"
	case KeyDown:
		return "down"
	case KeyLeft:
		return "left"
	case KeyRight:
		return "right"
	case KeyEnter:
		return "enter"
	case KeyEsc:
		return "esc"
	case KeyTab:
		return "tab"
	case KeyShiftTab:
		return "shift+tab"
	case KeyCtrlC:
		return "ctrl+c"
	case KeyCtrlP:
		return "ctrl+p"
	case KeyCtrlN:
		return "ctrl+n"
	case KeyBackspace:
		return "backspace"
	default:
		return ""
	}
}
