// dawg.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file implements the Directed Acyclic Word Graph (DAWG)
// which encodes the dictionary of valid words.

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
	"go/build"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
)

// Dawg encapsulates the externally generated,
// compressed Directed Acyclic Word Graph as a byte buffer.
// Within the DAWG, letters from the alphabet are represented
// as indices into the alphabet string (below).
// The Coding map translates these indices to the actual
// letters.
// The iterNodeCache map is built on the fly, when
// each Dawg node is traversed for the first time.
// In practice, many nodes will never be traversed.
type Dawg struct {
	// The byte buffer containing the compressed DAWG
	b []byte
	// A mapping from alphabet indices, eventually having
	// the high bit (0x80) set to indicate finality, to rune slices
	coding Coding
	// The alphabet used by the DAWG vocabulary
	alphabet Alphabet
	// mux protects the iterNodeCache
	mux           sync.Mutex
	iterNodeCache map[uint32][]navState
	// crossCache is a cached map of matching patterns
	// to bitmap sets of allowed characters
	crossCache crossCache
}

// Coding maps an encoded byte to a legal letter, eventually
// suffixed with '|' to denote a final node in the Dawg
type Coding map[byte]Prefix

// BitMap maps runes to corresponding bit positions within an
// uint. It follows that the Alphabet cannot have more runes
// in it than uint has bits. Fortunately, few alphabets have
// more than 32/64 runes in them.
type BitMap map[rune]uint

// Alphabet stores the set of runes found within the DAWG,
// and supports bit map (set) operations
type Alphabet struct {
	asString string
	asRunes  []rune
	bitMap   BitMap
	allSet   uint
}

// A Prefix is an array of runes that prefixes an outgoing
// edge in the Dawg
type Prefix []rune

// Init initializes an Alphabet, including a precalculated
// bit map for its runes
func (a *Alphabet) Init(alphabet string) {
	a.asString = alphabet
	a.asRunes = []rune(alphabet)
	a.bitMap = make(BitMap)
	a.allSet = uint(0)
	last := uint(0)
	for i, r := range a.asRunes {
		bit := uint(1 << uint(i))
		if bit < last {
			// Bit overflow, too many runes to be stored in uint
			panic("Alphabet cannot have more runes than the number of bits in uint")
		}
		a.bitMap[r] = bit
		a.allSet |= bit
		last = bit
	}
}

// MakeSet converts a list of runes to a bit map,
// with the extra twist that if any of the runes is '?',
// a bit map with all bits set is returned
func (a *Alphabet) MakeSet(runes []rune) uint {
	s := uint(0)
	for _, r := range runes {
		// Note: if r is not in the map, Go returns uint(0),
		// which is what we want here, so an
		// if bit, ok := ... test is not required.
		if r == '?' {
			return a.allSet
		}
		s |= a.bitMap[r]
	}
	return s
}

// Member checks whether a rune is represented in a bit map
func (a *Alphabet) Member(r rune, set uint) bool {
	// If r is not in the map, the lookup returns uint(0)
	return (set & a.bitMap[r]) != 0
}

// Length returns the number of runes in the Alphabet
func (a *Alphabet) Length() int {
	return len(a.asRunes)
}

// navState holds a navigation state, i.e. an edge where a prefix
// leads to a nextNode
type navState struct {
	prefix   Prefix
	nextNode uint32
}

// iterNode is an internal function that returns a list of
// prefixes and associated next node offsets. We calculate
// this list only once, and then cache it in the Dawg instance.
func (dawg *Dawg) iterNode(offset uint32) []navState {
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
	result := make([]navState, numEdges)
	for i := 0; i < numEdges; i++ {
		lenByte := b[offset]
		state := &result[i]
		offset++
		if lenByte&0x40 != 0 {
			state.prefix = make(Prefix, 0, 2)
			state.prefix = append(state.prefix, (*coding)[lenByte&0x3f]...)
		} else {
			lenByte &= 0x3f
			state.prefix = make(Prefix, 0, lenByte+1)
			for j := 0; j < int(lenByte); j++ {
				state.prefix = append(state.prefix, (*coding)[b[int(offset)+j]]...)
			}
			offset += uint32(lenByte)
		}
		if b[offset-1]&0x80 == 0 {
			// Not a final state
			state.nextNode = binary.LittleEndian.Uint32(b[offset : offset+4])
			offset += 4
		}
	}
	dawg.iterNodeCache[originalOffset] = result
	return result
}

// Init reads the Dawg into memory (TODO: or memory-maps it)
func (dawg *Dawg) Init(filePath string, alphabet string) error {
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
	dawg.alphabet.Init(alphabet)
	for _, chr := range alphabet {
		dawg.coding[i] = make(Prefix, 1)
		dawg.coding[i][0] = chr
		iHigh := i | 0x80
		dawg.coding[iHigh] = make(Prefix, 2)
		dawg.coding[iHigh][0] = chr
		dawg.coding[iHigh][1] = '|'
		i++
	}
	// Create the iteration node cache
	dawg.iterNodeCache = make(map[uint32][]navState)
	// Initialize the cache of cross-check match sets
	dawg.crossCache.Init(2048)
	return nil
}

// Navigate performs a navigation through the DAWG under the
// control of a Navigator
func (dawg *Dawg) Navigate(navigator Navigator) {
	var nav Navigation
	nav.Go(dawg, navigator)
}

