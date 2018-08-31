// board.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file implements the Board and the Racks, together
// with their Squares and the Tiles that may occupy them

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

// BoardSize is the size of the Board
const BoardSize = 15

// RackSize contains the number of slots in the Rack
const RackSize = 7

// Board represents the board as a matrix of Squares,
// and caches an adjacency matrix for each Square,
// consisting of pointers to adjacent Squares
type Board struct {
	Squares   [BoardSize][BoardSize]Square
	Adjacents [BoardSize][BoardSize]AdjSquares
	// The number of tiles on the board
	NumTiles int
}

// Indices into AdjSquares
const (
	ABOVE = 0
	LEFT  = 1
	RIGHT = 2
	BELOW = 3
)

// AdjSquares is a list of four Square pointers,
// with a nil if the corresponding adjacent Square does not exist
type AdjSquares [4]*Square

// Rack represents a player's rack of Tiles
type Rack struct {
	Slots [RackSize]Square
	// Letters is a map of letters in the rack with their count,
	// with blank tiles being represented by '?'
	Letters map[rune]int
}

// Tile is a tile from the Bag
type Tile struct {
	Letter   rune
	Meaning  rune // Meaning of blank tile (if Letter=='?')
	Score    int  // The nominal score of the tile
	PlayedBy int  // Which player played the tile
}

// Square is a Board square or Rack slot that can hold a Tile
type Square struct {
	Tile             *Tile
	LetterMultiplier int
	WordMultiplier   int
	Row              int // Board row 0..14, or -1 if rack square
	Col              int // Board column 0..14, or rack square 0..6
}

// String represents a Square as a string. An empty
// Square is indicated by a dot ('.').
func (square *Square) String() string {
	if square.Tile == nil {
		// Empty square
		return "."
	}
	if square.Tile.Letter == '?' && square.Row >= 0 {
		// Blank tile on the board: show its meaning
		return string(square.Tile.Meaning)
	}
	// If a blank tile is in the rack, show '?'
	return string(square.Tile.Letter)
}

// colIds are the column identifiers of a board
var colIds = [BoardSize]string{
	"A", "B", "C", "D", "E",
	"F", "G", "H", "I", "J",
	"L", "M", "N", "O", "P",
}

// rowIds are the row identifiers of a board
var rowIds = [BoardSize]string{
	"1", "2", "3", "4", "5",
	"6", "7", "8", "9", "10",
	"11", "12", "13", "14", "15",
}

// Fill draws tiles from the bag to fill a rack.
// Returns false if unable to fill all empty slots.
func (rack *Rack) Fill(bag *Bag) bool {
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		if sq.Tile == nil {
			// Empty slot: draw a tile from the bag
			sq.Tile = bag.DrawTile()
		}
		if sq.Tile != nil {
			// Got a new tile in the rack:
			// increment the letter's count in the rack map
			letter := sq.Tile.Letter
			if _, ok := rack.Letters[letter]; ok {
				rack.Letters[letter]++
			} else {
				rack.Letters[letter] = 1
			}
		} else {
			// Can't fill all empty slots: return false
			return false
		}
	}
	// Able to fill all empty slots
	return true
}

// String represents a Tile as a string
func (tile *Tile) String() string {
	if tile == nil {
		return "."
	}
	return string(tile.Letter)
}

// Sq returns a pointer to a Board square
func (board *Board) Sq(row, col int) *Square {
	return &board.Squares[row][col]
}

// TileAt returns a pointer to the Tile in a given Square
func (board *Board) TileAt(row, col int) *Tile {
	if row < 0 || row >= BoardSize || col < 0 || col >= BoardSize {
		return nil
	}
	return board.Squares[row][col].Tile
}

