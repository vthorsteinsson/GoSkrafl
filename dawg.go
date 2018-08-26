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
	"path/filepath"
	"strings"
	"sync"
)

// Dawg encapsulates the compressed DAWG as a byte buffer
type Dawg struct {
	b      []byte
	coding Coding
	// mux protects the iterNodeCache
	mux           sync.Mutex
	iterNodeCache map[uint32][]iterPair
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
	// Convert the word to a list of runes
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
	// There can only be one correct outgoing edge for the
	// Find function, so we return false to prevent other edges
	// from being tried
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
		// Not a correct next character in the word
		return false
	}
	// This is a correct character: advance our index
	fn.index++
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (fn *FindNavigator) Accept(matched string, final bool) {
	if final && fn.index == len(fn.word) {
		// This is a whole word (final=true) and matches our
		// length, so that's it
		fn.found = true
	}
}

// PermutationNavigator stores the state for a plain word search in the Dawg,
// and implements the Navigator interface
type PermutationNavigator struct {
	rack    string
	stack   []string
	results []string
	minLen  int
}

// Init initializes a PermutationNavigator with the word to search for
func (pn *PermutationNavigator) Init(rack string, minLen int) {
	pn.rack = rack
	pn.minLen = minLen
	pn.stack = make([]string, 0, RackSize)
	pn.results = make([]string, 0)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (pn *PermutationNavigator) PushEdge(chr rune) bool {
	if strings.ContainsRune(pn.rack, chr) || strings.ContainsRune(pn.rack, '?') {
		pn.stack = append(pn.stack, pn.rack)
		return true
	}
	return false
}

// PopEdge return false if there is no need to visit other edges
// after this one has been traversed
func (pn *PermutationNavigator) PopEdge() bool {
	last := len(pn.stack) - 1
	pn.rack = pn.stack[last]
	pn.stack = pn.stack[0:last]
	return true
}

// Done is called when the navigation is complete
func (pn *PermutationNavigator) Done() {
	// The results come out sorted alphabetically.
	// It would be possible to order them by length;
	// the sorting would then appear here.
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (pn *PermutationNavigator) IsAccepting() bool {
	return len(pn.rack) > 0
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (pn *PermutationNavigator) Accepts(chr rune) bool {
	exactMatch := strings.ContainsRune(pn.rack, chr)
	if !exactMatch && !strings.ContainsRune(pn.rack, '?') {
		// The next letter is not in the rack, and the rack
		// does not contain a wildcard/blank: return false
		return false
	}
	if exactMatch {
		// This is a regular letter match
		pn.rack = strings.Replace(pn.rack, string(chr), "", 1)
	} else {
		// This is a wildcard match
		pn.rack = strings.Replace(pn.rack, "?", "", 1)
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (pn *PermutationNavigator) Accept(matched string, final bool) {
	if final && len([]rune(matched)) >= pn.minLen {
		// This is a full word (final=true) and the number of letters
		// is above the minimum limit: add it to the results
		pn.results = append(pn.results, matched)
	}
}

// MatchNavigator stores the state for a pattern matching
// navigation of a Dawg, and implements the Navigator interface
type MatchNavigator struct {
	pattern    []rune
	lenP       int
	index      int
	chMatch    rune
	isWildcard bool
	stack      []matchTuple
	results    []string
}

type matchTuple struct {
	index      int
	chMatch    rune
	isWildcard bool
}

// Init initializes a MatchNavigator with the word to search for
func (mn *MatchNavigator) Init(pattern string) {
	// Convert the word to a list of runes
	mn.pattern = []rune(pattern)
	mn.lenP = len(mn.pattern)
	mn.chMatch = mn.pattern[0]
	mn.isWildcard = mn.chMatch == '?'
	mn.stack = make([]matchTuple, 0, RackSize)
	mn.results = make([]string, 0)
}

// PushEdge determines whether the navigation should proceed into
// an edge having chr as its first letter
func (mn *MatchNavigator) PushEdge(chr rune) bool {
	if chr != mn.chMatch && !mn.isWildcard {
		return false
	}
	mn.stack = append(mn.stack, matchTuple{mn.index, mn.chMatch, mn.isWildcard})
	return true
}

// PopEdge return false if there is no need to visit other edges
// after this one has been traversed
func (mn *MatchNavigator) PopEdge() bool {
	last := len(mn.stack) - 1
	mt := &mn.stack[last]
	mn.index, mn.chMatch, mn.isWildcard = mt.index, mt.chMatch, mt.isWildcard
	mn.stack = mn.stack[0:last]
	return mn.isWildcard
}

// Done is called when the navigation is complete
func (mn *MatchNavigator) Done() {
}

// IsAccepting returns false if the navigator should not expect more
// characters
func (mn *MatchNavigator) IsAccepting() bool {
	return mn.index < mn.lenP
}

// Accepts returns true if the navigator should accept and 'eat' the
// given character
func (mn *MatchNavigator) Accepts(chr rune) bool {
	if chr != mn.chMatch && !mn.isWildcard {
		// Not a correct next character in the word
		return false
	}
	// This is a correct character: advance our index
	mn.index++
	if mn.index < mn.lenP {
		mn.chMatch = mn.pattern[mn.index]
		mn.isWildcard = mn.chMatch == '?'
	}
	return true
}

// Accept is called to inform the navigator of a match and
// whether it is a final word
func (mn *MatchNavigator) Accept(matched string, final bool) {
	if final && mn.index == mn.lenP {
		// Entire pattern match
		mn.results = append(mn.results, matched)
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

// iterPair holds a single iteration result
type iterPair struct {
	prefix   Prefix
	nextNode uint32
}

// iterNode is an internal function that returns a list of
// prefixes and associated next node offsets. We calculate
// this list only once, and then cache it in the Dawg instance.
func (dawg *Dawg) iterNode(offset uint32) []iterPair {
	// Start by looking for this offset in the cached map.
	// We must lock the shared iterNodeCache object since
	// we're reading it and possibly updating it.
	// However, in the great majority of cases, the lock
	// will be held for a very short time only.
	dawg.mux.Lock()
	defer dawg.mux.Unlock()
	if result, ok := dawg.iterNodeCache[offset]; ok {
		// Found: return it
		return result
	}
	// This node has not been previously iterated:
	// create the iteration data, cache them and return them
	originalOffset := offset
	b := dawg.b
	coding := &dawg.coding
	numEdges := int(b[offset] & 0x7f)
	offset++
	result := make([]iterPair, numEdges)
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
		result[i] = iterPair{prefix: prefix, nextNode: nextNode}
	}
	dawg.iterNodeCache[originalOffset] = result
	return result
}

// FromNode continues a navigation from a node in the Dawg
func (nav *Navigation) FromNode(offset uint32, matched string) {
	for _, iter := range nav.dawg.iterNode(offset) {
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

// Init reads the Dawg into memory (TODO: or memory-maps it)
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
	// Create the iteration node cache
	dawg.iterNodeCache = make(map[uint32][]iterPair)
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

// Permute finds all permutations of the given rack,
// returning them as a list (slice) of strings.
// The rack may contain '?' wildcards/blanks.
func (dawg *Dawg) Permute(rack string, minLen int) []string {
	var pn PermutationNavigator
	pn.Init(rack, minLen)
	var nav Navigation
	nav.Go(dawg, &pn)
	return pn.results
}

// Match returns all words in the Dawg that match a
// given pattern, which can include '?' wildcards/blanks.
func (dawg *Dawg) Match(pattern string) []string {
	var mn MatchNavigator
	mn.Init(pattern)
	var nav Navigation
	nav.Go(dawg, &mn)
	return mn.results
}

// makeDawg initializes a Dawg instance and loads its contents
// from a binary file located in the same directory as the
// skrafl module
func makeDawg() *Dawg {
	dawg := &Dawg{}
	path := os.ExpandEnv("${GOPATH}/src/github.com/vthorsteinsson/GoSkrafl/ordalisti.bin.dawg")
	path = filepath.FromSlash(path)
	err := dawg.Init(path)
	if err != nil {
		return nil
	}
	return dawg
}

// WordBase is a singleton instance of the Dawg
var WordBase = makeDawg()
