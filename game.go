// game.go
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.

// This file implements the Game and GameState classes

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
	MoveList    []*MoveItem
	// The DAWG dictionary to use in the game
	Dawg *Dawg
	// The tile set to use in the game
	TileSet *TileSet
	// The number of consecutive non-tile, zero-point moves
	// (when 6 consecutive such moves have been made,
	// the game is over)
	NumPassMoves int
	// Whether to validate words formed by tile moves in
	// the game
	ValidateWords bool
}

// GameState contains the bare minimum of information
// that is needed for a robot player to decide on a move
// in a Game.
type GameState struct {
	Dawg    *Dawg
	TileSet *TileSet
	Board   *Board
	// The rack of the player whose move it is
	Rack *Rack
	// If there are fewer than RackSize tiles in the bag,
	// an exchange move is not allowed
	exchangeForbidden bool
}

// MoveItem is an entry in the MoveList of a Game.
// It contains the player's Rack as it was before the move,
// as well as the move itself.
type MoveItem struct {
	RackBefore string
	Move       Move
}

// Init initializes a new game with a fresh bag copied
// from the given tile set, and draws the player racks
// from the bag
func (game *Game) Init(boardType string, tileSet *TileSet, dawg *Dawg) {
	game.Board.Init(boardType)
	game.Racks[0].Init()
	game.Racks[1].Init()
	game.TileSet = tileSet
	game.Bag = makeBag(tileSet)
	game.Racks[0].Fill(game.Bag)
	game.Racks[1].Fill(game.Bag)
	// Initial capacity for 30 moves
	game.MoveList = make([]*MoveItem, 0, 30)
	game.Dawg = dawg
	// By default, we validate words formed by tile moves
	game.ValidateWords = true
}

// NewIcelandicGame instantiates a new Game with the Icelandic TileSet
// and returns a reference to it
func NewIcelandicGame(boardType string) *Game {
	if IcelandicDictionary == nil {
		// Unable to read Icelandic DAWG
		return nil
	}
	game := &Game{}
	game.Init(boardType, NewIcelandicTileSet, IcelandicDictionary)
	return game
}

// NewOspsGame instantiates a new Game with the Polish TileSet
// and returns a reference to it
func NewOspsGame(boardType string) *Game {
	if OspsDictionary == nil {
		// Unable to read Polish (OSPS37) DAWG
		return nil
	}
	game := &Game{}
	game.Init(boardType, PolishTileSet, OspsDictionary)
	return game
}

// NewNorwegianBokmålGame instantiates a new Game with the
// Norwegian (Bokmål) TileSet and returns a reference to it
func NewNorwegianBokmålGame(boardType string) *Game {
	if NorwegianBokmålDictionary == nil {
		// Unable to read Norwegian (Bokmål) DAWG
		return nil
	}
	game := &Game{}
	game.Init(boardType, NorwegianTileSet, NorwegianBokmålDictionary)
	return game
}

// NewNorwegianNynorskGame instantiates a new Game with the
// Norwegian (Nynorsk) TileSet and returns a reference to it
func NewNorwegianNynorskGame(boardType string) *Game {
	if NorwegianNynorskDictionary == nil {
		// Unable to read Norwegian (Nynorsk) DAWG
		return nil
	}
	game := &Game{}
	game.Init(boardType, NorwegianTileSet, NorwegianNynorskDictionary)
	return game
}

// NewOtcwlGame instantiates a new Game with the
// English ('standard' board type) or New English ('explo' board type)
// TileSet, and returns a reference to it
func NewOtcwlGame(boardType string) *Game {
	if OtcwlDictionary == nil {
		// Unable to read OTCWL2014 DAWG
		return nil
	}
	game := &Game{}
	var tileSet *TileSet
	if boardType == "explo" {
		tileSet = NewEnglishTileSet
	} else {
		tileSet = EnglishTileSet
	}
	game.Init(boardType, tileSet, OtcwlDictionary)
	return game
}

// NewSowpodsGame instantiates a new Game with the
// English ('standard' board type) or New English ('explo' board type)
// TileSet, and returns a reference to it
func NewSowpodsGame(boardType string) *Game {
	if SowpodsDictionary == nil {
		// Unable to read SOWPODS DAWG
		return nil
	}
	game := &Game{}
	var tileSet *TileSet
	if boardType == "explo" {
		tileSet = NewEnglishTileSet
	} else {
		tileSet = EnglishTileSet
	}
	game.Init(boardType, tileSet, SowpodsDictionary)
	return game
}

func NewState(dawg *Dawg, tileSet *TileSet, board *Board, rack *Rack, exchangeForbidden bool) *GameState {
	return &GameState{
		Dawg:              dawg,
		TileSet:           tileSet,
		Board:             board,
		Rack:              rack,
		exchangeForbidden: exchangeForbidden,
	}
}

