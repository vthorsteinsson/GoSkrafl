// game.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file implements the Game class

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

// Game is a container for an in-progress game between
// two players, having a Board and two Racks, as well
// as a Bag and a list of Moves made so far. We also keep
// track of the number of Tiles that have been placed on
// the Board.
type Game struct {
	PlayerNames [2]string
	Scores      [2]int
	Board       Board
	Racks       [2]Rack
	Bag         *Bag
	MoveList    []Move
	// The DAWG dictionary to use in the game
	Dawg *Dawg
}

// GameState contains the bare minimum of information
// that is needed for a robot player to decide on a move
// in a Game.
type GameState struct {
	Dawg  *Dawg
	Board *Board
	Rack  *Rack // The rack of the player whose move it is
}

// Init initializes a new game with a fresh bag copied
// from the given tile set, and draws the player racks
// from the bag
func (game *Game) Init(tileSet *TileSet, dawg *Dawg) {
	game.Board.Init()
	game.Racks[0].Init()
	game.Racks[1].Init()
	game.Bag = makeBag(tileSet)
	game.Racks[0].Fill(game.Bag)
	game.Racks[1].Fill(game.Bag)
	game.MoveList = make([]Move, 0, 30) // Initial capacity for 30 moves
	game.Dawg = dawg
}

// NewIcelandicGame instantiates a new Game with the Icelandic TileSet
// and returns a reference to it
func NewIcelandicGame() *Game {
	game := &Game{}
	game.Init(NewIcelandicTileSet, IcelandicDictionary)
	return game
}

// State returns a new GameState instance describing the state of the
// game in a minimal manner so that a robot player can decide on a move
func (game *Game) State() *GameState {
	player := game.PlayerToMove()
	return &GameState{game.Dawg, &game.Board, &game.Racks[player]}
}

// TileAt is a convenience function for returning the Tile at
// a given coordinate on the Game Board
func (game *Game) TileAt(row, col int) *Tile {
	sq := game.Board.Sq(row, col)
	if sq == nil {
		return nil
	}
	return sq.Tile
}

// PlayTile moves a tile from the player's rack to the board
func (game *Game) PlayTile(tile *Tile, row, col int) bool {
	sq := game.Board.Sq(row, col)
	if sq == nil {
		// No such square
		return false
	}
	if sq.Tile != nil {
		// We already have a tile in this location
		return false
	}
	playerToMove := game.PlayerToMove()
	if !game.Racks[playerToMove].RemoveTile(tile) {
		// This tile isn't in the rack
		return false
	}
	if tile.Meaning == '?' {
		// Tile must have an associated meaning when played
		return false
	}
	if tile.Letter != '?' {
		tile.Meaning = tile.Letter
	}
	tile.PlayedBy = playerToMove
	sq.Tile = tile
	game.Board.NumTiles++
	return true
}

// TilesOnBoard returns the number of tiles already laid down
// on the board
func (game *Game) TilesOnBoard() int {
	return game.Board.NumTiles
}

// SetPlayerNames sets the names of the two players
func (game *Game) SetPlayerNames(player0, player1 string) {
	game.PlayerNames[0] = player0
	game.PlayerNames[1] = player1
}

// PlayerToMove returns 0 or 1 depending on which player's move it is
func (game *Game) PlayerToMove() int {
	return len(game.MoveList) % 2
}

// MakePassMove appends a pass move to the Game's move list
func (game *Game) MakePassMove() bool {
	return game.Apply(NewPassMove())
}

// MakeTileMove creates a tile move and appends it to the Game's move list
func (game *Game) MakeTileMove(row, col int, horizontal bool, tiles []*Tile) bool {
	// Basic sanity checks
	if row < 0 || row >= BoardSize || col < 0 || col >= BoardSize ||
		len(tiles) < 1 || len(tiles) > RackSize {
		return false
	}
	// Check that the played tiles are actually in the player's rack
	rack := &game.Racks[game.PlayerToMove()]
	for _, tile := range tiles {
		if !rack.HasTile(tile) {
			// This tile isn't in the player's rack
			return false
		}
	}
	// A tile move must start at an empty square
	if game.TileAt(row, col) != nil {
		return false
	}
	var rowInc, colInc int
	if horizontal {
		colInc = 1
	} else {
		rowInc = 1
	}
	covers := make(Covers)
	for _, tile := range tiles {
		if row >= BoardSize || col >= BoardSize {
			// Gone off the board
			return false
		}
		for game.TileAt(row, col) != nil {
			// Occupied square: try the next one
			row += rowInc
			col += colInc
			if row >= BoardSize || col >= BoardSize {
				// Gone off the edge of the board
				return false
			}
		}
		covers[Coordinate{row, col}] = tile
		row += rowInc
		col += colInc
	}
	// Apply a fresh TileMove to the game
	return game.Apply(NewTileMove(game, covers))
}

// Apply applies a move to the game, appends it to the
// move list, replenishes the player's rack if needed,
// and updates scores.
func (game *Game) Apply(move Move) bool {
	if !move.IsValid(game) || !move.Apply(game) {
		// Not valid!
		return false
	}
	// Valid: calculate the score
	score := move.Score(game)
	// Be careful to call PlayerToMove() before appending
	// a move to the move list (this reverses the players)
	playerToMove := game.PlayerToMove()
	// Append to move list
	game.MoveList = append(game.MoveList, move)
	// Replenish the player's rack, as needed
	game.Racks[playerToMove].Fill(game.Bag)
	// Update the player's score
	game.Scores[playerToMove] += score
	return true
}

// String returns a string representation of a Game
func (game *Game) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v (%v : %v) %v\n",
		game.PlayerNames[0],
		game.Scores[0],
		game.Scores[1],
		game.PlayerNames[1],
	))
	sb.WriteString(fmt.Sprintf("%v\n", &game.Board))
	sb.WriteString(fmt.Sprintf("Rack 0: %v\n", &game.Racks[0]))
	sb.WriteString(fmt.Sprintf("Rack 1: %v\n", &game.Racks[1]))
	sb.WriteString(fmt.Sprintf("Bag: %v\n", game.Bag))
	// Show the move list, if present
	if len(game.MoveList) > 0 {
		sb.WriteString("Moves:\n")
		for i, m := range game.MoveList {
			if i%2 == 0 {
				// Left side player
				sb.WriteString(fmt.Sprintf("  %2d: (%v) %v", (i/2)+1, m.Score(game), m))
			} else {
				// Right side player
				sb.WriteString(fmt.Sprintf(" / %v (%v)\n", m, m.Score(game)))
			}
		}
		if len(game.MoveList)%2 == 1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