// String represents a Board as a string
func (board *Board) String() string {
	var sb strings.Builder
	sb.WriteString("   ")
	for i := 0; i < BoardSize; i++ {
		sb.WriteString(colIds[i] + " ")
	}
	sb.WriteString("\n")
	for i := 0; i < BoardSize; i++ {
		sb.WriteString(fmt.Sprintf("%2s ", rowIds[i]))
		for j := 0; j < BoardSize; j++ {
			sb.WriteString(fmt.Sprintf("%v ", board.Sq(i, j)))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// NumAdjacentTiles returns the number of tiles on the
// Board that are adjacent to the given coordinate
func (board *Board) NumAdjacentTiles(row, col int) int {
	adj := &board.Adjacents[row][col]
	var count = 0
	for _, sq := range adj {
		if sq != nil && sq.Tile != nil {
			count++
		}
	}
	return count
}

// Fragment returns a list of the tiles that extend from the square
// at row, col in the direction specified (ABOVE/BELOW/LEFT/RIGHT).
func (board *Board) Fragment(row, col int, direction int) []*Tile {
	if row < 0 || col < 0 || row >= BoardSize || col >= BoardSize {
		return nil
	}
	if direction < ABOVE || direction > BELOW {
		return nil
	}
	frag := make([]*Tile, 0, BoardSize-1)
	for {
		sq := board.Adjacents[row][col][direction]
		if sq == nil || sq.Tile == nil {
			break
		}
		frag = append(frag, sq.Tile)
		row, col = sq.Row, sq.Col
	}
	return frag
}

// WordFragment returns the word formed by the tile sequence emanating
// from the given square in the indicated direction, not including the
// square itself.
func (board *Board) WordFragment(row, col int, direction int) (result string) {
	frag := board.Fragment(row, col, direction)
	if direction == LEFT || direction == ABOVE {
		// We need to reverse the order of the fragment
		for _, tile := range frag {
			result = string(tile.Meaning) + result
		}
	} else {
		// The fragment is in correct reading order
		for _, tile := range frag {
			result += string(tile.Meaning)
		}
	}
	return // result
}

// CrossScore returns the sum of the scores of the tiles crossing
// the given tile, either horizontally or vertically. If there are no
// crossings, returns false, 0. (Note that true, 0 is a valid return
// value, if a crossing has only blank tiles.)
func (board *Board) CrossScore(row, col int, horizontal bool) (hasCrossing bool, score int) {
	var direction int
	// The C ternary operator is sorely missed :-(
	if horizontal {
		direction = LEFT
	} else {
		direction = ABOVE
	}
	for _, tile := range board.Fragment(row, col, direction) {
		score += tile.Score
		hasCrossing = true
	}
	if horizontal {
		direction = RIGHT
	} else {
		direction = BELOW
	}
	for _, tile := range board.Fragment(row, col, direction) {
		score += tile.Score
		hasCrossing = true
	}
	return // hasCrossing, score
}

// CrossWords returns the word fragments above and below, or to the left and right of, the
// given co-ordinate on the board.
func (board *Board) CrossWords(row, col int, horizontal bool) (left, right string) {
	var direction int
	// The C ternary operator is sorely missed :-(
	if horizontal {
		direction = LEFT
	} else {
		direction = ABOVE
	}
	for _, tile := range board.Fragment(row, col, direction) {
		left = string(tile.Meaning) + left
	}
	if horizontal {
		direction = RIGHT
	} else {
		direction = BELOW
	}
	for _, tile := range board.Fragment(row, col, direction) {
		right += string(tile.Meaning)
	}
	return // left, right
}

// Init initializes an empty board
func (board *Board) Init() {

	var wordMultipliers = [BoardSize]string{
		"311111131111113",
		"121111111111121",
		"112111111111211",
		"111211111112111",
		"111121111121111",
		"111111111111111",
		"111111111111111",
		"311111121111113",
		"111111111111111",
		"111111111111111",
		"111121111121111",
		"111211111112111",
		"112111111111211",
		"121111111111121",
		"311111131111113",
	}

	var letterMultipliers = [BoardSize]string{
		"111211111112111",
		"111113111311111",
		"111111212111111",
		"211111121111112",
		"111111111111111",
		"131113111311131",
		"112111212111211",
		"111211111112111",
		"112111212111211",
		"131113111311131",
		"111111111111111",
		"211111121111112",
		"111111212111111",
		"111113111311111",
		"111211111112111",
	}

	const zero = int('0')
	for i := 0; i < BoardSize; i++ {
		for j := 0; j < BoardSize; j++ {
			sq := board.Sq(i, j)
			sq.Row = i
			sq.Col = j
			sq.LetterMultiplier = int(letterMultipliers[i][j]) - zero
			sq.WordMultiplier = int(wordMultipliers[i][j]) - zero
		}
	}
	// Initialize the cached matrix of adjacent square lists
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			var adj = &board.Adjacents[row][col]
			if row > 0 {
				// Square above
				adj[ABOVE] = board.Sq(row-1, col)
			}
			if row < BoardSize-1 {
				// Square below
				adj[BELOW] = board.Sq(row+1, col)
			}
			if col > 0 {
				// Square to the left
				adj[LEFT] = board.Sq(row, col-1)
			}
			if col < BoardSize-1 {
				// Square to the right
				adj[RIGHT] = board.Sq(row, col+1)
			}
		}
	}
}

// Init initializes an empty rack
func (rack *Rack) Init() {
	// Make an empty letter map
	rack.Letters = make(map[rune]int)
	// Initialize empty rack slots
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		sq.Row = -1
		sq.Col = i
		sq.LetterMultiplier = 1
		sq.WordMultiplier = 1
	}
}

