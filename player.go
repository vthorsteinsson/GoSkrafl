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
	matched string
	state   *navState
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
		lfn.matched = matched
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
	var nav Navigation
	nav.isResumable = true
	nav.Go(dawg, &lpn)
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
	game *Game
}

// Axis stores information about a row or column on the board where
// the autoplayer is looking for valid moves
type Axis struct {
	game       *Game
	isAnchor   [BoardSize]bool
	crossCheck [BoardSize]uint
	sq         [BoardSize]*Square
}

// Init initializes a fresh Axis object, associating it with a board
// row or column
func (axis *Axis) Init(game *Game, index int, horizontal bool) {
	axis.game = game
	// Build an array of pointers to the squares on this axis
	for i := 0; i < BoardSize; i++ {
		if horizontal {
			axis.sq[i] = game.Board.Sq(index, i)
		} else {
			axis.sq[i] = game.Board.Sq(i, index)
		}
	}
	if game.NumTiles == 0 {
		// If no tile has yet been placed on the board,
		// mark the center square as an anchor
		axis.isAnchor[BoardSize/2] = (index == BoardSize/2)
	} else {
		// TODO: Calculate isAnchor array
	}
	// TODO: Compute cross sets, etc.
}

// genMovesFromAnchor returns the available moves that use the given square
// within the Axis as an anchor
func (axis *Axis) genMovesFromAnchor(anchor int, maxLeft int, leftParts [][]*LeftPart) []Move {
	// TODO
	return make([]Move, 0)
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
		if !axis.isAnchor[i] {
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

// Init initializes a fresh Robot instance
func (robot *Robot) Init(game *Game) {
	robot.game = game
	// robot.moves is intentionally left as a nil slice
}

// GenerateMoves returns a list of all legal moves in a Game
func (robot *Robot) GenerateMoves() []Move {
	game := robot.game
	rack := game.Racks[game.PlayerToMove()].AsString()
	lenRack := len([]rune(rack))
	leftParts := FindLeftParts(game.Dawg, rack)
	// Result channel containing up to BoardSize*2 move lists
	resultMoves := make(chan []Move, BoardSize*2)
	// Goroutine to find moves on a particular axis
	// (row or column)
	kickOffAxis := func(index int, horizontal bool) {
		var axis Axis
		axis.Init(game, index, horizontal)
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

// PickMove chooses a 'best' move to play from a list of legal moves,
// in accordance with the Robot's strategy
func (robot *Robot) PickMove([]Move) Move {
	// TODO
	return nil
}

// HighestScoreRobot returns an instance of a Robot playing with the
// HighestScore strategy, i.e. one that always picks the
// highest-scoring available move
func HighestScoreRobot(game *Game) *Robot {
	robot := &Robot{}
	robot.Init(game)
	return robot
}
