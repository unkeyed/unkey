package util

import (
	"cmp"
)

func Max[T cmp.Ordered](s []T) T {
	if len(s) == 0 {
		var t T
		return t
	}
	max := s[0]
	for _, v := range s[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func Min[T cmp.Ordered](s []T) T {
	if len(s) == 0 {
		var t T
		return t
	}
	min := s[0]
	for _, v := range s[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
