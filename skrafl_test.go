// skrafl_test.go
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.
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

func TestIcelandicDawg(t *testing.T) {
	// Test finding words in the DAWG
	wordBase := IcelandicDictionary
	positiveCases := []string{
		"góðan", "daginn", "hér", "er", "prófun", "orðum", "ti", "do", "álínis",
		"bríostur", "feik", "frisbí", "umr", "hæi",
	}
	negativeCases := []string{
		"blex", "fauð", "á", "é", "this", "ösi",
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

func TestNorwegianDawg(t *testing.T) {
	// Test finding words in the DAWG
	wordBase := NorwegianBokmålDictionary
	positiveCases := []string{
		"god", "dag", "her", "er", "prøve", "ord", "ti", "do", "alene",
		"gründer",
	}
	negativeCases := []string{
		"blex", "fåser", "c", "abcd", "this", "korleis", "kvifor", "ikkje",
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
	results := wordBase.Permute("børnene", 6)
	if !compareResults(results, []string{
		"brenne", "brønne", "bønner", "børene", "enøren", "nørene", "øreben", "ørnene",
	}) {
		t.Errorf("Permute() returns incorrect result: %v", results)
	}
	results = wordBase.Permute("lei?der", RackSize)
	if !compareResults(results, []string{
		"blidere", "defiler", "deilder", "deleier", "deliren",
		"delirer", "deliret", "depiler", "desiler", "diltere",
		"dveiler", "elidere", "elidert", "firdele", "firedel",
		"gildere", "glidere", "idealer", "idelære", "ilderen",
		"ilderne", "ildeter", "ildrene", "leidere", "leirdue",
		"leirdye", "leirede", "leivder", "lesider", "liender",
		"lirende", "lirkede", "lydiere", "midlere", "mildere",
		"nideler", "pilrede", "redelig", "redline", "riflede",
		"rillede", "seidler", "sleider", "tideler", "tilrede",
	}) {
		t.Errorf("Permute() returns incorrect result: %v", results)
	}
	// Test pattern matching
	results = wordBase.Match("mo?ile?")
	if !compareResults(results, []string{
		"mobilen", "mobiler",
	}) {
		t.Errorf("Match() returns incorrect result: %v", results)
	}
}

func TestNynorskDawg(t *testing.T) {
	// Test finding words in the DAWG
	wordBase := NorwegianNynorskDictionary
	positiveCases := []string{
		"god", "dag", "her", "er", "prøve", "ord", "ti", "do", "aleine",
		"gründer", "ikkje", "berre", "først", "sist", "såleis", "korleis",
		"kvifor",
	}
	negativeCases := []string{
		// Include words that are in Bokmål but not in Nynorsk
		"blex", "fåser", "c", "abcd", "this", "hvordan", "hvorfor",
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
	results := wordBase.Permute("børnene", 6)
	if !compareResults(results, []string{
		"brenne", "bønene", "bønner", "børene", "ørnene",
	}) {
		t.Errorf("Permute() returns incorrect result: %v", results)
	}
	results = wordBase.Permute("lei?der", RackSize)
	if !compareResults(results, []string{
		"defiler", "deigler", "deilder", "delirer", "dreiela", "firedel",
		"idelære", "ikleder", "ilderen", "leirdue", "leivder", "lesider",
		"ordleie", "redline",
	}) {
		t.Errorf("Permute() returns incorrect result: %v", results)
	}
	// Test pattern matching
	results = wordBase.Match("?ske?")
	if !compareResults(results, []string{
		"asken", "asket", "esker",
	}) {
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
	game := NewIcelandicGame("standard")
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
		return
	}
	// For testing, disable word validation for tile moves
	game.ValidateWords = false
	game.SetPlayerNames("Villi", "Gopher")
	if game.IsOver() {
		t.Errorf("Game can't be over before it starts")
	}
	// Construct a move from the player 0 rack
	move := game.Racks[0].Extract(4, 'x')
	if game.MakeTileMove(3, 3, false, move) {
		t.Errorf("First move must go through H8")
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
	tileMove = NewUncheckedTileMove(board,
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
	tileMove = NewUncheckedTileMove(board,
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
	tileMove = NewUncheckedTileMove(board,
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
	state := game.State()
	exchangeMove := NewExchangeMove(state.Rack.AsString())
	if !exchangeMove.IsValid(game) {
		t.Errorf("ExchangeMove is incorrectly seen as not valid")
	}
	if !game.ApplyValid(exchangeMove) {
		t.Errorf("Unable to apply valid ExchangeMove")
	}
	exchangeMove = NewExchangeMove("")
	if exchangeMove.IsValid(game) {
		t.Errorf("ExchangeMove is incorrectly seen as valid")
	}
	exchangeMove = NewExchangeMove("czbleh")
	if exchangeMove.IsValid(game) {
		t.Errorf("ExchangeMove is incorrectly seen as valid")
	}
	exchangeMove = NewExchangeMove(state.Rack.AsString() + state.Rack.AsString())
	if exchangeMove.IsValid(game) {
		t.Errorf("ExchangeMove is incorrectly seen as valid")
	}
}

func TestStartSquare(t *testing.T) {
	game := NewIcelandicGame("explo")
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
		return
	}
	// For testing, disable word validation for tile moves
	game.ValidateWords = false
	game.SetPlayerNames("Villi", "Gopher")
	if game.IsOver() {
		t.Errorf("Game can't be over before it starts")
	}
	// Construct a move from the player 0 rack
	move := game.Racks[0].Extract(4, 'x')
	// Attempt to make a move that starts at H8
	if game.MakeTileMove(7, 7, false, move) {
		t.Errorf("First move must go through D4 square")
	}
	// Check number of tiles now on the Board
	if game.TilesOnBoard() != 0 {
		t.Errorf("Board should have 0 tiles after erroneous move")
	}
	// Make a move that starts at D4
	if !game.MakeTileMove(3, 3, false, move) {
		t.Errorf("First move through D4 rejected")
	}
	// Check number of tiles now on the Board
	if game.TilesOnBoard() != 4 {
		t.Errorf("Board should have 4 tiles after valid move")
	}
}

func TestWordCheck(t *testing.T) {
	game := NewIcelandicGame("standard")
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
		return
	}
	if !game.ValidateWords {
		t.Errorf("Game should validate words in tile moves by default")
	}
	makeMove := func(rackLetters string, word string, row, col int, horizontal bool) bool {
		player := game.PlayerToMove()
		rack := &game.Racks[1-player]
		rack.ReturnToBag(game.Bag)
		rack = &game.Racks[player]
		rack.ReturnToBag(game.Bag)
		if ok := rack.FillByLetters(game.Bag, []rune(rackLetters)); !ok {
			t.Errorf("Unable to draw specific letters from bag")
		}
		tiles := rack.FindTiles([]rune(word))
		return game.MakeTileMove(row, col, horizontal, tiles)
	}
	if !makeMove("prófaðu", "prófaðu", 5, 7, false) {
		t.Errorf("Valid tile move rejected")
	}
	if !makeMove("akurhái", "akur", 10, 6, false) {
		t.Errorf("Valid tile move rejected")
	}
	if !makeMove("hálarsx", "al", 10, 8, false) {
		t.Errorf("Valid tile move rejected")
	}
	if makeMove("nálarsx", "ns", 10, 9, false) {
		t.Errorf("Invalid tile move accepted")
	}
}

func TestFindLeftParts(t *testing.T) {
	// Find left parts
	game := NewIcelandicGame("standard")
	if game == nil {
		t.Errorf("Unable to create a new Icelandic game")
		return
	}
	rack := game.Racks[game.PlayerToMove()].AsRunes()
	leftParts := FindLeftParts(game.Dawg, rack)
	for lenParts, lp := range leftParts {
		for _, part := range lp {
			if len(part.matched) != lenParts+1 {
				t.Errorf("Unexpected length of left part %v", string(part.matched))
			}
			if len(part.rack) != RackSize-(lenParts+1) {
				t.Errorf("Unexpected length of rack %v", string(rack))
			}
			tempRack := []rune(rack)
			for _, r := range part.matched {
				if ContainsRune(tempRack, r) {
					tempRack = RemoveRune(tempRack, r)
				} else {
					if ContainsRune(tempRack, '?') {
						tempRack = RemoveRune(tempRack, '?')
					} else {
						t.Errorf("Left prefix contains a letter that is not in the rack")
					}
				}
			}
		}
	}
}

func TestAxis(t *testing.T) {

	type TilePlacement struct {
		Row  int
		Col  int
		Tile *Tile
	}

	// Test the move generation on a single Axis instance
	board := NewBoard("explo")
	for _, tp := range []TilePlacement{
		{3, 3, &Tile{'d', 'd', 3, 0}},
		{3, 4, &Tile{'o', 'o', 4, 0}},
	} {
		board.PlaceTile(tp.Row, tp.Col, tp.Tile)
	}
	if board.NumTiles != 2 {
		t.Errorf("Board should have 2 tiles")
	}
	if !board.HasStartTile() {
		t.Errorf("Board should have start tile")
	}
	rack := NewRack([]rune("prófaðu"), NewIcelandicTileSet)
	rackRunes := rack.AsRunes()
	state := NewState(
		IcelandicDictionary,
		NewIcelandicTileSet,
		board,
		rack,
		false,
	)
	rackSet := state.Dawg.alphabet.MakeSet(rackRunes)
	leftParts := FindLeftParts(state.Dawg, rackRunes)
	var axis Axis
	// Horizontal axis representing row 4 (E)
	axis.Init(state, rackSet, 4, true)
	moves := axis.GenerateMoves(leftParts)
	if len(moves) == 0 {
		t.Errorf("No moves generated")
	}
	// Validate that all words formed are found in the dictionary
	for _, move := range moves {
		if wordMove, ok := move.(Validatable); ok {
			if !wordMove.ValidateWord(IcelandicDictionary) {
				t.Errorf(
					"Invalid word '%v' generated",
					wordMove.CleanWord(),
				)
			}
		}
	}
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
	var game *Game
	for i := 0; ; i++ {
		game = NewIcelandicGame("standard")
		if game == nil {
			t.Errorf("Unable to create a new Icelandic game")
			return
		}
		// Forcing a rack may fail because some of the unique tiles may
		// be in the opponent's rack. In that case, we just try again.
		if game.ForceRack(0, "villiþo") {
			// Success: continue
			break
		}
		if i > 20 {
			// Something is very likely wrong
			t.Errorf("Unable to force the rack after 20 attempts")
			return
		}
	}
	rack := &game.Racks[0]
	tiles := rack.FindTiles([]rune("vill"))
	if !game.MakeTileMove(4, 7, false, tiles) {
		t.Errorf("Unable to make tile move")
	}
	if !game.ForceRack(1, "rsteins") {
		t.Errorf("Unable to force the rack")
	}
	rack = &game.Racks[1]
	tiles = rack.FindTiles([]rune("stein"))
	if !game.MakeTileMove(8, 6, true, tiles) {
		t.Errorf("Unable to make tile move")
	}
	_ = game.String()
}

func BenchmarkRobot(b *testing.B) {

	// Try to make the benchmark as deterministic as possible
	// rand.Seed(31743) // Commodore Basic 4.0 / 31743 bytes free / ready.

	// Generate a sequence of moves and responses
	simulateGame := func(robot *RobotWrapper) {
		game := NewIcelandicGame("standard")
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
	runTest := func(boardType string, ctor func(boardType string) *Game) {
		robot := NewHighScoreRobot()
		if robot == nil {
			t.Errorf("Unable to create HighScoreRobot")
		}
		game := ctor(boardType)
		if game == nil {
			t.Errorf("Unable to create a new game for board type '%s'", boardType)
		}
		game.SetPlayerNames("Villi", "Gopher")
		// Go through an entire game
		i := 0
		for {
			state := game.State()
			if state == nil {
				t.Errorf("Unexpected nil game state")
			}
			move := robot.GenerateMove(state)
			if move == nil || !move.IsValid(game) {
				t.Errorf("Invalid move generated")
			} else {
				if !game.ApplyValid(move) {
					t.Errorf("Move not valid when applied")
				}
				i++
				if game.IsOver() {
					// Should add two moves to the move list
					i += 2
					break
				}
			}
			if i >= 50 {
				t.Errorf("Game appears not to terminate")
				break
			}
		}
		if len(game.MoveList) != i {
			t.Errorf("Incorrect number of moves recorded")
		}
	}
	// Cycle through 5 rounds of 2 (board types) x 5 (dictionaries)
	// simulated games, each with its own board type, Dawg and alphabet
	for cycle := 0; cycle < 5; cycle++ {
		for _, boardType := range []string{"standard", "explo"} {
			runTest(boardType, NewIcelandicGame)
			runTest(boardType, NewOtcwlGame)
			runTest(boardType, NewSowpodsGame)
			runTest(boardType, NewOspsGame)
			runTest(boardType, NewNorwegianBokmålGame)
		}
	}
}
