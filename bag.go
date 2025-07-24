// bag.go
//
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.
//
// This file contains the Bag and TileSet logic

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
	"math/rand"
	"strings"
)

// Bag is a randomized list of tiles, initialized from a tile
// set, that is yet to be drawn and used in a game
type Bag struct {
	// Tiles is a fixed array of all tiles in a game,
	// copied at the start of the game from a TileSet
	Tiles []Tile
	// Contents is a list of pointers into the Tiles array,
	// corresponding to the current contents of the bag
	Contents []*Tile
}

// TileSet is a static list of tiles, used as a prototype
// to copy new Bags from
type TileSet struct {
	Tiles  []Tile
	Scores map[rune]int
	// The initial size of the bag (before tiles are drawn)
	Size int
}

// initTileSet makes a complete tile set, given a scoring map
// and a map of letters and their associated counts
func initTileSet(scores map[rune]int, tiles map[rune]int) *TileSet {
	// Count the tiles in the tile set
	numTiles := 0
	for _, count := range tiles {
		numTiles += count
	}
	// Make a tile slice/array to hold the entire tile set
	tileSet := make([]Tile, numTiles)
	// Assign each tile in the tile set
	i := 0
	for letter, count := range tiles {
		score := scores[letter]
		for j := 0; j < count; j++ {
			t := &tileSet[i]
			i++
			t.Letter = letter
			t.Meaning = letter
			t.Score = score
		}
	}
	if i != numTiles {
		panic("Did not assign all tiles in tile set")
	}
	return &TileSet{Tiles: tileSet, Scores: scores, Size: numTiles}
}

// initNewIcelandicTileSet creates the "new" Icelandic
// tile set (as defined by Skraflfélag Íslands) as a fresh array
// (slice) of tiles with the correct number of each letter,
// and marked with the individual tile scores
func initNewIcelandicTileSet() *TileSet {

	// The scores of each letter
	scores := map[rune]int{
		'a': 1, 'á': 3, 'b': 5, 'd': 5, 'ð': 2,
		'e': 3, 'é': 7, 'f': 3, 'g': 3, 'h': 4,
		'i': 1, 'í': 4, 'j': 6, 'k': 2, 'l': 2,
		'm': 2, 'n': 1, 'o': 5, 'ó': 3, 'p': 5,
		'r': 1, 's': 1, 't': 2, 'u': 2, 'ú': 4,
		'v': 5, 'x': 10, 'y': 6, 'ý': 5, 'þ': 7,
		'æ': 4, 'ö': 6, '?': 0,
	}

	// The number of tiles for each letter
	tiles := map[rune]int{
		'a': 11, 'á': 2, 'b': 1, 'd': 1, 'ð': 4,
		'e': 3, 'é': 1, 'f': 3, 'g': 3, 'h': 1,
		'i': 7, 'í': 1, 'j': 1, 'k': 4, 'l': 5,
		'm': 3, 'n': 7, 'o': 1, 'ó': 2, 'p': 1,
		'r': 8, 's': 7, 't': 6, 'u': 6, 'ú': 1,
		'v': 1, 'x': 1, 'y': 1, 'ý': 1, 'þ': 1,
		'æ': 2, 'ö': 1, '?': 2,
	}

	return initTileSet(scores, tiles)
}

// NewIcelandicTileSet is the new standard Icelandic tile set
var NewIcelandicTileSet = initNewIcelandicTileSet()

