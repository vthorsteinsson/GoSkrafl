// player.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file implements a Scrabble(tm) playing robot.

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

// LeftFindNavigator is similar to FindNavigator, but instead of returning
// only a bool result, it returns the full navigation state as it is when
// the requested word prefix is found. This makes it possible to continue the
// navigation with further constraints.
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
func (lfn *LeftFindNavigator) Accept(matched string, final bool, state *navState) {
	if lfn.index == lfn.lenP {
		// Found the whole left part; save its position (state)
		lfn.state = state
	}
}

// LeftPermutationNavigator finds all left parts of words that are
// possible with a particular rack, and accumulates them by length.
// This is done once at the start of move generation.
type LeftPermutationNavigator struct {
	rack      string
	stack     []leftPermItem
	maxLeft   int
	leftParts [][]*LeftPart
	index     int
}

type leftPermItem struct {
	rack  string
	index int
}

// LeftPart stores the navigation state after matching a particular
// left part within the DAWG, so we can resume navigation from that
// point to complete an anchor square followed by a right part
type LeftPart struct {
	matched string
	rack    string
	state   *navState
}

// String returns a string representation of a LeftPart, for debugging purposes
func (lp *LeftPart) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("LeftPart: matched '%v' from rack '%v", lp.matched, lp.rack))
	return sb.String()
}

// FindLeftParts returns all left part permutations that can be generated
// from the given rack, grouped by length
func FindLeftParts(dawg *Dawg, rack string) [][]*LeftPart {
	var lpn LeftPermutationNavigator
	lpn.Init(rack)
	dawg.NavigateResumable(&lpn)
	return lpn.leftParts
}

