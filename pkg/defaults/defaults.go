// Package defaults provides tiny helpers for filling in zero-value config
// fields with sensible fallbacks. The shape:
//
//	cfg.MaxSize = defaults.Or(cfg.MaxSize, 10_000)
//	cfg.Clock   = defaults.OrFunc(cfg.Clock, clock.New)
//
// instead of the same idea spelled as four lines per field with a clear
// regression target (typo "==" → "!=" silently inverts every default).
//
// We could just use stdlib cmp.Or for the comparable cases, but
// cmp.Or always evaluates both arguments, which is wasteful when the
// fallback allocates (e.g. a fresh clock). OrFunc handles that.
package defaults

// Or returns first when it differs from the zero value of T, otherwise
// returns second. T must be comparable so the zero check is well-defined
// for ints, durations, strings, and nilable interfaces alike.
func Or[T comparable](first, second T) T {
	var zero T
	if first != zero {
		return first
	}

	return second
}

// OrFunc is Or with a lazy fallback. Use it when the fallback allocates
// or has side-effects you only want to pay when the field is actually
// missing.
func OrFunc[T comparable](first T, fallback func() T) T {
	var zero T
	if first != zero {
		return first
	}

	return fallback()
}
