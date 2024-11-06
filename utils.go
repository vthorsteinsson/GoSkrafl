// utils.go
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.

// This file contains general utility functions.

package skrafl

// Remove a single instance of a given rune from a slice of runes,
// returning a new slice.
func RemoveRune(s []rune, r rune) []rune {
	// Preallocate the result slice with the same length as the input slice.
	// This is the maximum possible size it could be if no runes are removed.
	result := make([]rune, 0, len(s))
	for ix, runeValue := range s {
		if runeValue == r {
			// Found the sought-after rune; append the
			// rest and return the result
			return append(result, s[ix+1:]...)
		}
		result = append(result, runeValue)
	}
	// The sought-after rune was not found; return the original slice
	return s
}

// Return true if a slice of runes contains a given rune.
func ContainsRune(s []rune, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
