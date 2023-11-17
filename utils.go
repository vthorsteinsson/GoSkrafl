// utils.go
// Copyright (C) 2023 Vilhjálmur Þorsteinsson / Miðeind ehf.

// This file contains general utility functions.

package skrafl

// Remove a given rune from a slice of runes, returning a new slice.
func RemoveRune(s []rune, r rune) []rune {
	// Preallocate the result slice with the same length as the input slice.
	// This is the maximum possible size it could be if no runes are removed.
	result := make([]rune, 0, len(s))
	for _, runeValue := range s {
		if runeValue != r {
			result = append(result, runeValue)
		}
	}
	return result
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
