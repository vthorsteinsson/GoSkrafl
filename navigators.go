// navigators.go
// Copyright (C) 2023 Vilhjálmur Þorsteinsson / Miðeind ehf.

// This file contains the Navigator interface and declares
// a couple of classes that implement it to provide various
// types of navigation over a DAWG. For instance, there are
// navigators to find words in the DAWG, to find permutations
// of letters, to find words that match patterns with wildcards,
// and to find all left permutations (prefixes) of a rack.

/*

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/

package skrafl

import (
	"fmt"
	"strings"
)

// Navigator is an interface that describes behaviors that control the
// navigation of a Dawg
type Navigator interface {
	IsAccepting() bool
	Accepts(rune) bool
	Accept(matched []rune, final bool, state *navState)
	PushEdge(rune) bool
	PopEdge() bool
	Done()
}

// Navigation contains the state of a single navigation that is
// underway within a Dawg
type Navigation struct {
	dawg      *Dawg
	navigator Navigator
	// isResumable is set to true if we should call navigator.Accept()
	// with the full state of the navigation in the last parameter.
	// If the navigation doesn't require this, leave isResumable set
	// to false for best performance.
	isResumable bool
}

// FromNode continues a navigation from a node in the Dawg,
// enumerating through outgoing edges until the navigator is
// satisfied
func (nav *Navigation) FromNode(offset uint32, matched []rune) {
	iter := nav.dawg.iterNode(offset)
	for i := 0; i < len(*iter); i++ {
		state := &((*iter)[i])
		if nav.navigator.PushEdge(state.prefix[0]) {
			// The navigator wants us to enter this edge
			nav.FromEdge(state, matched)
			if !nav.navigator.PopEdge() {
				// The navigator doesn't want to visit
				// other edges, so we're done with this node
				break
			}
		}
	}
}

// FromEdge navigates along an edge in the Dawg. An edge
// consists of a prefix string, which may be longer than
// one letter.
func (nav *Navigation) FromEdge(state *navState, alreadyMatched []rune) {
	lenP := len(state.prefix)
	j := 0
	navigator := nav.navigator
	// Copy the alreadyMatched rune slice into a new rune slice
	var matched []rune
	numMatched := len(alreadyMatched)
	if numMatched > 0 {
		matched = make([]rune, numMatched, numMatched+lenP)
		copy(matched, alreadyMatched)
	}
	for j < lenP && navigator.IsAccepting() {
		if !navigator.Accepts(state.prefix[j]) {
			// The navigator doesn't want this prefix letter:
			// we're done
			return
		}
		// The navigator wants this prefix letter:
		// add it to the matched prefix and find out whether
		// it is now in a final state (i.e. an entire valid word)
		matched = append(matched, state.prefix[j])
		j++
		// Have we just completed an entire word?
		final := false
		if j < lenP {
			// The edge prefix contains a vertical bar ('|') after
			// the accepted letter: we're at a complete word boundary
			if state.prefix[j] == '|' {
				final = true
				j++
			}
		} else {
			// The prefix is complete: if there is no next node, or if
			// the next node is marked with a final bit, we're at a
			// complete word boundary
			if state.nextNode == 0 || nav.dawg.b[state.nextNode]&0x80 != 0 {
				final = true
			}
		}
		// Notify the navigator of the match
		if nav.isResumable {
			// We want the full navigation state to be passed to navigator.Accept()
			navigator.Accept(
				matched,
				final,
				// Create a navState that would resume the navigation at our
				// current location within the prefix, with the same nextNode
				&navState{prefix: state.prefix[j:], nextNode: state.nextNode},
			)
		} else {
			// No need to pass the full state
			navigator.Accept(matched, final, nil)
		}
	}
	if j >= lenP && state.nextNode != 0 && navigator.IsAccepting() {
		// Completed a whole prefix and still the navigator
		// has appetite: continue to the following node
		nav.FromNode(state.nextNode, matched)
	}
}

// Go starts a navigation on the underlying Dawg using the given
// Navigator
func (nav *Navigation) Go(dawg *Dawg, navigator Navigator) {
	if nav == nil || dawg == nil || navigator == nil {
		return
	}
	nav.dawg = dawg
	nav.navigator = navigator
	if navigator.IsAccepting() {
		// Leave our home harbor and set sail for the open seas
		nav.FromNode(0, []rune{})
	}
	navigator.Done()
}

// Resume continues a navigation on the underlying Dawg
// using the given Navigator, from a previously saved navigation
// state
func (nav *Navigation) Resume(dawg *Dawg, navigator Navigator, state *navState, matched []rune) {
	if nav == nil || dawg == nil || navigator == nil || state == nil {
		return
	}
	nav.dawg = dawg
	nav.navigator = navigator
	if navigator.IsAccepting() {
		// Leave from our previously dropped buoy
		nav.FromEdge(state, matched)
	}
	navigator.Done()
}

// FindNavigator stores the state for a plain word search in the Dawg,
// and implements the Navigator interface
type FindNavigator struct {
	word    []rune
	lenWord int
	index   int
	found   bool
}

// Init initializes a FindNavigator with the word to search for
func (fn *FindNavigator) Init(word string) {
	// Convert the word to a list of runes
	fn.word = []rune(word)
	fn.lenWord = len(fn.word) // Be careful! Not len(word)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (fn *FindNavigator) PushEdge(chr rune) bool {
	// If the edge matches our place in the sought word, go for it
	return fn.word[fn.index] == chr
}

// PopEdge returns false if there is no need to visit other edges
// after this one has been traversed
func (fn *FindNavigator) PopEdge() bool {
	// There can only be one correct outgoing edge for the
	// Find function, so we return false to prevent other edges
	// from being tried
	return false
}

// Done is called when the navigation is complete
func (fn *FindNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (fn *FindNavigator) IsAccepting() bool {
	return fn.index < fn.lenWord
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (fn *FindNavigator) Accepts(chr rune) bool {
	// For the FindNavigator, we never enter an edge unless
	// we have the correct character, so we simply advance
	// the index and return true
	fn.index++
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (fn *FindNavigator) Accept(matched []rune, final bool, state *navState) {
	if final && fn.index == fn.lenWord {
		// This is a whole word (final=true) and matches our
		// length, so that's it
		fn.found = true
	}
}

// PermutationNavigator stores the state for a plain word search in the Dawg,
// and implements the Navigator interface
type PermutationNavigator struct {
	rack    string
	stack   []string
	results []string
	minLen  int
}

// Init initializes a PermutationNavigator with the word to search for
func (pn *PermutationNavigator) Init(rack string, minLen int) {
	pn.rack = rack
	pn.minLen = minLen
	pn.stack = make([]string, 0, RackSize)
	pn.results = make([]string, 0)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (pn *PermutationNavigator) PushEdge(chr rune) bool {
	if strings.ContainsRune(pn.rack, chr) || strings.ContainsRune(pn.rack, '?') {
		pn.stack = append(pn.stack, pn.rack)
		return true
	}
	return false
}

// PopEdge returns false if there is no need to visit other edges
// after this one has been traversed
func (pn *PermutationNavigator) PopEdge() bool {
	last := len(pn.stack) - 1
	pn.rack = pn.stack[last]
	pn.stack = pn.stack[0:last]
	return true
}

// Done is called when the navigation is complete
func (pn *PermutationNavigator) Done() {
	// The results come out sorted alphabetically.
	// It would be possible to order them by length;
	// the sorting would then appear here.
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (pn *PermutationNavigator) IsAccepting() bool {
	return len(pn.rack) > 0
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (pn *PermutationNavigator) Accepts(chr rune) bool {
	exactMatch := strings.ContainsRune(pn.rack, chr)
	if !exactMatch && !strings.ContainsRune(pn.rack, '?') {
		// The next letter is not in the rack, and the rack
		// does not contain a wildcard/blank: return false
		return false
	}
	if exactMatch {
		// This is a regular letter match
		pn.rack = strings.Replace(pn.rack, string(chr), "", 1)
	} else {
		// This is a wildcard match
		pn.rack = strings.Replace(pn.rack, "?", "", 1)
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (pn *PermutationNavigator) Accept(matched []rune, final bool, state *navState) {
	if final && len(matched) >= pn.minLen {
		// This is a full word (final=true) and the number of letters
		// is above the minimum limit: add it to the results
		pn.results = append(pn.results, string(matched))
	}
}

// MatchNavigator stores the state for a pattern matching
// navigation of a Dawg, and implements the Navigator interface
type MatchNavigator struct {
	pattern    []rune
	lenP       int
	index      int
	chMatch    rune
	isWildcard bool
	stack      []matchItem
	results    []string
}

type matchItem struct {
	index      int
	chMatch    rune
	isWildcard bool
}

// Init initializes a MatchNavigator with the word to search for
func (mn *MatchNavigator) Init(pattern []rune) {
	// Convert the word to a list of runes
	mn.pattern = pattern
	mn.lenP = len(mn.pattern)
	mn.chMatch = mn.pattern[0]
	mn.isWildcard = mn.chMatch == '?'
	mn.stack = make([]matchItem, 0, RackSize)
	// The initial capacity of the results list, 16, is just
	// a guesstimate / magic number
	mn.results = make([]string, 0, 16)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (mn *MatchNavigator) PushEdge(chr rune) bool {
	if chr != mn.chMatch && !mn.isWildcard {
		return false
	}
	mn.stack = append(mn.stack, matchItem{mn.index, mn.chMatch, mn.isWildcard})
	return true
}

// PopEdge returns false if there is no need to visit other edges
// after this one has been traversed
func (mn *MatchNavigator) PopEdge() bool {
	last := len(mn.stack) - 1
	mt := &mn.stack[last]
	mn.index, mn.chMatch, mn.isWildcard = mt.index, mt.chMatch, mt.isWildcard
	mn.stack = mn.stack[0:last]
	return mn.isWildcard
}

// Done is called when the navigation is complete
func (mn *MatchNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (mn *MatchNavigator) IsAccepting() bool {
	return mn.index < mn.lenP
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (mn *MatchNavigator) Accepts(chr rune) bool {
	if chr != mn.chMatch && !mn.isWildcard {
		// Not a correct next character in the word
		return false
	}
	// This is a correct character: advance our index
	mn.index++
	if mn.index < mn.lenP {
		mn.chMatch = mn.pattern[mn.index]
		mn.isWildcard = mn.chMatch == '?'
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (mn *MatchNavigator) Accept(matched []rune, final bool, state *navState) {
	if final && mn.index == mn.lenP {
		// Entire pattern match
		mn.results = append(mn.results, string(matched))
	}
}

// LeftFindNavigator is similar to FindNavigator, but instead of returning
// only a bool result, it returns the full navigation state as it is when
// the requested word prefix is found. This makes it possible to continue the
// navigation later with further constraints.
type LeftFindNavigator struct {
	prefix Prefix
	lenP   int
	index  int
	// Below is the result of the LeftFindNavigator,
	// which is used to continue navigation after a left part
	// has been found on the board
	state *navState
}

// Init initializes a LeftFindNavigator with the word to search for
func (lfn *LeftFindNavigator) Init(prefix Prefix) {
	lfn.prefix = prefix
	lfn.lenP = len(prefix)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (lfn *LeftFindNavigator) PushEdge(chr rune) bool {
	// If the edge matches our place in the sought word, go for it
	return lfn.prefix[lfn.index] == chr
}

// PopEdge returns false if there is no need to visit other edges
// after this one has been traversed
func (lfn *LeftFindNavigator) PopEdge() bool {
	// There can only be one correct outgoing edge for the
	// Find function, so we return false to prevent other edges
	// from being tried
	return false
}

// Done is called when the navigation is complete
func (lfn *LeftFindNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (lfn *LeftFindNavigator) IsAccepting() bool {
	return lfn.index < lfn.lenP
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (lfn *LeftFindNavigator) Accepts(chr rune) bool {
	// For the LeftFindNavigator, we never enter an edge unless
	// we have the correct character, so we simply advance
	// the index and return true
	lfn.index++
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (lfn *LeftFindNavigator) Accept(matched []rune, final bool, state *navState) {
	if lfn.index == lfn.lenP {
		// Found the whole left part; save its position (state)
		lfn.state = state
	}
}

// LeftPermutationNavigator finds all left parts of words that are
// possible with a particular rack, and accumulates them by length.
// This is done once at the start of move generation.
type LeftPermutationNavigator struct {
	rack      []rune
	hasBlank  bool
	stack     []leftPermItem
	maxLeft   int
	leftParts [][]*LeftPart
	index     int
}

type leftPermItem struct {
	rack  []rune
	index int
}

// LeftPart stores the navigation state after matching a particular
// left part within the DAWG, so we can resume navigation from that
// point to complete an anchor square followed by a right part
type LeftPart struct {
	matched []rune
	rack    []rune
	state   *navState
}

// String returns a string representation of a LeftPart, for debugging purposes
func (lp *LeftPart) String() string {
	var sb strings.Builder
	sb.WriteString(
		fmt.Sprintf(
			"LeftPart: matched '%v' from rack '%v",
			string(lp.matched),
			string(lp.rack),
		),
	)
	return sb.String()
}

// Init initializes a fresh LeftPermutationNavigator using the given rack
func (lpn *LeftPermutationNavigator) Init(rack []rune) {
	// Copy rack into lpn.rack
	lenRack := len(rack)
	lpn.rack = make([]rune, lenRack)
	copy(lpn.rack, rack)
	// One tile from the rack will be put on the anchor square;
	// the rest is available to be played to the left of the anchor.
	// We thus find all permutations involving all rack tiles except
	// one.
	if lenRack <= 1 {
		// No left permutation possible
		lpn.maxLeft = 0
	} else {
		lpn.maxLeft = lenRack - 1
	}
	lpn.hasBlank = ContainsRune(lpn.rack, '?')
	lpn.stack = make([]leftPermItem, 0, 8)
	lpn.leftParts = make([][]*LeftPart, lpn.maxLeft)
	for i := 0; i < lpn.maxLeft; i++ {
		lpn.leftParts[i] = make([]*LeftPart, 0, 8)
	}
}

// LeftParts returns a list of strings containing the left parts of words
// that could be found in the given Rack
func (lpn *LeftPermutationNavigator) LeftParts(length int) []*LeftPart {
	if length < 1 || length > lpn.maxLeft {
		// Return a nil slice for unsupported lengths
		return nil
	}
	return lpn.leftParts[length-1]
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (lpn *LeftPermutationNavigator) PushEdge(chr rune) bool {
	if !lpn.hasBlank && !ContainsRune(lpn.rack, chr) {
		return false
	}
	lpn.stack = append(lpn.stack, leftPermItem{lpn.rack, lpn.index})
	return true
}

// PopEdge returns false if there is no need to visit other edges
// after this one has been traversed
func (lpn *LeftPermutationNavigator) PopEdge() bool {
	// Pop the previous rack and index from the stack
	last := len(lpn.stack) - 1
	lpn.rack, lpn.index = lpn.stack[last].rack, lpn.stack[last].index
	lpn.stack = lpn.stack[0:last]
	return true
}

// Done is called when the navigation is complete
func (lpn *LeftPermutationNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (lpn *LeftPermutationNavigator) IsAccepting() bool {
	return lpn.index < lpn.maxLeft
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (lpn *LeftPermutationNavigator) Accepts(chr rune) bool {
	exactMatch := ContainsRune(lpn.rack, chr)
	if !exactMatch && !lpn.hasBlank {
		return false
	}
	lpn.index++
	if exactMatch {
		lpn.rack = RemoveRune(lpn.rack, chr)
	} else {
		// Matched a blank
		lpn.rack = RemoveRune(lpn.rack, '?')
		// There might be a second blank in the rack,
		// so we need to check again
		lpn.hasBlank = ContainsRune(lpn.rack, '?')
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (lpn *LeftPermutationNavigator) Accept(matched []rune, final bool, state *navState) {
	ix := len(matched) - 1
	lpn.leftParts[ix] = append(
		lpn.leftParts[ix],
		&LeftPart{
			matched: matched,
			rack:    lpn.rack,
			state:   state,
		},
	)
}

// FindLeftParts returns all left part permutations that can be generated
// from the given rack, grouped by length
func FindLeftParts(dawg *Dawg, rack []rune) [][]*LeftPart {
	var lpn LeftPermutationNavigator
	lpn.Init(rack)
	dawg.NavigateResumable(&lpn)
	return lpn.leftParts
}
