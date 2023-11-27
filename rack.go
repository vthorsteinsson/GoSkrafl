// rack.go
// Copyright (C) 2023 Vilhjálmur Þorsteinsson / Miðeind ehf.

// This file implements the Rack struct and its operations

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

// RackSize contains the number of slots in the Rack
const RackSize = 7

// RackTiles contains a map of tiles with their count,
// with blank tiles being represented by '?'
type RackTiles struct {
	Tiles map[rune]int
}

// Rack represents a player's rack of Tiles
type Rack struct {
	Slots   [RackSize]Square
	Content RackTiles
}

func MakeRackTiles(rack []rune) *RackTiles {
	rt := RackTiles{}
	for _, r := range rack {
		rt.AddTile(r)
	}
	return &rt
}

// Add a tile (rune) to a RackTiles map
func (rack *RackTiles) AddTile(tile rune) {
	if rack.Tiles == nil {
		rack.Tiles = make(map[rune]int)
	}
	rack.Tiles[tile]++
}

// Remove a tile (rune) from a RackTiles map
func (rack *RackTiles) RemoveTile(tile rune) bool {
	if rack.Tiles == nil {
		return false
	} else if _, ok := rack.Tiles[tile]; !ok {
		return false
	}
	rack.Tiles[tile]--
	return true
}

func (rack *RackTiles) ContainsBlank() bool {
	return rack.Tiles != nil && rack.Tiles['?'] > 0
}

func (rack *RackTiles) ContainsTile(t rune) bool {
	return rack.Tiles != nil && rack.Tiles[t] > 0
}

// Shortcut to add a tile (rune) to a Rack
func (rack *Rack) AddTile(tile rune) {
	rack.Content.AddTile(tile)
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
			rack.AddTile(sq.Tile.Letter)
		} else {
			// Can't fill all empty slots: return false
			return false
		}
	}
	// Able to fill all empty slots
	return true
}

// FillByLetters draws tiles identified by the given
// array of letters from the Bag to fill the Rack,
// at least as far as possible.
// Returns false if a tile corresponding to a letter
// from the array is not found in the bag.
func (rack *Rack) FillByLetters(bag *Bag, letters []rune) bool {
	for i := 0; i < RackSize && len(letters) > 0; i++ {
		sq := &rack.Slots[i]
		if sq.Tile == nil {
			if sq.Tile = bag.DrawTileByLetter(letters[0]); sq.Tile == nil {
				// A requested tile was not found in the bag
				return false
			}
			// Success: cut the letter from the letters array
			letters = letters[1:]
		}
		// Got a new tile in the rack:
		// increment the letter's count in the rack map
		rack.AddTile(sq.Tile.Letter)
	}
	// Could fill rack as far as possible according to the letters array
	return true
}

// Init initializes an empty rack
func (rack *Rack) Init() {
	// Initialize empty rack slots
	for i := range rack.Slots {
		sq := &rack.Slots[i]
		sq.Row = -1
		sq.Col = i
		sq.LetterMultiplier = 1
		sq.WordMultiplier = 1
	}
}

// Create a rack containing the tiles specified in the string r,
// with '?' denoting the blank tile
func NewRack(r []rune, tileSet *TileSet) *Rack {
	rack := &Rack{}
	// Initialize rack slots
	slot := 0
	for _, letter := range r {
		sq := &rack.Slots[slot]
		sq.Row = -1
		sq.Col = slot
		sq.LetterMultiplier = 1
		sq.WordMultiplier = 1
		// If tileSet does not contain the letter, return nil
		score, ok := tileSet.Scores[letter]
		if !ok {
			return nil
		}
		sq.Tile = &Tile{
			Letter:  letter,
			Meaning: letter,
			Score:   score,
		}
		rack.AddTile(letter)
		slot++
	}
	// Fill in the rest of the rack, if not already full
	for i := slot; i < RackSize; i++ {
		sq := &rack.Slots[i]
		sq.Row = -1
		sq.Col = i
		sq.LetterMultiplier = 1
		sq.WordMultiplier = 1
	}
	return rack
}

// String returns a printable string representation of a Rack
func (rack *Rack) String() string {
	var sb strings.Builder
	for _, sq := range rack.Slots {
		sb.WriteString(fmt.Sprintf("%v ", &sq))
	}
	return sb.String()
}

// AsRunes returns the tiles in the Rack as a list of runes
func (rack *Rack) AsRunes() []rune {
	runes := make([]rune, 0, RackSize)
	for _, sq := range rack.Slots {
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
	if rack == nil || tile == nil {
		return false
	}
	for _, sq := range rack.Slots {
		if sq.Tile == tile {
			return true
		}
	}
	return false
}

// IsEmpty returns true if the Rack is empty
func (rack *Rack) IsEmpty() bool {
	if rack == nil {
		return true
	}
	for _, sq := range rack.Slots {
		if sq.Tile != nil {
			return false
		}
	}
	return true
}

// FindTile finds a tile with the given letter (or '?') in the
// rack and returns a pointer to it, or nil if not found
func (rack *Rack) FindTile(letter rune) *Tile {
	if rack == nil {
		return nil
	}
	for _, sq := range rack.Slots {
		if sq.Tile != nil && sq.Tile.Letter == letter {
			return sq.Tile
		}
	}
	return nil
}

// FindTiles finds tiles corresponding to the given letters (or '?')
// in the rack and returns a list. If tiles are not found in the rack,
// they are not included in the result. Note that the same tile is
// not returned twice, even if a particular letter is requested twice.
func (rack *Rack) FindTiles(letters []rune) []*Tile {
	if rack == nil {
		return nil
	}
	result := make([]*Tile, 0, len(letters))
	var picked [RackSize]bool
	for _, letter := range letters {
		for i, sq := range rack.Slots {
			if !picked[i] && sq.Tile != nil && sq.Tile.Letter == letter {
				result = append(result, sq.Tile)
				picked[i] = true
				break
			}
		}
	}
	return result
}

// RemoveTile removes a tile from a Rack
func (rack *Rack) RemoveTile(tile *Tile) bool {
	if rack == nil || tile == nil {
		return false
	}
	for i := range rack.Slots {
		sq := &rack.Slots[i]
		if sq.Tile == tile {
			// Found the slot with the tile:
			// remove it from the rack
			if !rack.Content.RemoveTile(tile.Letter) {
				// Should never happen!
				panic(
					fmt.Sprintf(
						"Rack does not contain tile: %v",
						tile,
					),
				)
			}
			sq.Tile = nil
			return true
		}
	}
	// Tile was not found in the rack
	return false
}

// ReturnToBag returns the tiles in the Rack to a Bag
func (rack *Rack) ReturnToBag(bag *Bag) {
	if rack == nil || bag == nil {
		return
	}
	for i := range rack.Slots {
		sq := &rack.Slots[i]
		if sq.Tile != nil {
			// This slot has a tile: remove it and
			// return it to the bag
			if !rack.Content.RemoveTile(sq.Tile.Letter) {
				// Should never happen!
				panic(
					fmt.Sprintf(
						"Rack does not contain tile: %v",
						sq.Tile.Letter,
					),
				)
			}
			bag.ReturnTile(sq.Tile)
			sq.Tile = nil
		}
	}
}

// Extract obtains the given number of tiles from the rack,
// returning them as a list. If a tile is blank,
// assign the given meaning to it. This function is useful
// for debugging and testing purposes.
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