// NavigateResumable performs a resumable navigation through the DAWG under the
// control of a Navigator
func (dawg *Dawg) NavigateResumable(navigator Navigator) {
	var nav Navigation
	nav.isResumable = true
	nav.Go(dawg, navigator)
}

// Resume resumes a navigation through the DAWG under the
// control of a Navigator, from a previously saved state
func (dawg *Dawg) Resume(navigator Navigator, state *navState, matched string) {
	var nav Navigation
	nav.Resume(dawg, navigator, state, matched)
}

// Find attempts to find a word in a DAWG, returning true if
// found or false if not.
func (dawg *Dawg) Find(word string) bool {
	var fn FindNavigator
	fn.Init(word)
	dawg.Navigate(&fn)
	return fn.found
}

// Permute finds all permutations of the given rack,
// returning them as a list (slice) of strings.
// The rack may contain '?' wildcards/blanks.
func (dawg *Dawg) Permute(rack string, minLen int) []string {
	var pn PermutationNavigator
	pn.Init(rack, minLen)
	dawg.Navigate(&pn)
	return pn.results
}

// Match returns all words in the Dawg that match a
// given pattern string, which can include '?' wildcards/blanks.
func (dawg *Dawg) Match(pattern string) []string {
	var mn MatchNavigator
	mn.Init([]rune(pattern))
	dawg.Navigate(&mn)
	return mn.results
}

// MatchRunes returns all words in the Dawg that match a
// given pattern, which can include '?' wildcards/blanks.
func (dawg *Dawg) MatchRunes(pattern []rune) []string {
	var mn MatchNavigator
	mn.Init(pattern)
	dawg.Navigate(&mn)
	return mn.results
}

// CrossSet calculates a bit-mapped set of allowed letters
// in a cross-check set, given a left/top and right/bottom
// string that intersects the square being checked.
func (dawg *Dawg) CrossSet(left, right string) uint {
	lenLeft := len([]rune(left))
	key := left + "?" + right
	fetchFunc := func(key string) uint {
		alphabetLength := dawg.alphabet.Length()
		// We ask the DAWG to find all words consisting of the
		// left cross word + wildcard + right cross word,
		// for instance 'f?lt' if the left word is 'f' and the
		// right one is 'lt' - yielding the result set
		// { 'falt', 'filt', fúlt' }, which we convert to the
		// legal cross set of [ 'a', 'i', 'ú' ] and intersect
		// that with the rack
		matches := dawg.Match(key)
		// Collect the 'middle' letters (the ones standing in
		// for the wildcard)
		runes := make([]rune, 0, alphabetLength)
		for _, match := range matches {
			rMatch := []rune(match)
			runes = append(runes, rMatch[lenLeft])
		}
		// Return the resulting bitmapped set
		return dawg.alphabet.MakeSet(runes)
	}
	return dawg.crossCache.Lookup(key, fetchFunc)
}

// crossCache encapsulates a simple LRU cached map of
// cross-set matching patterns ("af?a") to bitmapped sets
type crossCache struct {
	mux sync.Mutex
	lru *simplelru.LRU
}

// Init initalizes an empty crossCache
func (cc *crossCache) Init(size int) {
	cc.lru, _ = simplelru.NewLRU(size, nil)
}

// Lookup returns a bitmap set corresponding to a matching
// pattern key. If the key is found in the cache, it is
// returned immediately. Otherwise, the given fetchFunc() is
// called to calculate the associated bitmap set before storing
// it in the cache.
func (cc *crossCache) Lookup(key string, fetchFunc func(string) uint) uint {
	cc.mux.Lock()
	defer cc.mux.Unlock()
	if bitMap, ok := cc.lru.Get(key); ok {
		return bitMap.(uint)
	}
	bitMap := fetchFunc(key)
	cc.lru.Add(key, bitMap)
	return bitMap
}

// makeDawg initializes a Dawg instance and loads its contents
// from a binary file located in the same directory as the
// skrafl module
func makeDawg(fileName string, alphabet string) *Dawg {
	dawg := &Dawg{}
	goPath := build.Default.GOPATH
	// There should be a better way to do this?
	path := goPath + "/src/github.com/vthorsteinsson/GoSkrafl/" + fileName
	path = filepath.FromSlash(path)
	err := dawg.Init(path, alphabet)
	if err != nil {
		panic(err)
	}
	return dawg
}

// IcelandicAlphabet contains the Icelandic letters as they are indexed
// in the compressed binary DAWG. Note that the Icelandic alphabet does
// not contain 'c', 'q', w' or 'z'.
// TODO: move this to the DAWG file.
const IcelandicAlphabet = "aábdðeéfghiíjklmnoóprstuúvxyýþæö"

// EnglishAlphabet contains the English SCRABBLE(tm) alphabet.
const EnglishAlphabet = "abcdefghijklmnopqrstuvwxyz"

// IcelandicDictionary is a Dawg instance containing the Icelandic
// Scrabble(tm) dictionary, as derived from the BÍN database
// (Beygingarlýsing íslensks nútímamáls)
var IcelandicDictionary = makeDawg("ordalisti.bin.dawg", IcelandicAlphabet)

// Twl06Dictionary is a Dawg instance containing the Tournament
// Word List 06, used in U.S. SCRABBLE(tm).
var Twl06Dictionary = makeDawg("TWL06.bin.dawg", EnglishAlphabet)

// SowpodsDictionary is a Dawg instance containing the SOWPODS
// word list, used in European and U.S. SCRABBLE(tm).
var SowpodsDictionary = makeDawg("sowpods.bin.dawg", EnglishAlphabet)