// String returns a printable string representation of a Rack
func (rack *Rack) String() string {
	var sb strings.Builder
	for i := 0; i < RackSize; i++ {
		sb.WriteString(fmt.Sprintf("%v ", &rack.Slots[i]))
	}
	return sb.String()
}

// AsRunes returns the tiles in the Rack as a list of runes
func (rack *Rack) AsRunes() []rune {
	runes := make([]rune, 0, RackSize)
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		if sq.Tile != nil {
			runes = append(runes, sq.Tile.Letter)
		}
	}
	return runes
}

// AsString returns the tiles in the Rack as a contiguous string
func (rack *Rack) AsString() string {
	return string(rack.AsRunes())
}

// AsSet returns the rack as a bit-mapped set of runes.
// If the rack contains a blank tile ('?'), the bitmap
// will have all bits set.
func (rack *Rack) AsSet(alphabet *Alphabet) uint {
	return alphabet.MakeSet(rack.AsRunes())
}

// HasTile returns true if the given Tile is in the Rack
func (rack *Rack) HasTile(tile *Tile) bool {
	if tile == nil {
		return false
	}
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		if sq.Tile == tile {
			return true
		}
	}
	return false
}

// IsEmpty returns true if the Rack is empty
func (rack *Rack) IsEmpty() bool {
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		if sq.Tile != nil {
			return false
		}
	}
	return true
}

// FindTile finds a tile with the given letter (or '?') in the
// rack and returns a pointer to it, or nil if not found
func (rack *Rack) FindTile(letter rune) *Tile {
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		if sq.Tile != nil && sq.Tile.Letter == letter {
			return sq.Tile
		}
	}
	return nil
}

// RemoveTile removes a tile from a Rack
func (rack *Rack) RemoveTile(tile *Tile) bool {
	if tile == nil {
		return false
	}
	for i := 0; i < RackSize; i++ {
		sq := &rack.Slots[i]
		if sq.Tile == tile {
			// Found the slot with the tile:
			// remove it from the rack
			sq.Tile = nil
			rack.Letters[tile.Letter]--
			return true
		}
	}
	// Tile was not found in the rack
	return false
}

// Extract obtains the given number of tiles from the rack,
// returning them as a list. If a tile is blank,
// assign the given meaning to it.
func (rack *Rack) Extract(numTiles int, meaning rune) []*Tile {
	ex := make([]*Tile, 0, numTiles)
	for i := 0; i < RackSize && numTiles > 0; i++ {
		tile := rack.Slots[i].Tile
		if tile != nil {
			if tile.Letter == '?' {
				tile.Meaning = meaning
			}
			ex = append(ex, tile)
			numTiles--
		}
	}
	return ex
}
