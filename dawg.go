// dawg.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file implements the Directed Acyclic Word Graph (DAWG)
// which encodes the dictionary of valid words

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
	"encoding/binary"
	"fmt"
	"os"
)

// Dawg encapsulates the compressed DAWG as a byte buffer
type Dawg struct {
	b      []byte
	coding Coding
}

// ALPHABET contains the letters as they are indexed
// in the compressed binary DAWG
const ALPHABET = "aábdðeéfghiíjklmnoóprstuúvxyýþæö"

// Coding maps an encoded byte to a legal letter, eventually
// suffixed with '|' to denote a final node in the Dawg
type Coding map[byte]Prefix

// A Prefix is an array of runes
type Prefix []rune

// Navigation contains the state of a single navigation that is
// underway within a Dawg
type Navigation struct {
	dawg      *Dawg
	navigator Navigator
}

// Navigator is an interface that describes behaviors that control the
// navigation of a Dawg
type Navigator interface {
	IsAccepting() bool
	Accepts(rune) bool
	Accept(matched string, final bool)
	PushEdge(rune) bool
	PopEdge() bool
	Done()
}

// FindNavigator stores the state for a plain word search in the Dawg,
// and implements the Navigator interface
type FindNavigator struct {
	word  []rune
	index int
	found bool
}

// Init initializes a FindNavigator with the word to search for
func (fn *FindNavigator) Init(word string) {
	fn.word = []rune(word)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (fn *FindNavigator) PushEdge(chr rune) bool {
	return fn.word[fn.index] == chr
}

// PopEdge return false if there is no need to visit other edges
// after this one has been traversed
func (fn *FindNavigator) PopEdge() bool {
	return false
}

// Done is called when the navigation is complete
func (fn *FindNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (fn *FindNavigator) IsAccepting() bool {
	return fn.index < len(fn.word)
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (fn *FindNavigator) Accepts(chr rune) bool {
	if chr != fn.word[fn.index] {
		return false
	}
	fn.index++
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (fn *FindNavigator) Accept(matched string, final bool) {
	if final && fn.index == len(fn.word) {
		fn.found = true
	}
}

// Go starts a navigation on the underlying Dawg using the given
// Navigator
func (nav *Navigation) Go(dawg *Dawg, navigator Navigator) {
	if dawg == nil || navigator == nil {
		return
	}
	nav.dawg = dawg
	nav.navigator = navigator
	if navigator.IsAccepting() {
		nav.FromNode(0, "")
	}
	navigator.Done()
}

// IterPair holds a single iteration result
type IterPair struct {
	prefix   Prefix
	nextNode uint32
}

// IterNode returns a map of prefixes and associated next
// node offsets
func (nav *Navigation) IterNode(offset uint32) []IterPair {
	b := nav.dawg.b
	coding := &nav.dawg.coding
	numEdges := int(b[offset] & 0x7f)
	offset++
	result := make([]IterPair, numEdges)
	for i := 0; i < numEdges; i++ {
		lenByte := b[offset]
		var prefix Prefix
		var nextNode uint32
		offset++
		if lenByte&0x40 != 0 {
			prefix = make(Prefix, 0, 2)
			prefix = append(prefix, (*coding)[lenByte&0x3f]...)
		} else {
			lenByte &= 0x3f
			prefix = make(Prefix, 0, lenByte+1)
			for j := 0; j < int(lenByte); j++ {
				prefix = append(prefix, (*coding)[b[int(offset)+j]]...)
			}
			offset += uint32(lenByte)
		}
		if b[offset-1]&0x80 != 0 {
			nextNode = 0
		} else {
			nextNode = binary.LittleEndian.Uint32(b[offset : offset+4])
			offset += 4
		}
		result[i] = IterPair{prefix: prefix, nextNode: nextNode}
	}
	return result
}

// FromNode continues a navigation from a node in the Dawg
func (nav *Navigation) FromNode(offset uint32, matched string) {
	for _, iter := range nav.IterNode(offset) {
		if nav.navigator.PushEdge(iter.prefix[0]) {
			nav.FromEdge(iter.prefix, iter.nextNode, matched)
			if !nav.navigator.PopEdge() {
				break
			}
		}
	}
}

// FromEdge continues a navigation from an edge in the Dawg
func (nav *Navigation) FromEdge(prefix Prefix, nextNode uint32, matched string) {
	lenP := len(prefix)
	j := 0
	navigator := nav.navigator
	for j < lenP && navigator.IsAccepting() {
		if !navigator.Accepts(prefix[j]) {
			return
		}
		matched += string(prefix[j])
		j++
		final := false
		if j < lenP {
			if prefix[j] == '|' {
				final = true
				j++
			}
		} else {
			if nextNode == 0 || nav.dawg.b[nextNode]&0x80 != 0 {
				final = true
			}
		}
		navigator.Accept(matched, final)
	}
	if j >= lenP && nextNode != 0 && navigator.IsAccepting() {
		nav.FromNode(nextNode, matched)
	}
}

// Init reads the Dawg into memory (or memory-maps it)
func (dawg *Dawg) Init(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	// Get the file size
	info, err := f.Stat()
	if err != nil {
		return err
	}
	size := int(info.Size())
	// Allocate a buffer and read the entire file into it
	dawg.b = make([]byte, size)
	n, err := f.Read(dawg.b)
	if err != nil || n < size {
		return fmt.Errorf("Can't read entire file: '%v'", filePath)
	}
	// Create the alphabet decoding map
	dawg.coding = make(Coding)
	i := byte(0)
	for _, chr := range ALPHABET {
		dawg.coding[i] = make(Prefix, 1)
		dawg.coding[i][0] = chr
		iHigh := i | 0x80
		dawg.coding[iHigh] = make(Prefix, 2)
		dawg.coding[iHigh][0] = chr
		dawg.coding[iHigh][1] = '|'
		i++
	}
	return nil
}

// Find attempts to find a word in a DAWG, returning true if
// found or false if not.
func (dawg *Dawg) Find(word string) bool {
	var fn FindNavigator
	fn.Init(word)
	var nav Navigation
	nav.Go(dawg, &fn)
	return fn.found
}

// Initialize and load an instance of a Dawg from a binary file
// located in the same directory as the skrafl module
func initDawg() *Dawg {
	dawg := &Dawg{}
	err := dawg.Init(os.ExpandEnv("${GOPATH}\\src\\github.com\\vthorsteinsson\\GoSkrafl\\ordalisti.bin.dawg"))
	if err != nil {
		panic(err)
	}
	return dawg
}

// Create a singleton instance of the Dawg
var dawg = initDawg()
