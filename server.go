// server.go
//
// Copyright (C) 2023 Vilhjálmur Þorsteinsson / Miðeind ehf.
//
// This file implements a compact HTTP server that receives
// JSON encoded requests and returns JSON encoded responses.

package skrafl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"unicode"
)

// A class describing incoming requests
type SkraflRequest struct {
	Dictionary string   `json:"dictionary"`
	Board      []string `json:"board"`
	Rack       string   `json:"rack"`
	BagSize    int      `json:"bag_size"`
}

// A kludge to be able to marshal a Move with its score
type MoveWithScore struct {
	json.Marshaler
	Move  Move
	Score int
}

func (m *MoveWithScore) MarshalJSON() ([]byte, error) {
	// Let the move marshal itself, but adding the score
	return m.Move.Marshal(m.Score)
}

// The JSON response header
type HeaderJson struct {
	Version string          `json:"version"`
	Moves   []MoveWithScore `json:"moves"`
}

// Handle an incoming request
func HandleRequest(w http.ResponseWriter, req SkraflRequest) {
	// Set the dictionary and tile set
	tileSet := NewIcelandicTileSet
	dawg := IcelandicDictionary

	switch req.Dictionary {
	case "twl06":
		tileSet = EnglishTileSet
		dawg = Twl06Dictionary
	case "sowpods":
		tileSet = EnglishTileSet
		dawg = SowpodsDictionary
	case "ice":
		// Already set
	// TODO: Add Polish dictionary
	default:
		msg := fmt.Sprintf("Unknown dictionary '%v'. Specify one of 'twl06', 'sowpods' or 'ice'.\n", req.Dictionary)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if len(req.Board) != 15 {
		msg := "Invalid board. Must be 15 rows.\n"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	board := NewBoard()
	for r, rowString := range req.Board {
		row := []rune(rowString)
		if len(row) != 15 {
			msg := fmt.Sprintf("Invalid board row (#%v). Must be 15 characters long.\n", r)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		for c, letter := range row {
			if letter != '.' && letter != ' ' {
				meaning := letter
				score := 0
				// Uppercase letters represent
				// blank tiles that have been assigned a letter;
				// convert these to lowercase letters and
				// give them a score of 0
				if unicode.IsUpper(letter) {
					meaning = unicode.ToLower(letter)
					letter = '?'
				} else {
					score = tileSet.Scores[letter]
				}
				t := &Tile{
					Letter:  letter,
					Meaning: meaning,
					Score:   score,
				}
				board.Sq(r, c).Tile = t
			}
		}
	}

	// Parse the incoming rack string
	rack := NewRack(req.Rack, tileSet)

	// Create a GameState object, then find the valid moves
	state := NewState(
		dawg,
		tileSet,
		board,
		rack,
		req.BagSize,
	)

	// Generate all valid moves and calculate their scores
	moves := state.GenerateMoves()
	movesWithScores := make([]MoveWithScore, len(moves))
	for i, move := range moves {
		movesWithScores[i] = MoveWithScore{
			Move:  move,
			Score: move.Score(state),
		}
	}

	// Return the result as JSON
	result := HeaderJson{
		Version: "1.0",
		Moves:   movesWithScores,
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
