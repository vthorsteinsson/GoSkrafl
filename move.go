// move.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson

// This file implements the Move interface and associated logic,
// including the various types of moves and their validation.

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
	"strings"
)

// Move is an interface to various types of moves
type Move interface {
	IsValid(*Game) bool
	Apply(*Game) bool
	Score(*GameState) int
}

// PassMove is a move that is always valid, has no effect when applied,
// and has a score of 0
type PassMove struct {
}

// ExchangeMove is a move that exchanges 1-7 tiles from the player's
// Rack with the Bag. It is only valid when at least 7 tiles are
// left in the Bag.
type ExchangeMove struct {
	Letters string
}

// FinalMove represents the final adjustments that are made to
// player scores at the end of a Game
type FinalMove struct {
	OpponentRack   string
	MultiplyFactor int
}

// TileMove represents a normal tile move by a player, where
// one or more Squares are covered by a Tile from the player's Rack
type TileMove struct {
	TopLeft     Coordinate
	BottomRight Coordinate
	Covers      Covers
	Horizontal  bool
	Word        string
	CachedScore *int
	// If ValidateWords is true, IsValid() should check all words
	// formed by this move against the game dictionary
	ValidateWords bool
}

// Coordinate stores a Board co-ordinate as as row, col tuple
type Coordinate struct {
	Row, Col int
}

// Cover is a part of a TileMove, describing the covering of
// a single Square by a Letter. The Letter may be '?' indicating a
// blank tile, in which case the Meaning gives its meaning.
type Cover struct {
	Letter  rune
	Meaning rune
}

// Covers is a map of board coordinates to a tile covering
type Covers map[Coordinate]Cover

// BingoBonus is the number of extra points awarded for laying down
// all the 7 tiles in the rack in one move
const BingoBonus = 50

// NewTileMove creates a new TileMove object with the given
// Covers, i.e. Tile coverings
func NewTileMove(board *Board, covers Covers) *TileMove {
	move := &TileMove{}
	move.Init(board, covers)
	return move
}

// String return a string description of a TileMove
func (move *TileMove) String() string {
	// TODO: This returns only the coordinates of the move
	var coord string
	if move.Horizontal {
		coord = rowIds[move.TopLeft.Row] + colIds[move.TopLeft.Col]
	} else {
		coord = colIds[move.TopLeft.Col] + rowIds[move.TopLeft.Row]
	}
	return coord + " " + move.Word
}

// IllegalMoveWord is the move.Word of an illegal move
const IllegalMoveWord = "[???]"

// Init initializes a TileMove instance for a particular Board
// using a map of Coordinate to Cover
func (move *TileMove) Init(board *Board, covers Covers) {
	move.Covers = covers
	top, left := BoardSize, BoardSize
	bottom, right := -1, -1
	for coord := range covers {
		if coord.Row < top {
			top = coord.Row
		}
		if coord.Col < left {
			left = coord.Col
		}
		if coord.Row > bottom {
			bottom = coord.Row
		}
		if coord.Col > right {
			right = coord.Col
		}
	}
	move.TopLeft = Coordinate{top, left}
	move.BottomRight = Coordinate{bottom, right}
	if len(covers) >= 2 {
		// This is horizontal if the first two covers are in the same row
		move.Horizontal = top == bottom
	} else {
		// Single cover: get smart and figure out whether the
		// horizontal cross is longer than the vertical cross
		hcross := len(board.Fragment(top, left, LEFT)) +
			len(board.Fragment(top, left, RIGHT))
		vcross := len(board.Fragment(top, left, ABOVE)) +
			len(board.Fragment(top, left, BELOW))
		move.Horizontal = hcross >= vcross
	}
	// Collect the entire word that is being laid down
	var direction, reverse int
	if move.Horizontal {
		direction = RIGHT
		reverse = LEFT
	} else {
		direction = BELOW
		reverse = ABOVE
	}
	sq := board.Sq(top, left)
	if sq == nil {
		move.Word = IllegalMoveWord
		return
	}
	// Start with any left prefix that is being extended
	word := board.WordFragment(top, left, reverse)
	// Next, traverse the covering line from top left to bottom right
	for {
		if cover, ok := covers[Coordinate{sq.Row, sq.Col}]; ok {
			// This square is being covered by the tile move
			word += string(cover.Meaning)
		} else {
			// This square must be covered by a previously laid tile
			if sq.Tile == nil {
				move.Word = IllegalMoveWord
				return
			}
			word += string(sq.Tile.Meaning)
		}
		if sq.Row == bottom && sq.Col == right {
			// This was the last tile laid down in the move:
			// the loop is done
			break
		}
		// Move to the next adjacent square, in the direction of the move
		sq = board.Adjacents[sq.Row][sq.Col][direction]
		if sq == nil {
			move.Word = IllegalMoveWord
			return
		}
	}
	// Add any suffix that may already have been on the board
	word += board.WordFragment(bottom, right, direction)
	move.Word = word
}

