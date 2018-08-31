// bag.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file contains the Bag logic

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
type Bag []Tile

// TileSet is a static list of tiles, used as a prototype
// to copy new Bags from
type TileSet struct {
	Tiles  []Tile
	Scores map[rune]int
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
	return &TileSet{Tiles: tileSet, Scores: scores}
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

// Initialize a bag from a tile set and return a reference to it
func makeBag(tileSet *TileSet) *Bag {
	// Make a fresh array for the bag and copy the tile set to it
	bag := make(Bag, len(tileSet.Tiles))
	copy(bag, tileSet.Tiles)
	// Shuffle the bag
	rand.Shuffle(len(bag), func(i, j int) {
		bag[i], bag[j] = bag[j], bag[i]
	})
	// Return a reference
	return &bag
}

// DrawTile pops one tile from the (randomized) bag
// and returns it
func (bag *Bag) DrawTile() *Tile {
	if bag == nil || len(*bag) == 0 {
		// No tiles left in the bag
		return nil
	}
	// We pop the last tile from the bag and return it
	lenBag := len(*bag)
	tile := &(*bag)[lenBag-1]
	*bag = (*bag)[0 : lenBag-1]
	return tile
}

// String returns a string representation of a Bag
func (bag *Bag) String() string {
	if bag == nil {
		return ""
	}
	var sb strings.Builder
	if len(*bag) == 0 {
		sb.WriteString("Empty")
	} else {
		sb.WriteString(fmt.Sprintf("(%v tiles): ", bag.TileCount()))
		for i := 0; i < len(*bag); i++ {
			sb.WriteString(fmt.Sprintf("%v ", &(*bag)[i]))
		}
	}
	return sb.String()
}

// TileCount returns the number of tiles in a Bag
func (bag *Bag) TileCount() int {
	if bag == nil {
		return 0
	}
	return len(*bag)
}

// ExchangeAllowed returns true if there are at least RackSize
// tiles left in the bag, thus allowing exchange of tiles
func (bag *Bag) ExchangeAllowed() bool {
	return bag != nil && len(*bag) >= RackSize
}
