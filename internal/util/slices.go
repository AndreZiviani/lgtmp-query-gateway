package util

import "slices"

func SlicesContains[S ~[]E, E comparable](a, b S) bool {
	for _, elem := range a {
		if slices.Contains(b, elem) {
			return true
		}
	}

	return false
}
