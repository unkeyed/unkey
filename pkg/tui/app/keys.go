package app

// Directional/action predicates on a key event, accepting the common aliases
// (arrows, vim h/j/k/l, ctrl+p/n) alongside the named key types.

func (k KeyMsg) IsUp() bool {
	switch k.String() {
	case "up", "k", "ctrl+p":
		return true
	default:
		return k.Type == KeyUp
	}
}

func (k KeyMsg) IsDown() bool {
	switch k.String() {
	case "down", "j", "ctrl+n":
		return true
	default:
		return k.Type == KeyDown
	}
}

func (k KeyMsg) IsEnter() bool {
	return k.String() == "enter" || k.Type == KeyEnter
}

func (k KeyMsg) IsEsc() bool {
	return k.String() == "esc" || k.Type == KeyEsc
}

func (k KeyMsg) IsLeft() bool {
	switch k.String() {
	case "left", "h":
		return true
	default:
		return k.Type == KeyLeft
	}
}

func (k KeyMsg) IsRight() bool {
	switch k.String() {
	case "right", "l":
		return true
	default:
		return k.Type == KeyRight
	}
}

// MoveCursor clamps cur+delta to [0, max-1]; returns 0 when the list is empty.
func MoveCursor(cur, delta, max int) int {
	if max <= 0 {
		return 0
	}
	cur += delta
	if cur < 0 {
		return 0
	}
	if cur >= max {
		return max - 1
	}
	return cur
}