// IsValid returns true if the TileMove is valid in the current Game
func (move *TileMove) IsValid(game *Game) bool {
	// Check the validity of the move
	if len(move.Covers) < 1 || len(move.Covers) > RackSize {
		return false
	}
	board := game.Board
	// Count the number of tiles adjacent to the covers
	var numAdjacentTiles = 0
	for coord := range move.Covers {
		if coord.Row < 0 || coord.Row >= BoardSize ||
			coord.Col < 0 || coord.Col >= BoardSize {
			return false
		}
		if board.TileAt(coord.Row, coord.Col) != nil {
			// There is already a tile in this square
			return false
		}
		numAdjacentTiles += board.NumAdjacentTiles(coord.Row, coord.Col)
	}
	if move.BottomRight.Row > move.TopLeft.Row &&
		move.BottomRight.Col > move.TopLeft.Col {
		// Not strictly horizontal or strictly vertical
		return false
	}
	// Check for gaps
	if move.Horizontal {
		// This is a horizontal move
		row := move.TopLeft.Row
		for i := move.TopLeft.Col; i <= move.BottomRight.Col; i++ {
			_, covered := move.Covers[Coordinate{row, i}]
			if !covered && board.TileAt(row, i) == nil {
				// There is a missing square in the covers
				return false
			}
		}
	} else {
		// This is a vertical move
		col := move.TopLeft.Col
		for i := move.TopLeft.Row; i <= move.BottomRight.Row; i++ {
			_, covered := move.Covers[Coordinate{i, col}]
			if !covered && board.TileAt(i, col) == nil {
				// There is a missing square in the covers
				return false
			}
		}
	}
	// The first tile move must go through the center
	if board.NumTiles == 0 {
		if _, covered := move.Covers[Coordinate{BoardSize / 2, BoardSize / 2}]; !covered {
			return false
		}
	} else {
		// At least one cover must touch a tile
		// that is already on the board
		if numAdjacentTiles == 0 {
			return false
		}
	}
	if !move.ValidateWords {
		// No need to validate the words formed by this move on the board:
		// return true, we're done
		return true
	}
	if move.Word == IllegalMoveWord || move.Word == "" {
		return false
	}
	if !game.Dawg.Find(move.Word) {
		return false
	}
	// Check the cross words
	for coord, cover := range move.Covers {
		left, right := game.Board.CrossWords(coord.Col, coord.Row, !move.Horizontal)
		if len(left) > 0 || len(right) > 0 {
			// There is a cross word here: check it
			if !game.Dawg.Find(left + string(cover.Meaning) + right) {
				// Not found in the dictionary
				return false
			}
		}
	}
	return true
}

// Apply moves the tiles in the Covers from the player's Rack
// to the board Squares
func (move *TileMove) Apply(game *Game) bool {
	// The move is assumed to have already been validated via Move.IsValid()
	rack := &game.Racks[game.PlayerToMove()]
	for coord, cover := range move.Covers {
		// Find the tile in the player's rack
		tile := rack.FindTile(cover.Letter)
		if tile == nil {
			// Not found: abort
			return false
		}
		if cover.Letter == '?' {
			tile.Meaning = cover.Meaning
		} else {
			tile.Meaning = cover.Letter
		}
		if !game.PlayTile(tile, coord.Row, coord.Col) {
			// The tile was not found in the player's rack.
			// This is not good as the move may have been only partially applied.
			return false
		}
	}
	// Reset the counter of consecutive zero-point moves
	game.NumPassMoves = 0
	return true
}

// Score returns the score of the TileMove, if
// played in the given Game
func (move *TileMove) Score(state *GameState) int {
	if move.CachedScore != nil {
		return *move.CachedScore
	}
	// Cumulative letter score
	var score = 0
	// Cumulative cross scores
	var crossScore = 0
	// Word multiplier
	var multiplier = 1
	var rowIncr, colIncr = 0, 0
	var direction int
	if move.Horizontal {
		direction = LEFT
		colIncr = 1
	} else {
		direction = ABOVE
		rowIncr = 1
	}
	// Start with tiles above the top left
	row, col := move.TopLeft.Row, move.TopLeft.Col
	for _, tile := range state.Board.Fragment(row, col, direction) {
		score += tile.Score
	}
	// Then, progress from the top left to the bottom right
	for {
		sq := state.Board.Sq(row, col)
		if sq == nil {
			break
		}
		if cover, covered := move.Covers[Coordinate{row, col}]; covered {
			// This square is covered by the move: apply its letter
			// and word multipliers
			thisScore := state.TileSet.Scores[cover.Letter] * sq.LetterMultiplier
			score += thisScore
			multiplier *= sq.WordMultiplier
			// Add cross score, if any
			hasCrossing, csc := state.Board.CrossScore(row, col, !move.Horizontal)
			if hasCrossing {
				crossScore += (csc + thisScore) * sq.WordMultiplier
			}
		} else {
			// This square was already covered: add its letter score only
			score += sq.Tile.Score
		}
		if row >= move.BottomRight.Row && col >= move.BottomRight.Col {
			break
		}
		row += rowIncr
		col += colIncr
	}
	// Finally, add tiles below the bottom right
	row, col = move.BottomRight.Row, move.BottomRight.Col
	if move.Horizontal {
		direction = RIGHT
	} else {
		direction = BELOW
	}
	for _, tile := range state.Board.Fragment(row, col, direction) {
		score += tile.Score
	}
	// Multiply the accumulated letter score with the word multiplier
	score *= multiplier
	// Add cross scores
	score += crossScore
	if len(move.Covers) == RackSize {
		// The player played his entire rack: add the bingo bonus
		score += BingoBonus
	}
	// Only calculate the score once, then cache it
	move.CachedScore = &score
	return score
}

