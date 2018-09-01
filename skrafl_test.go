// skrafl_test.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file contains tests for the skrafl package

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
	"testing"
)

func TestDawg(t *testing.T) {
	// Test finding words in the DAWG
	wordBase := IcelandicDictionary
	positiveCases := []string{
		"góðan", "daginn", "hér", "er", "prófun", "orðum", "ti", "do", "álínis",
	}
	negativeCases := []string{
		"blex", "fauð", "á", "é", "this",
	}
	for _, word := range positiveCases {
		if !wordBase.Find(word) {
			t.Errorf("Did not find word '%v' that should be in the DAWG", word)
		}
	}
	for _, word := range negativeCases {
		if wordBase.Find(word) {
			t.Errorf("Found word '%v' that should not be in the DAWG", word)
		}
	}
	compareResults := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i, s := range a {
			if s != b[i] {
				return false
			}
		}
		return true
	}
	// Test word permutations
	results := wordBase.Permute("stálins", RackSize)
	if !compareResults(results, []string{"látsins", "tálsins"}) {
		t.Errorf("Permute() returns incorrect result: %v", results)
	}
	results = wordBase.Permute("böl?nna", RackSize)
	if !compareResults(results, []string{
		"bannböl", "bannlög", "bölanna", "böltann", "lögbann"}) {
		t.Errorf("Permute() returns incorrect result: %v", results)
	}
	// Test pattern matching
	results = wordBase.Match("fa?gin?")
	if !compareResults(results, []string{
		"fagginn", "fanginn", "fangins", "fanginu", "farginu"}) {
		t.Errorf("Match() returns incorrect result: %v", results)
	}
}

func BenchmarkDawg(b *testing.B) {
	// Define the permuter goroutine
	wordBase := IcelandicDictionary
	permuter := func(word string, ch chan int) {
		cnt := 0
		sumLength := 0
		for _, w := range wordBase.Permute(word, RackSize) {
			cnt++
			sumLength += len(w) // Use w
		}
		// Send the results back on this permuter's channel
		ch <- cnt
		ch <- sumLength
	}
	// We will permute four racks in each benchmark loop
	// iteration, using four parallel goroutines
	racks := []string{"?an?ins", "un?ansk", "gle?ina", "von??ði"}
	// Make the channels, one for each rack
	ch := make([]chan int, len(racks))
	for j := 0; j < len(ch); j++ {
		ch[j] = make(chan int)
	}
	// Now run the benchmark proper
	for i := 0; i < b.N; i++ {
		// Kick off the parallel permuters
		for j, rack := range racks {
			go permuter(rack, ch[j])
		}
		// Collect the results as they come back
		var cnt, sumLength int
		for _, c := range ch {
			cnt += <-c
			sumLength += <-c
		}
	}
}

