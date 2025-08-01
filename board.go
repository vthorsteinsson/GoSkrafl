// board.go
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.

// This file implements Board, Square and Tile structs
// and their associated operations

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
	"unicode"
)

const zero = int('0')

// BoardSize is the size of the Board
const BoardSize = 15

// Word multiplication factors on a standard board
var WORD_MULTIPLIERS_STANDARD = [BoardSize]string{
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

// Letter multiplication factors on a standard board
var LETTER_MULTIPLIERS_STANDARD = [BoardSize]string{
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

// Word multiplication factors on an Explo board
var WORD_MULTIPLIERS_EXPLO = [BoardSize]string{
	"311111131111113",
	"111111112111111",
	"111111111211111",
	"111211111111111",
	"111121111111111",
	"111112111111211",
	"111111211111121",
	"311111121111113",
	"121111112111111",
	"112111111211111",
	"111111111121111",
	"111111111112111",
	"111112111111111",
	"111111211111111",
	"311111131111113",
}

// Letter multiplication factors on an Explo board
var LETTER_MULTIPLIERS_EXPLO = [BoardSize]string{
	"111121111112111",
	"131112111111131",
	"112111311111211",
	"111111121131112",
	"211111111113111",
	"121111111211111",
	"113111112111111",
	"111211111112111",
	"111111211111311",
	"111112111111121",
	"111311111111112",
	"211131121111111",
	"112111113111211",
	"131111111211131",
	"111211111121111",
}

// colIds are the column identifiers of a board
var colIds = [BoardSize]string{
	"1", "2", "3", "4", "5",
	"6", "7", "8", "9", "10",
	"11", "12", "13", "14", "15",
}

// rowIds are the row identifiers of a board
var rowIds = [BoardSize]string{
	"A", "B", "C", "D", "E",
	"F", "G", "H", "I", "J",
	"K", "L", "M", "N", "O",
}

// Board represents the board as a matrix of Squares,
// and caches an adjacency matrix for each Square,
// consisting of pointers to adjacent Squares
type Board struct {
	Type      string // 'standard' or 'explo'
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

// String represents a Tile as a string
func (tile *Tile) String() string {
	if tile == nil {
		return "."
	}
	return string(tile.Letter)
}

// Coord converts row, col, across to "A1"/"1A" format.
func Coord(row, col int, horizontal bool) string {
	if horizontal {
		return fmt.Sprintf("%c%d", 'A'+row, col+1)
	}
	return fmt.Sprintf("%d%c", row+1, 'A'+col)
}

// Return the coordinate of the start square for this board type
func (board *Board) StartSquare() Coordinate {
	if board == nil || board.Type == "standard" {
		return Coordinate{7, 7} // H8
	} else {
		return Coordinate{3, 3} // D4
	}
}

// Return true if the board has a tile in the start square
func (board *Board) HasStartTile() bool {
	startSquare := board.StartSquare()
	sq := board.Sq(startSquare.Row, startSquare.Col)
	return sq != nil && sq.Tile != nil
}

// Sq returns a pointer to a Board square
func (board *Board) Sq(row, col int) *Square {
	if board == nil || row < 0 || row >= BoardSize ||
		col < 0 || col >= BoardSize {
		return nil
	}
	return &board.Squares[row][col]
}

// TileAt returns a pointer to the Tile in a given Square
func (board *Board) TileAt(row, col int) *Tile {
	if board == nil || row < 0 || row >= BoardSize ||
		col < 0 || col >= BoardSize {
		return nil
	}
	return board.Squares[row][col].Tile
}

// Place a tile in a board square, if it is empty
func (board *Board) PlaceTile(row, col int, tile *Tile) bool {
	sq := board.Sq(row, col)
	if sq == nil {
		return false
	}
	sq.Tile = tile
	board.NumTiles++
	return true
}

// String represents a Board as a string
func (board *Board) String() string {
	var sb strings.Builder
	sb.WriteString("  ")
	for i := 0; i < BoardSize; i++ {
		// Print the column id right-justified in a 2-character field,
		// plus a space, making the column 3 characters wide
		sb.WriteString(fmt.Sprintf("%2s ", colIds[i]))
	}
	sb.WriteString("\n")
	for i := 0; i < BoardSize; i++ {
		sb.WriteString(fmt.Sprintf("%s ", rowIds[i]))
		for j := 0; j < BoardSize; j++ {
			sb.WriteString(fmt.Sprintf(" %v ", board.Sq(i, j)))
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
func (board *Board) WordFragment(row, col int, direction int) (result []rune) {
	frag := board.Fragment(row, col, direction)
	lenFrag := len(frag)
	r := make([]rune, lenFrag)
	if direction == LEFT || direction == ABOVE {
		// We need to reverse the order of the fragment
		for ix, tile := range frag {
			// Assign the tile meaning to r in reverse order
			r[lenFrag-1-ix] = tile.Meaning
		}
	} else {
		// The fragment is in correct reading order
		for ix, tile := range frag {
			// Append the tile's meaning to the result
			r[ix] = tile.Meaning
		}
	}
	return r
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
func (board *Board) CrossWords(row, col int, horizontal bool) (left, right []rune) {
	var direction int
	// The C ternary operator is sorely missed :-(
	if horizontal {
		direction = LEFT
	} else {
		direction = ABOVE
	}
	hfrag := board.Fragment(row, col, direction)
	lenHfrag := len(hfrag)
	left = make([]rune, lenHfrag)
	for ix, tile := range hfrag {
		// Assign the tile meaning to left in reverse order
		left[lenHfrag-1-ix] = tile.Meaning
	}
	if horizontal {
		direction = RIGHT
	} else {
		direction = BELOW
	}
	vfrag := board.Fragment(row, col, direction)
	lenVfrag := len(vfrag)
	right = make([]rune, lenVfrag)
	for ix, tile := range vfrag {
		// Assign the tile meaning to right in forward order
		right[ix] = tile.Meaning
	}
	return // left, right
}

// Init initializes an empty board
func (board *Board) Init(boardType string) error {
	// Select the correct multipliers for the board type
	var letterMultipliers *[BoardSize]string
	var wordMultipliers *[BoardSize]string
	switch boardType {
	case "standard":
		letterMultipliers = &LETTER_MULTIPLIERS_STANDARD
		wordMultipliers = &WORD_MULTIPLIERS_STANDARD
	case "explo":
		letterMultipliers = &LETTER_MULTIPLIERS_EXPLO
		wordMultipliers = &WORD_MULTIPLIERS_EXPLO
	default:
		return fmt.Errorf("unknown board type: '%s'", boardType)
	}
	board.Type = boardType
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
	return nil
}

func NewBoard(boardType string) (*Board, error) {
	board := &Board{}
	if err := board.Init(boardType); err != nil {
		return nil, err
	}
	return board, nil
}

// ToStrings converts a Board object to a compact slice of strings,
// where blank tile meanings are represented by uppercase letters.
func (b *Board) ToStrings() []string {
	s := make([]string, BoardSize)
	for r := 0; r < BoardSize; r++ {
		var sb strings.Builder
		for c := 0; c < BoardSize; c++ {
			sq := b.Sq(r, c)
			if sq.Tile == nil {
				sb.WriteRune('.')
			} else if sq.Tile.Letter == '?' {
				// Blank tile on board is represented by its meaning, uppercase
				sb.WriteRune(unicode.ToUpper(sq.Tile.Meaning))
			} else {
				sb.WriteRune(sq.Tile.Letter)
			}
		}
		s[r] = sb.String()
	}
	return s
}
