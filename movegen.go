// movegen.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file contains code to generate all valid tile moves
// on a SCRABBLE(tm) board, given a player's rack.
// It is a part of the Go 'skrafl' package.

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

/*

The code herein finds all legal moves on a SCRABBLE(tm)-like board.

The algorithm is based on the classic paper by Appel & Jacobson,
"The World's Fastest Scrabble Program",
http://www.cs.cmu.edu/afs/cs/academic/class/15451-s06/www/lectures/scrabble.pdf

The main function in this module is GameState.GenerateMoves(). Given
a game state, comprising a Board, a Rack, and a vocabulary word graph
(DAWG), it returns all legal tile moves.

Moves are found by examining each one-dimensional Axis of the board
in turn, i.e. 15 rows and 15 columns for a total of 30 axes.
For each Axis an array of pointers to its corresponding Board Squares
is constructed. The cross-check set of each empty Square is calculated,
i.e. the set of letters that form valid words by connecting with word parts
across the square's Axis. To save processing time, the cross-check sets
are also intersected with the letters in the rack, unless the rack contains
a blank tile.

Any empty square with a non-null cross-check set or adjacent to
a covered square within the axis is a potential anchor square.
Each anchor square is examined in turn, from "left" to "right".
The algorithm roughly proceeds as follows:

1) Count the number of empty non-anchor squares to the left of
	the anchor, which may be zero. Call the number 'maxleft'.
2) Generate all permutations of rack tiles found by navigating
	from the root of the DAWG, of length 1..maxleft, i.e. all
	possible word beginnings from the rack. (We calculate these
	permutation lists only once for the entire move generation
	phase.)
3) For each such permutation, attempt to complete the
	word by placing the rest of the available tiles on the
	anchor square and to its right.
4) In any case, even if maxleft=0, place a starting tile on the
	anchor square and attempt to complete a word to its right.
5) When placing a tile on the anchor square or to its right,
	do so under three constraints: (a) the cross-check
	set of the square in question; (b) that there is
	a path in the DAWG corresponding to the tiles that have
	been laid down so far, incl. step 2 and 3; (c) a matching
	tile is still available in the rack (with blank tiles always
	matching).
6) If extending to the right and coming to a tile that is
	already on the board, it must correspond to the DAWG path
	being followed.
7) If we are running off the edge of the axis, or have come
	to an empty square, and we are at a final node in the
	DAWG indicating that a word is completed, we have a candidate
	move. Calculate its score and add it to the list of potential
	moves.

Steps 1)-3) above are mostly implemented in the class LeftPartNavigator,
while steps 4)-7) are found in ExtendRightNavigator. These classes
correspond to the Appel & Jacobson LeftPart and ExtendRight functions.

Note: SCRABBLE is a registered trademark. This software or its author
are in no way affiliated with or endorsed by the owners or licensees
of the SCRABBLE trademark.

*/

package skrafl

import (
	"fmt"
	"strings"
)

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
	lenRack := len([]rune(rack))
	if lenRack <= 1 {
		// No left permutation possible
		lpn.maxLeft = 0
	} else {
		lpn.maxLeft = lenRack - 1
	}
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

// ExtendRightNavigator implements the core of the Appel-Jacobson
// algorithm. It proceeds along an Axis, covering empty Squares with
// Tiles from the Rack while obeying constraints from the Dawg and
// the cross-check sets. As final nodes in the Dawg are encountered,
// valid tile moves are generated and saved.
type ExtendRightNavigator struct {
	axis           *Axis
	anchor         int
	index          int
	rack           string
	stack          []ernItem
	lastCheck      int
	wildcardInRack bool
	// The list of valid tile moves found
	moves []Move
}

type ernItem struct {
	rack           string
	index          int
	wildcardInRack bool
}

// Matching constants
const (
	mNo        = 1
	mBoardTile = 2
	mRackTile  = 3
)

// Init initializes a fresh ExtendRightNavigator for an axis, starting
// from the given anchor, using the indicated rack
func (ern *ExtendRightNavigator) Init(axis *Axis, anchor int, rack string) {
	ern.axis = axis
	ern.anchor = anchor
	ern.index = anchor
	ern.rack = rack
	ern.wildcardInRack = strings.ContainsRune(rack, '?')
	ern.stack = make([]ernItem, 0, RackSize)
	ern.moves = make([]Move, 0)
}