// State returns a new GameState instance describing the state of the
// game in a minimal manner so that a robot player can decide on a move
func (game *Game) State() *GameState {
	player := game.PlayerToMove()
	exchangeForbidden := game.Bag.TileCount() < RackSize
	return NewState(
		game.Dawg,
		game.TileSet,
		&game.Board,
		&game.Racks[player],
		exchangeForbidden,
	)
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
	if sq := game.Board.Sq(row, col); sq == nil || sq.Tile != nil {
		// No such square or already occupied
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
	return game.Board.PlaceTile(row, col, tile)
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

// ForceRack forces the rack of the given player
// (or -1 = player whose move it is)
// to contain the given tiles. The tiles that were previously
// in the rack are first returned to the bag, and then the specified new
// tiles are drawn. If any of them is not found in the bag, the
// operation is aborted and false is returned.
func (game *Game) ForceRack(player int, letters string) bool {
	if player < 0 || player > 1 {
		player = game.PlayerToMove()
	}
	rack := &game.Racks[player]
	// Return all tiles from the Rack to the Bag
	for i := 0; i < RackSize; i++ {
		if tile := rack.Slots[i].Tile; tile != nil {
			if !rack.RemoveTile(tile) {
				// Weird, should not happen
				return false
			}
			game.Bag.ReturnTile(tile)
		}
	}
	// And re-fill the Rack with the given letters
	return rack.FillByLetters(game.Bag, []rune(letters))
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
		covers[Coordinate{row, col}] = Cover{tile.Letter, tile.Meaning}
		row += rowInc
		col += colInc
	}
	// Apply a fresh TileMove to the game
	var move Move
	if game.ValidateWords {
		move = NewTileMove(&game.Board, covers)
	} else {
		move = NewUncheckedTileMove(&game.Board, covers)
	}
	return game.Apply(move)
}

// ApplyValid applies an already validated Move to a Game,
// appends it to the move list, replenishes the player's Rack
// if needed, and updates scores.
func (game *Game) ApplyValid(move Move) bool {
	// Be careful to call PlayerToMove() before appending
	// a move to the move list (this reverses the players)
	if game == nil || move == nil {
		// Should not happen
		return false
	}
	playerToMove := game.PlayerToMove()
	rack := &game.Racks[playerToMove]
	rackBefore := rack.AsString()
	if !move.Apply(game) {
		// Not valid! Should not happen...
		return false
	}
	// Update the scores and append to the move list
	game.acceptMove(rackBefore, move)
	// Replenish the player's rack, as needed
	rack.Fill(game.Bag)
	if game.IsOver() {
		// The game is now over: add the FinalMoves
		rackThis := game.Racks[playerToMove].AsString()
		rackOpp := game.Racks[1-playerToMove].AsString()
		var multiplyFactor = 2
		if len(rackThis) > 0 {
			// The game is not finishing by the final player
			// completing his rack: both players then get the
			// opponent's remaining tile scores
			multiplyFactor = 1
		}
		// Add a final move for the opponent
		// (which in most cases yields zero points, since
		// the finishing player has no tiles left)
		finalOpp := NewFinalMove(rackThis, multiplyFactor)
		game.acceptMove(rackOpp, finalOpp)
		// Add a final move for the finishing player
		// (which in most cases yields double the tile scores
		// of the opponent's rack)
		finalThis := NewFinalMove(rackOpp, multiplyFactor)
		game.acceptMove(rackThis, finalThis)
	}
	return true
}

// acceptMove updates the scores and appends a given Move
// to the Game's MoveList
func (game *Game) acceptMove(rackBefore string, move Move) {
	// Calculate the score
	if game == nil || move == nil {
		return
	}
	score := move.Score(game.State())
	// Update the player's score
	game.Scores[game.PlayerToMove()] += score
	// Append to the move list
	moveItem := &MoveItem{RackBefore: rackBefore, Move: move}
	game.MoveList = append(game.MoveList, moveItem)
}

// Apply applies a move to the game, after validating it
func (game *Game) Apply(move Move) bool {
	if game == nil || move == nil {
		return false
	}
	if !move.IsValid(game) {
		// Not valid!
		return false
	}
	return game.ApplyValid(move)
}

// IsOver returns true if the Game is over after the last
// move played
func (game *Game) IsOver() bool {
	ix := len(game.MoveList)
	if ix == 0 {
		// No moves yet: cannot be over
		return false
	}
	// TODO: Check for resignation
	if game.NumPassMoves == 6 {
		// Six consecutive zero-point moves
		// (e.g. three rounds of passes) finish the game
		return true
	}
	lastPlayer := 1 - (ix % 2)
	// The game is over if the last player's move emptied her rack
	return game.Racks[lastPlayer].IsEmpty()
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
		state := game.State()
		sb.WriteString("Moves:\n")
		for i, item := range game.MoveList {
			m := item.Move
			if i%2 == 0 {
				// Left side player
				sb.WriteString(fmt.Sprintf("  %2d: (%v) %v", (i/2)+1, m.Score(state), m))
			} else {
				// Right side player
				sb.WriteString(fmt.Sprintf(" / %v (%v)\n", m, m.Score(state)))
			}
		}
		if len(game.MoveList)%2 == 1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