// NewPassMove returns a reference to a fresh PassMove
func NewPassMove() *PassMove {
	return &PassMove{}
}

// String return a string description of the PassMove
func (move *PassMove) String() string {
	return "Pass"
}

// IsValid always returns true for a PassMove
func (move *PassMove) IsValid(game *Game) bool {
	return true
}

// Apply always succeeds and returns true for a PassMove
func (move *PassMove) Apply(game *Game) bool {
	// Increment the number of consecutive zero-point moves
	game.NumPassMoves++
	return true
}

// Score is always 0 for a PassMove
func (move *PassMove) Score(state *GameState) int {
	return 0
}

// NewExchangeMove returns a reference to a fresh ExchangeMove
func NewExchangeMove(letters string) *ExchangeMove {
	return &ExchangeMove{Letters: letters}
}

// String return a string description of the ExchangeMove
func (move *ExchangeMove) String() string {
	return "Exch " + move.Letters
}

// IsValid returns true if an exchange is allowed and all
// exchanged tiles are actually in the player's rack
func (move *ExchangeMove) IsValid(game *Game) bool {
	if move == nil || game == nil {
		return false
	}
	if !game.Bag.ExchangeAllowed() {
		// Too few tiles left in the bag
		return false
	}
	runes := []rune(move.Letters)
	if len(runes) < 1 || len(runes) > RackSize {
		return false
	}
	rack := game.Racks[game.PlayerToMove()].AsString()
	for _, letter := range runes {
		if !strings.ContainsRune(rack, letter) {
			// This exchanged letter is not in the player's rack
			return false
		}
		rack = strings.Replace(rack, string(letter), "", 1)
	}
	// All exchanged letters found: the move is OK
	return true
}

// Apply replenishes the exchanged tiles in the Rack
// from the Bag
func (move *ExchangeMove) Apply(game *Game) bool {
	runes := []rune(move.Letters)
	rack := &game.Racks[game.PlayerToMove()]
	tiles := make([]*Tile, 0, RackSize)
	// First, remove the exchanged tiles from the player's Rack
	for _, letter := range runes {
		tile := rack.FindTile(letter)
		if tile == nil {
			// Should not happen!
			return false
		}
		if !rack.RemoveTile(tile) {
			// Should not happen!
			return false
		}
		tiles = append(tiles, tile)
	}
	// Replenish the Rack from the Bag...
	rack.Fill(game.Bag)
	// ...before returning the exchanged tiles to the Bag
	for _, tile := range tiles {
		game.Bag.ReturnTile(tile)
	}
	// Increment the number of consecutive zero-point moves
	game.NumPassMoves++
	return true
}

// Score is always 0 for an ExchangeMove
func (move *ExchangeMove) Score(state *GameState) int {
	return 0
}

// NewFinalMove returns a reference to a fresh FinalMove
func NewFinalMove(rackOpp string, multiplyFactor int) *FinalMove {
	return &FinalMove{OpponentRack: rackOpp, MultiplyFactor: multiplyFactor}
}

// String return a string description of the FinalMove
func (move *FinalMove) String() string {
	return "Rack " + move.OpponentRack
}

// IsValid always returns true for a FinalMove
func (move *FinalMove) IsValid(game *Game) bool {
	return true
}

// Apply always succeeds and returns true for a FinalMove
func (move *FinalMove) Apply(game *Game) bool {
	return true
}

// Score returns the opponent's rack leave, multiplied
// by a multiplication factor that can be 1 or 2
func (move *FinalMove) Score(state *GameState) int {
	var adj = 0
	for _, letter := range []rune(move.OpponentRack) {
		adj += state.TileSet.Scores[letter]
	}
	return adj * move.MultiplyFactor
}