func (ern *ExtendRightNavigator) check(letter rune) int {
	tileAtSq := ern.axis.sq[ern.index].Tile
	if tileAtSq != nil {
		// There is a tile in the square: must match it exactly
		if letter == tileAtSq.Letter {
			// Matches, from the board
			return mBoardTile
		}
		// Doesn't match the tile that is already there
		return mNo
	}
	// Does the current rack allow this letter?
	if !ern.wildcardInRack && !strings.ContainsRune(ern.rack, letter) {
		// No, it doesn't
		return mNo
	}
	// Finally, test the cross-checks
	if ern.axis.Allows(ern.index, letter) {
		// The tile successfully completes any cross-words
		/*
			// DEBUG: verify that the cross-checks hold
			sq := ern.axis.sq[ern.index]
			left, right := ern.axis.state.Board.CrossWords(sq.Row, sq.Col, !ern.axis.horizontal)
			if left != "" || right != "" {
				word := left + string(letter) + right
				if !ern.axis.state.Dawg.Find(word) {
					panic("Cross-check violation!")
				}
			}
		*/
		return mRackTile
	}
	return mNo
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (ern *ExtendRightNavigator) PushEdge(letter rune) bool {
	ern.lastCheck = ern.check(letter)
	if ern.lastCheck == mNo {
		// No way that this letter can be laid down here
		return false
	}
	// Match: save our rack and our index and move into the edge
	ern.stack = append(ern.stack, ernItem{ern.rack, ern.index, ern.wildcardInRack})
	return true
}

// PopEdge returns false if there is no need to visit other edges
// after this one has been traversed
func (ern *ExtendRightNavigator) PopEdge() bool {
	// Pop the previous rack and index from the stack
	last := len(ern.stack) - 1
	sp := &ern.stack[last]
	ern.rack, ern.index, ern.wildcardInRack = sp.rack, sp.index, sp.wildcardInRack
	ern.stack = ern.stack[0:last]
	// We need to visit all outgoing edges, so return true
	return true
}

// Done is called when the navigation is complete
func (ern *ExtendRightNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (ern *ExtendRightNavigator) IsAccepting() bool {
	if ern.index >= BoardSize {
		// Gone off the board edge
		return false
	}
	// Otherwise, continue while we have something on the rack
	// or we're at an occupied square
	return len(ern.rack) > 0 || ern.axis.sq[ern.index] != nil
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (ern *ExtendRightNavigator) Accepts(letter rune) bool {
	// We are on the anchor square or to its right
	match := ern.lastCheck
	if match == 0 {
		// No cached check available from PushEdge
		match = ern.check(letter)
	}
	ern.lastCheck = 0
	if match == mNo {
		// No fit anymore: we're done with this edge
		return false
	}
	// This letter is OK: accept it and remove from the rack if
	// it came from there
	ern.index++
	if match == mRackTile {
		if strings.ContainsRune(ern.rack, letter) {
			// Used a normal tile
			ern.rack = strings.Replace(ern.rack, string(letter), "", 1)
		} else {
			// Used a blank tile
			ern.rack = strings.Replace(ern.rack, "?", "", 1)
		}
		ern.wildcardInRack = strings.ContainsRune(ern.rack, '?')
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (ern *ExtendRightNavigator) Accept(matched string, final bool, state *navState) {
	if state != nil {
		panic("ExtendRightNavigator should not be resumable")
	}
	if !final ||
		(ern.index < BoardSize && ern.axis.sq[ern.index].Tile != nil) {
		// Not a complete word, or ends on an occupied square:
		// not a legal tile move
		return
	}
	runes := []rune(matched)
	if len(runes) < 2 {
		// Less than 2 letters long: not a legal tile move
		return
	}
	// Legal move found: make a TileMove object for it and add to
	// the move list
	covers := make(Covers)
	// Calculate the starting index within the axis
	start := ern.index - len(runes)
	// The original rack
	rack := ern.axis.rackString
	for i, meaning := range runes {
		sq := ern.axis.sq[start+i]
		if sq.Tile == nil {
			letter := meaning
			if strings.ContainsRune(rack, meaning) {
				rack = strings.Replace(rack, string(meaning), "", 1)
			} else {
				// Must be using a blank tile
				letter = '?'
				rack = strings.Replace(rack, "?", "", 1)
			}
			covers[Coordinate{sq.Row, sq.Col}] = Cover{letter, meaning}
		}
	}
	tileMove := NewTileMove(ern.axis.state.Board, covers)
	ern.moves = append(ern.moves, tileMove)
}

// Axis stores information about a row or column on the board where
// the robot player is looking for valid moves
type Axis struct {
	state      *GameState
	horizontal bool
	// A bitmap of the letters in the rack, having all bits set if
	// the rack has a blank ('?') in it
	rackSet uint
	// rackString is the original rack, stored as a string
	rackString string
	// Array of convenience pointers to the board squares on this Axis
	sq [BoardSize]*Square
	// A bitmap of the letters that are allowed on each square,
	// intersected with the current rack
	crossCheck [BoardSize]uint
	// A boolean for each square indicating whether it is an anchor
	// square
	isAnchor [BoardSize]bool
}

// Init initializes a fresh Axis object, associating it with a board
// row or column
func (axis *Axis) Init(state *GameState, rackSet uint, index int, horizontal bool) {
	axis.state = state
	axis.rackSet = rackSet
	axis.horizontal = horizontal
	axis.rackString = state.Rack.AsString()
	board := state.Board
	// Build an array of pointers to the squares on this axis
	for i := 0; i < BoardSize; i++ {
		if horizontal {
			axis.sq[i] = board.Sq(index, i)
		} else {
			axis.sq[i] = board.Sq(i, index)
		}
	}
	// Mark all empty squares having at least one occupied
	// adjacent square as anchors
	for i := 0; i < BoardSize; i++ {
		sq := axis.sq[i]
		if sq.Tile != nil {
			// Already have a tile here: not an anchor and no
			// cross-check set needed
			continue
		}
		var isAnchor bool
		if board.NumTiles == 0 {
			// Special case:
			// If no tile has yet been placed on the board,
			// mark the center square of the center column as an anchor
			isAnchor = (index == BoardSize/2) && (i == BoardSize/2) && !horizontal
		} else {
			isAnchor = board.NumAdjacentTiles(sq.Row, sq.Col) > 0
		}
		if !isAnchor {
			// Empty square with no adjacent tiles: not an anchor,
			// and we can place any letter from the rack here
			axis.crossCheck[i] = rackSet
		} else {
			// This is an anchor square, i.e. an empty square with
			// at least one adjacent tile. Note, however, that the
			// cross-check set for it may be zero, if no tile from
			// the rack can be placed in it due to cross-words.
			axis.isAnchor[i] = true
			axis.crossCheck[i] = rackSet & axis.crossSet(sq)
		}
	}
}

func (axis *Axis) crossSet(sq *Square) uint {
	// Check whether the cross word(s) limit the set of allowed
	// letters in this anchor square
	left, right := axis.state.Board.CrossWords(sq.Row, sq.Col, !axis.horizontal)
	if len(left) == 0 && len(right) == 0 {
		// No cross word, so no cross check constraint
		return ^uint(0)
	}
	return axis.state.Dawg.CrossSet(left, right)
}

// IsAnchor returns true if the given square within the Axis
// is an anchor square
func (axis *Axis) IsAnchor(index int) bool {
	return axis.isAnchor[index]
}

// IsOpen returns true if the given square within the Axis
// is open for a new Tile from the Rack
func (axis *Axis) IsOpen(index int) bool {
	return axis.sq[index].Tile == nil && axis.crossCheck[index] > 0
}

// Allows returns true if the given letter can be placed
// in the indexed square within the Axis, in compliance
// with the cross checks
func (axis *Axis) Allows(index int, letter rune) bool {
	if axis == nil || axis.sq[index].Tile != nil {
		// We already have a tile in this square
		return false
	}
	return axis.state.Dawg.alphabet.Member(letter, axis.crossCheck[index])
}

// genMovesFromAnchor returns the available moves that use the given square
// within the Axis as an anchor
func (axis *Axis) genMovesFromAnchor(anchor int, maxLeft int, leftParts [][]*LeftPart) []Move {
	dawg, board := axis.state.Dawg, axis.state.Board
	sq := axis.sq[anchor]

	// Do we have a left part already on the board?
	if maxLeft == 0 && anchor > 0 && axis.sq[anchor-1].Tile != nil {
		// Yes: try to complete it
		var direction int
		if axis.horizontal {
			direction = LEFT
		} else {
			direction = ABOVE
		}
		// Get the entire left part, as a list of Tiles
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
			// there.
			return nil
		}
		// We found a matching prefix in the graph:
		// do an ExtendRight from that location, using the whole rack
		var ern ExtendRightNavigator
		ern.Init(axis, anchor, axis.rackString)
		dawg.Resume(&ern, lfn.state, string(left))
		// Return the move list accumulated by the ExtendRightNavigator
		return ern.moves
	}

	// We are not completing an existing left part
	// Begin by extending an empty prefix to the right, i.e. placing
	// tiles on the anchor square itself and to its right
	moves := make([]Move, 0)
	var ern ExtendRightNavigator
	ern.Init(axis, anchor, axis.rackString)
	dawg.Navigate(&ern)
	// Collect the moves found so far
	moves = append(moves, ern.moves...)

	// Follow this by an effort to permute left prefixes into the
	// open space to the left of the anchor square, if any
	for leftLen := 1; leftLen <= maxLeft; leftLen++ {
		// Try all left prefixes of length leftLen
		leftList := leftParts[leftLen-1]
		for _, leftPart := range leftList {
			var ern ExtendRightNavigator
			ern.Init(axis, anchor, leftPart.rack)
			dawg.Resume(&ern, leftPart.state, leftPart.matched)
			moves = append(moves, ern.moves...)
		}
	}

	// Return the accumulated move list
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
		if !axis.IsAnchor(i) {
			continue
		}
		// This is an anchor
		if axis.crossCheck[i] > 0 {
			// A tile from the rack can actually be placed here:
			// count open squares to the anchor's left,
			// up to but not including the previous anchor, if any.
			// Open squares are squares that are empty and can
			// accept a tile from the rack.
			openCnt := 0
			left := i
			for left > 0 && left > (lastAnchor+1) && axis.IsOpen(left-1) {
				openCnt++
				left--
			}
			moves = append(moves,
				axis.genMovesFromAnchor(i, min(openCnt, lenRack-1), leftParts)...,
			)
		}
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