// Init initializes a fresh LeftPermutationNavigator using the given rack
func (lpn *LeftPermutationNavigator) Init(rack string) {
	lpn.rack = rack
	// One tile from the rack will be put on the anchor square;
	// the rest is available to be played to the left of the anchor.
	// We thus find all permutations involving all rack tiles except
	// one.
	lpn.maxLeft = len([]rune(rack)) - 1
	lpn.stack = make([]leftPermItem, 0)
	lpn.leftParts = make([][]*LeftPart, lpn.maxLeft, lpn.maxLeft)
	for i := 0; i < lpn.maxLeft; i++ {
		lpn.leftParts[i] = make([]*LeftPart, 0)
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
	if !strings.ContainsRune(lpn.rack, chr) && !strings.ContainsRune(lpn.rack, '?') {
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
	exactMatch := strings.ContainsRune(lpn.rack, chr)
	if !exactMatch && !strings.ContainsRune(lpn.rack, '?') {
		return false
	}
	lpn.index++
	if exactMatch {
		lpn.rack = strings.Replace(lpn.rack, string(chr), "", 1)
	} else {
		lpn.rack = strings.Replace(lpn.rack, "?", "", 1)
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (lpn *LeftPermutationNavigator) Accept(matched string, final bool, state *navState) {
	ix := len([]rune(matched)) - 1
	lpn.leftParts[ix] = append(lpn.leftParts[ix],
		&LeftPart{matched: matched, rack: lpn.rack, state: state},
	)
}

// Robot finds a Move to play in a Game according to its strategy
type Robot struct {
}

// Axis stores information about a row or column on the board where
// the autoplayer is looking for valid moves
type Axis struct {
	state      *GameState
	horizontal bool
	// A bitmap of the letters in the rack, having all bits set if
	// the rack has a blank ('?') in it
	rackSet uint
	// Array of convenience pointers to the board squares on this Axis
	sq [BoardSize]*Square
	// A bitmap of the letters that are allowed on this square,
	// 0 if not an anchor square
	crossCheck [BoardSize]uint
}

// Init initializes a fresh Axis object, associating it with a board
// row or column
func (axis *Axis) Init(state *GameState, rackSet uint, index int, horizontal bool) {
	axis.state = state
	axis.rackSet = rackSet
	axis.horizontal = horizontal
	board := state.Board
	// Build an array of pointers to the squares on this axis
	for i := 0; i < BoardSize; i++ {
		if horizontal {
			axis.sq[i] = board.Sq(index, i)
		} else {
			axis.sq[i] = board.Sq(i, index)
		}
	}
	if board.NumTiles == 0 {
		// If no tile has yet been placed on the board,
		// mark the center square of the center column as an anchor
		// by setting its crossCheck set to allow the entire rack
		if index == BoardSize/2 && !horizontal {
			axis.crossCheck[BoardSize/2] = rackSet
		}
	} else {
		// Mark all empty squares having at least one occupied
		// adjacent square as anchors
		dawg := state.Dawg
		alphabetLength := dawg.alphabet.Length()
		for i := 0; i < BoardSize; i++ {
			sq := axis.sq[i]
			if sq.Tile == nil && board.NumAdjacentTiles(sq.Row, sq.Col) > 0 {
				// This is an anchor square
				crossSet := rackSet
				// Check whether the cross word(s) limit the set of allowed
				// letters in this anchor square
				left, right := board.CrossWords(sq.Row, sq.Col, !horizontal)
				lenLeft := len(left)
				if lenLeft > 0 || len(right) > 0 {
					// We ask the DAWG to find all words consisting of the
					// left cross word + wildcard + right cross word,
					// for instance 'f?lt' if the left word is 'f' and the
					// right one is 'lt' - yielding the result set
					// { 'falt', 'filt', fúlt' }, which we convert to the
					// legal cross set of [ 'a', 'i', 'ú' ] and intersect
					// that with the rack
					matches := dawg.Match(left + "?" + right)
					// Collect the 'middle' letters (the ones standing in
					// for the wildcard)
					runes := make([]rune, 0, alphabetLength)
					for _, match := range matches {
						runes = append(runes, ([]rune(match))[lenLeft])
					}
					// Intersect the set of allowed cross-check letters
					// with the rack
					crossSet &= dawg.alphabet.MakeSet(runes)
				}
				axis.crossCheck[i] = crossSet
			}
		}
	}
}

// genMovesFromAnchor returns the available moves that use the given square
// within the Axis as an anchor
func (axis *Axis) genMovesFromAnchor(anchor int, maxLeft int, leftParts [][]*LeftPart) []Move {
	dawg, board := axis.state.Dawg, axis.state.Board
	sq := axis.sq[anchor]
	var direction int
	if axis.horizontal {
		direction = LEFT
	} else {
		direction = ABOVE
	}
	if maxLeft == 0 && anchor > 0 && axis.sq[anchor-1].Tile == nil {
		// We have a left part already on the board: try to complete it
		// Get the left part, as a list of Tiles
		fragment := board.Fragment(sq.Row, sq.Col, direction)
		// The fragment list is backwards; convert it to a proper Prefix,
		// which is a list of runes
		left := make(Prefix, len(fragment))
		for i, tile := range fragment {
			left[len(fragment)-1-i] = tile.Meaning
		}
		// Do the DAWG navigation to find the left part
		var lfn LeftFindNavigator
		lfn.Init(left)
		dawg.NavigateResumable(&lfn)
		if lfn.state == nil {
			// No matching prefix found: there cannot be any
			// valid completions of the left part that is already
			// there. Return a nil slice.
			return nil
		}
		// We found a matching prefix in the graph:
		// do an ExtendRight from that location, using the whole rack
		var ern ExtendRightNavigator
		ern.Init(anchor, axis.state.Rack)
		dawg.Resume(&ern, lfn.state, string(left))
		// Return the move list accumulated by the ExtendRightNavigator
		return ern.moves
	}
	// We are not completing an existing left part
	// Begin by extending an empty prefix to the right, i.e. placing
	// tiles on the anchor square itself and to its right
	moves := make([]Move, 0)
	var ern ExtendRightNavigator
	ern.Init(anchor, axis.state.Rack)
	dawg.Navigate(&ern)
	// Collect the moves found so far
	moves = append(moves, ern.moves...)

	// Follow this by an effort to permute left prefixes into the
	// open space to the left of the anchor square
	for leftLen := 1; leftLen <= maxLeft; leftLen++ {
		leftList := leftParts[leftLen-1]
		for _, leftPart := range leftList {
			var ern ExtendRightNavigator
			ern.Init(anchor, leftPart.rack)
			dawg.Resume(&ern, leftPart.state, leftPart.matched)
			moves = append(moves, ern.moves...)
		}
	}
	return moves
}

func min(i1, i2 int) int {
	if i1 <= i2 {
		return i1
	}
	return i2
}

// GenerateMoves returns a list of all legal moves along this Axis
func (axis *Axis) GenerateMoves(lenRack int, leftParts [][]*LeftPart) []Move {
	moves := make([]Move, 0)
	lastAnchor := -1
	// Process the anchors, one by one, from left to right
	for i := 0; i < BoardSize; i++ {
		if axis.crossCheck[i] == 0 {
			// This is not an anchor, or at least not a square that we
			// can put a rack tile on
			continue
		}
		// This is an anchor square: count open squares to its left,
		// up to but not including the previous anchor, if any
		openCnt := 0
		left := i
		for left > 0 && left > (lastAnchor+1) && axis.sq[left-1].Tile == nil {
			openCnt++
			left--
		}
		moves = append(moves,
			axis.genMovesFromAnchor(i, min(openCnt, lenRack-1), leftParts)...,
		)
		lastAnchor = i
	}
	return moves
}

// GenerateMoves returns a list of all legal moves in the GameState,
// considering the Board and the player's Rack. The generation works
// by dividing the task into 30 sub-tasks of finding legal moves within
// each Axis, i.e. all columns and rows of the board. These sub-tasks
// are performed concurrently (and hopefully to some degree in parallel)
// by 30 goroutines.
func (state *GameState) GenerateMoves() []Move {
	rack := state.Rack.AsRunes()
	// Generate a bit map for the letters in the rack. If the rack
	// contains blank tiles ('?'), the bit map will have all bits set.
	rackSet := state.Dawg.alphabet.MakeSet(rack)
	lenRack := len(rack)
	leftParts := FindLeftParts(state.Dawg, string(rack))
	// Result channel containing up to BoardSize*2 move lists
	resultMoves := make(chan []Move, BoardSize*2)
	// Goroutine to find moves on a particular axis
	// (row or column)
	kickOffAxis := func(index int, horizontal bool) {
		var axis Axis
		axis.Init(state, rackSet, index, horizontal)
		// Generate a list of moves and send it on the result channel
		resultMoves <- axis.GenerateMoves(lenRack, leftParts)
	}
	// Start the 30 goroutines (columns and rows = 2 * BoardSize)
	// Horizontal rows
	for i := 0; i < BoardSize; i++ {
		go kickOffAxis(i, true)
	}
	// Vertical columns
	for i := 0; i < BoardSize; i++ {
		go kickOffAxis(i, false)
	}
	// Collect move candidates from all goroutines and
	// append them to the moves list
	moves := make([]Move, 0)
	for i := 0; i < BoardSize*2; i++ {
		moves = append(moves, (<-resultMoves)...)
	}
	// All goroutines have returned and we have a complete list
	// of generated moves
	return moves
}

// Init initializes a fresh Robot instance
func (robot *Robot) Init() {
}

// PickMove chooses a 'best' move to play from a list of legal moves,
// in accordance with the Robot's strategy
func (robot *Robot) PickMove([]Move) Move {
	// TODO
	return nil
}

// HighestScoreRobot returns an instance of a Robot playing with the
// HighestScore strategy, i.e. one that always picks the
// highest-scoring available move
func HighestScoreRobot() *Robot {
	return &Robot{}
}