func TestTileMove(t *testing.T) {
	game := NewIcelandicGame()
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
	}
	game.SetPlayerNames("Villi", "Gopher")
	// Construct a move from the player 0 rack
	move := game.Racks[0].Extract(4, 'x')
	if game.MakeTileMove(2, 2, false, move) {
		t.Errorf("First move must go through center")
	}
	// Check number of tiles now on the Board
	if game.TilesOnBoard() != 0 {
		t.Errorf("Board should have 0 tiles after erroneous move")
	}
	if game.PlayerToMove() != 0 {
		t.Errorf("PlayerToMove should still be 0 after erroneous move")
	}
	// Make a legal move, starting at row 4, column 7 (0-based),
	// vertical
	if !game.MakeTileMove(4, 7, false, move) {
		t.Errorf("Legal initial move rejected")
	}
	// Check number of tiles now on the Board
	if game.TilesOnBoard() != 4 {
		t.Errorf("Board should have 4 tiles after correct move")
	}
	// Check number of tiles left in the bag
	if game.Bag.TileCount() != 100-7-7-4 {
		t.Errorf("Bag should have 86 tiles after 4 tiles have been laid down")
	}
	if game.PlayerToMove() != 1 {
		t.Errorf("PlayerToMove should be 1 after correct move")
	}
	move = game.Racks[1].Extract(4, 'y')
	// Attempt to make a disconnected move
	if game.MakeTileMove(2, 2, false, move) {
		t.Errorf("Disconnected move erroneously returns true")
	}
	// Attempt to make a move that runs off the board
	if game.MakeTileMove(12, 2, false, move) {
		t.Errorf("Move that runs off the bottom of the board erroneously returns true")
	}
	// Attempt to make a move that runs off the board
	if game.MakeTileMove(2, 12, true, move) {
		t.Errorf("Move that runs off the right edge of the board erroneously returns true")
	}
	// Attempt to make a move that starts at an occupied square
	if game.MakeTileMove(7, 7, true, move) {
		t.Errorf("Move that starts at an occupied square erroneously returns true")
	}
	// Do a legal cross move
	if !game.MakeTileMove(7, 5, true, move) {
		t.Errorf("Legal cross move returns false")
	}
	// Check number of tiles left in the bag
	if game.Bag.TileCount() != 100-7-7-4-4 {
		t.Errorf("Bag should have 82 tiles after 2 * 4 tiles have been laid down")
	}
	if game.PlayerToMove() != 0 {
		t.Errorf("PlayerToMove should be 0 after correct move")
	}
	// Make a pass move for player 0
	if !game.MakePassMove() {
		t.Errorf("MakePassMove returns false")
	}
	if game.PlayerToMove() != 1 {
		t.Errorf("PlayerToMove should be 1 after pass move")
	}
	// Check number of tiles left in the bag
	if game.Bag.TileCount() != 100-7-7-4-4 {
		t.Errorf("Bag should still have 82 tiles after pass move")
	}
	// Check a few hand-crafted, buggy TileMoves
	// First, a disconnected single tile
	grabTile := func(player int, slot int) *Tile {
		tile := game.Racks[player].Slots[slot].Tile
		if tile.Letter == '?' {
			tile.Meaning = 'x'
		}
		return tile
	}
	tile := grabTile(1, 0)
	board := &game.Board
	tileMove := NewTileMove(board,
		Covers{
			{10, 8}: Cover{tile.Letter, tile.Meaning},
		},
	)
	if game.Apply(tileMove) {
		t.Errorf("Accepted disconnected single-tile move")
	}
	// Make a non-contiguous move
	tile2 := grabTile(1, 1)
	tileMove = NewTileMove(board,
		Covers{
			{10, 8}: Cover{tile.Letter, tile.Meaning},
			{12, 8}: Cover{tile2.Letter, tile2.Meaning},
		},
	)
	if game.Apply(tileMove) {
		t.Errorf("Accepted noncontiguous move")
	}
	// Make a non-linear move
	tileMove = NewTileMove(board,
		Covers{
			{5, 6}: Cover{tile.Letter, tile.Meaning},
			{6, 8}: Cover{tile2.Letter, tile2.Meaning},
		},
	)
	if game.Apply(tileMove) {
		t.Errorf("Accepted nonlinear move")
	}
	// Cover an already occupied square
	tileMove = NewTileMove(board,
		Covers{
			{5, 6}: Cover{tile.Letter, tile.Meaning},
			{5, 7}: Cover{tile2.Letter, tile2.Meaning},
		},
	)
	if game.Apply(tileMove) {
		t.Errorf("Accepted cover of already occupied square")
	}
	// Empty move
	tileMove = &TileMove{}
	if game.Apply(tileMove) {
		t.Errorf("Accepted empty move")
	}
	tileMove = NewTileMove(board, Covers{})
	if game.Apply(tileMove) {
		t.Errorf("Accepted empty move")
	}
	// Cover a nonexistent square
	tileMove = NewTileMove(board,
		Covers{
			{-1, 6}: Cover{tile.Letter, tile.Meaning},
			{0, 6}:  Cover{tile2.Letter, tile2.Meaning},
		},
	)
	if game.Apply(tileMove) {
		t.Errorf("Accepted cover of nonexistent square")
	}
	// Cover a nonexistent square
	tileMove = NewTileMove(board,
		Covers{
			{BoardSize - 1, 6}: Cover{tile.Letter, tile.Meaning},
			{BoardSize, 6}:     Cover{tile2.Letter, tile2.Meaning},
		},
	)
	if game.Apply(tileMove) {
		t.Errorf("Accepted cover of nonexistent square")
	}
	// Horizontal move
	tileMove = NewTileMove(board,
		Covers{
			{7, 4}:  Cover{tile.Letter, tile.Meaning},
			{7, 10}: Cover{tile2.Letter, tile2.Meaning},
		},
	)
	// t.Logf("%v\n", &game)
	if !tileMove.IsValid(game) {
		t.Errorf("Move is incorrectly seen as not valid")
	}
	if !tileMove.Horizontal {
		t.Errorf("Move is incorrectly identified as being vertical")
	}
	// Vertical move
	tileMove = NewTileMove(board,
		Covers{
			{7, 4}: Cover{tile.Letter, tile.Meaning},
			{8, 4}: Cover{tile2.Letter, tile2.Meaning},
		},
	)
	if !tileMove.IsValid(game) {
		t.Errorf("Move is incorrectly seen as not valid")
	}
	if tileMove.Horizontal {
		t.Errorf("Move is incorrectly identified as being horizontal")
	}
	// Single cover which creates a vertical move
	tileMove = NewTileMove(board,
		Covers{
			{8, 7}: Cover{tile.Letter, tile.Meaning},
		},
	)
	if !tileMove.IsValid(game) {
		t.Errorf("Move is incorrectly seen as not valid")
	}
	if tileMove.Horizontal {
		t.Errorf("Move is incorrectly identified as being horizontal")
	}
}