// initPolishTileSet creates the Polish tile set
func initPolishTileSet() *TileSet {

	scores := map[rune]int{
		'a': 1, 'ą': 5, 'b': 3, 'c': 2, 'ć': 6,
		'd': 2, 'e': 1, 'ę': 5, 'f': 5, 'g': 3,
		'h': 3, 'i': 1, 'j': 3, 'k': 3, 'l': 2,
		'ł': 3, 'm': 2, 'n': 1, 'ń': 7, 'o': 1,
		'ó': 5, 'p': 2, 'r': 1, 's': 1, 'ś': 5,
		't': 2, 'u': 3, 'w': 1, 'y': 2, 'z': 1,
		'ź': 9, 'ż': 5, '?': 0,
	}

	tiles := map[rune]int{
		'a': 9, 'ą': 1, 'b': 2, 'c': 3, 'ć': 1,
		'd': 3, 'e': 7, 'ę': 1, 'f': 1, 'g': 2,
		'h': 2, 'i': 8, 'j': 2, 'k': 3, 'l': 3,
		'ł': 2, 'm': 3, 'n': 5, 'ń': 1, 'o': 6,
		'ó': 1, 'p': 3, 'r': 4, 's': 4, 'ś': 1,
		't': 3, 'u': 2, 'w': 4, 'y': 4, 'z': 5,
		'ź': 1, 'ż': 1, '?': 2,
	}

	return initTileSet(scores, tiles)
}

// PolishTileSet is the standard Polish tile set
var PolishTileSet = initPolishTileSet()

// initNorwegianTileSet creates the new Norwegian tile set
// designed by Taral Guldahl Seierstad, used by permission.
// Thanks Taral!
func initNorwegianTileSet() *TileSet {

	scores := map[rune]int{
		'a': 1, 'b': 3, 'c': 8, 'd': 2, 'e': 1,
		'f': 4, 'g': 2, 'h': 3, 'i': 1, 'j': 5,
		'k': 2, 'l': 1, 'm': 2, 'n': 1, 'o': 2,
		'p': 3, 'r': 1, 's': 1, 't': 1, 'u': 3,
		'v': 3, 'w': 10, 'y': 3, 'æ': 6, 'ø': 4,
		'å': 3, '?': 0,
	}

	tiles := map[rune]int{
		'a': 11, 'b': 3, 'c': 1, 'd': 4, 'e': 12,
		'f': 2, 'g': 3, 'h': 3, 'i': 5, 'j': 2,
		'k': 4, 'l': 5, 'm': 2, 'n': 5, 'o': 4,
		'p': 2, 'r': 6, 's': 4, 't': 5, 'u': 4,
		'v': 3, 'w': 1, 'y': 2, 'æ': 1, 'ø': 2,
		'å': 2, '?': 2,
	}

	return initTileSet(scores, tiles)
}

// NorwegianTileSet is the new Norwegian tile set
var NorwegianTileSet = initNorwegianTileSet()

// initEnglishTileSet creates the standard English tile set
func initEnglishTileSet() *TileSet {

	// The scores of each letter
	scores := map[rune]int{
		'a': 1, 'b': 3, 'c': 3, 'd': 2, 'e': 1,
		'f': 4, 'g': 2, 'h': 4, 'i': 1, 'j': 8,
		'k': 5, 'l': 1, 'm': 3, 'n': 1, 'o': 1,
		'p': 3, 'q': 10, 'r': 1, 's': 1, 't': 1,
		'u': 1, 'v': 4, 'w': 4, 'x': 8, 'y': 4,
		'z': 10, '?': 0,
	}

	// The number of tiles for each letter
	tiles := map[rune]int{
		'a': 9, 'b': 2, 'c': 2, 'd': 4, 'e': 12,
		'f': 2, 'g': 3, 'h': 2, 'i': 9, 'j': 1,
		'k': 1, 'l': 4, 'm': 2, 'n': 6, 'o': 8,
		'p': 2, 'q': 1, 'r': 6, 's': 4, 't': 6,
		'u': 4, 'v': 2, 'w': 2, 'x': 1, 'y': 2,
		'z': 1,
	}

	return initTileSet(scores, tiles)
}

// EnglishTileSet is the (old) standard English tile set
var EnglishTileSet = initEnglishTileSet()