func TestFindLeftParts(t *testing.T) {
	// Find left parts
	game := NewIcelandicGame()
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
	}
	leftParts := FindLeftParts(game.Dawg, game.Racks[game.PlayerToMove()].AsString())
	_ = leftParts
}

func TestBitMaps(t *testing.T) {
	// Test bit-mapped sets of runes. Only runes that are already in the alphabet
	// can occur in a bit-mapped set.
	alphabet := IcelandicDictionary.alphabet
	set := alphabet.MakeSet([]rune{'á', 'l', 'a', 'f', 'o', 's', 's'})
	if !alphabet.Member('á', set) {
		t.Errorf("Rune 'á' should be member of set")
	}
	if !alphabet.Member('s', set) {
		t.Errorf("Rune 's' should be member of set")
	}
	if alphabet.Member('j', set) {
		t.Errorf("Rune 'j' should not be member of set")
	}
	if alphabet.Member('c', set) {
		t.Errorf("Rune 'c' should not be member of set")
	}
	// Test smiley face
	if alphabet.Member('😄', set) {
		t.Errorf("Rune '😄' should not be member of set")
	}
}

func TestStringify(t *testing.T) {
	// Stringify the game (no test but at least this enhances coverage)
	game := NewIcelandicGame()
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
	}
	_ = game.String()
}

func BenchmarkRobot(b *testing.B) {

	// Generate a sequence of moves and responses
	simulateGame := func(robot *RobotWrapper) {
		game := NewIcelandicGame()
		game.SetPlayerNames("Villi", "Gopher")
		for {
			state := game.State()
			move := robot.GenerateMove(state)
			game.ApplyValid(move)
			if game.IsOver() {
				break
			}
		}
	}

	robot := NewHighScoreRobot()
	for i := 0; i < b.N; i++ {
		simulateGame(robot)
	}
}

func TestRobot(t *testing.T) {
	robot := NewHighScoreRobot()
	if robot == nil {
		t.Errorf("Unable to create HighScoreRobot")
	}
	game := NewIcelandicGame()
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
	}
	game.SetPlayerNames("Villi", "Gopher")
	// Go through 8 moves (4 per player)
	const numMoves = 8
	for i := 0; i < numMoves; i++ {
		state := game.State()
		move := robot.GenerateMove(state)
		if move == nil || !move.IsValid(game) {
			t.Errorf("Invalid move generated")
		} else {
			game.ApplyValid(move)
		}
	}
	if len(game.MoveList) != numMoves {
		t.Errorf("Incorrect number of moves recorded")
	}
}