// initNewEnglishTileSet creates the Explo English tile set
func initNewEnglishTileSet() *TileSet {

	// The scores of each letter
	scores := map[rune]int{
		'i': 1, 'o': 1, 's': 1, 'a': 1, 'e': 1,
		't': 2, 'h': 2, 'y': 2, 'm': 2, 'u': 2,
		'd': 2, 'n': 2, 'l': 2, 'r': 2, 'p': 2,
		'k': 3, 'b': 3, 'g': 3, 'c': 3, 'f': 3,
		'w': 4, 'x': 5, 'v': 5, 'j': 6, 'z': 6,
		'q': 12, '?': 0, // Blank tiles
	}

	// The number of tiles for each letter
	tiles := map[rune]int{
		'e': 12, 'a': 11, 's': 9, 'o': 7, 'i': 6,
		'r': 6, 'n': 5, 'l': 5, 't': 4, 'u': 4,
		'd': 4, 'm': 3, 'g': 3, 'c': 3, 'h': 2,
		'y': 2, 'p': 2, 'b': 2, 'k': 1, 'w': 1,
		'f': 1, 'x': 1, 'v': 1, 'j': 1, 'z': 1,
		'q': 1, '?': 2, // Blank tiles
	}

	return initTileSet(scores, tiles)
}

// NewEnglishTileSet is the Explo English tile set
var NewEnglishTileSet = initNewEnglishTileSet()

// Initialize a bag from a tile set and return a reference to it
func makeBag(tileSet *TileSet) *Bag {
	// Make a fresh array for the bag and perform a deep copy of the tile set
	bag := &Bag{}
	bag.Tiles = make([]Tile, len(tileSet.Tiles))
	copy(bag.Tiles, tileSet.Tiles)
	// Create an array of tile pointers as the initial contents of the bag
	bag.Contents = make([]*Tile, len(bag.Tiles))
	for i := range bag.Contents {
		bag.Contents[i] = &bag.Tiles[i]
	}
	// Return a reference
	return bag
}

func (tileSet *TileSet) Contains(letter rune) bool {
	_, ok := tileSet.Scores[letter]
	return ok
}

// DrawTile pops one tile from the (randomized) bag
// and returns it
func (bag *Bag) DrawTile() *Tile {
	tileCount := bag.TileCount()
	if tileCount == 0 {
		// No tiles left in the bag
		return nil
	}
	// Find a random tile in the bag and return it
	i := rand.Intn(tileCount)
	tile := bag.Contents[i]
	bag.Contents = append(bag.Contents[:i], bag.Contents[i+1:]...)
	return tile
}

// DrawTileByLetter draws the specified tile from the bag and
// returns it
func (bag *Bag) DrawTileByLetter(letter rune) *Tile {
	tileCount := bag.TileCount()
	// Find a corresponding tile in the bag
	var i = 0
	for i < tileCount && bag.Contents[i].Letter != letter {
		i++
	}
	if i >= tileCount {
		// No such tile found
		return nil
	}
	// Found the tile: draw it from the bag and return it
	tile := bag.Contents[i]
	bag.Contents = append(bag.Contents[:i], bag.Contents[i+1:]...)
	return tile
}

// ReturnTile returns a previously drawn Tile to the Bag
func (bag *Bag) ReturnTile(tile *Tile) {
	if bag == nil {
		return
	}
	bag.Contents = append(bag.Contents, tile)
}

// String returns a string representation of a Bag
func (bag *Bag) String() string {
	if bag == nil {
		return ""
	}
	var sb strings.Builder
	tileCount := bag.TileCount()
	if tileCount == 0 {
		sb.WriteString("Empty")
	} else {
		sb.WriteString(fmt.Sprintf("(%v tiles): ", tileCount))
		for _, tile := range bag.Contents {
			sb.WriteString(fmt.Sprintf("%v ", tile))
		}
	}
	return sb.String()
}

// TileCount returns the number of tiles in a Bag
func (bag *Bag) TileCount() int {
	if bag == nil {
		return 0
	}
	return len(bag.Contents)
}

// ExchangeAllowed returns true if there are at least RackSize
// tiles left in the bag, thus allowing exchange of tiles
func (bag *Bag) ExchangeAllowed() bool {
	return bag.TileCount() >= RackSize
}
