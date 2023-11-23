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
	"sort"
	"unicode"
)

// A class describing incoming requests
type SkraflRequest struct {
	Locale    string   `json:"locale"`
	BoardType string   `json:"board_type"`
	Board     []string `json:"board"`
	Rack      string   `json:"rack"`
	Limit     int      `json:"limit"`
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
	Count   int             `json:"count"`
	Moves   []MoveWithScore `json:"moves"`
}

// Handle an incoming request
func HandleRequest(w http.ResponseWriter, req SkraflRequest) {
	// Set the board type, dictionary and tile set
	boardType := req.BoardType
	if boardType != "standard" && boardType != "explo" {
		msg := "Invalid board type. Must be 'standard' or 'explo'.\n"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// Set the dictionary from the request's locale
	var dictionary string
	locale := req.Locale
	// Obtain the first three characters of locale
	locale3 := locale
	if len(locale) > 3 {
		locale3 = locale[0:3]
	}
	if locale == "" || locale == "en_US" || locale == "en-US" {
		// U.S. English
		dictionary = "otcwl"
	} else if locale == "en" || locale3 == "en_" || locale3 == "en-" {
		// U.K. English (SOWPODS)
		dictionary = "sowpods"
	} else if locale == "is" || locale3 == "is_" || locale3 == "is-" {
		// Icelandic
		dictionary = "ice"
	} else if locale == "pl" || locale3 == "pl_" || locale3 == "pl-" {
		// Polish
		dictionary = "osps"
	} else {
		// Default to U.S. English for other locales
		dictionary = "otcwl"
	}

	var tileSet *TileSet
	var dawg *Dawg

	switch dictionary {
	case "otcwl":
		dawg = OtcwlDictionary
		if boardType == "explo" {
			tileSet = NewEnglishTileSet
		} else {
			tileSet = EnglishTileSet
		}
	case "sowpods":
		dawg = SowpodsDictionary
		if boardType == "explo" {
			tileSet = NewEnglishTileSet
		} else {
			tileSet = EnglishTileSet
		}
	case "ice":
		dawg = IcelandicDictionary
		tileSet = NewIcelandicTileSet
	case "osps":
		dawg = OspsDictionary
		tileSet = PolishTileSet
	}

	rackRunes := []rune(req.Rack)
	if len(rackRunes) == 0 || len(rackRunes) > RackSize {
		msg := "Invalid rack.\n"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if len(req.Board) != BoardSize {
		msg := fmt.Sprintf("Invalid board. Must be %v rows.\n", BoardSize)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	board := NewBoard(boardType)
	for r, rowString := range req.Board {
		row := []rune(rowString)
		if len(row) != BoardSize {
			msg := fmt.Sprintf(
				"Invalid board row (#%v). Must be %v characters long.\n",
				r,
				BoardSize,
			)
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
				if !tileSet.Contains(letter) {
					msg := fmt.Sprintf("Invalid letter '%c' at %v,%v.\n", letter, r, c)
					http.Error(w, msg, http.StatusBadRequest)
					return
				}
				t := &Tile{
					Letter:  letter,
					Meaning: meaning,
					Score:   score,
				}
				if ok := board.PlaceTile(r, c, t); !ok {
					// Should not happen, and if it does, it's a serious bug,
					// so no point in continuing
					panic(fmt.Sprintf("Square already occupied: %v,%v", r, c))
				}
			}
		}
	}

	// The board must either be empty or have a tile in the start square
	if board.NumTiles > 0 && !board.HasStartTile() {
		msg := "The start square must be occupied.\n"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// Parse the incoming rack string
	rack := NewRack(rackRunes, tileSet)
	if rack == nil {
		msg := "Rack contains invalid letter.\n"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// Create a fresh GameState object, then find the valid moves
	exchangeForbidden := tileSet.Size-board.NumTiles-2*RackSize < RackSize
	state := NewState(
		dawg,
		tileSet,
		board,
		rack,
		exchangeForbidden,
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
	// Sort the movesWithScores list in descending order by Score
	sort.Slice(movesWithScores, func(i, j int) bool {
		return movesWithScores[i].Score > movesWithScores[j].Score
	})
	// If a limit is specified, use that as a cap on the number of moves returned
	if req.Limit > 0 {
		movesWithScores = movesWithScores[0:min(req.Limit, len(movesWithScores))]
	}

	// Return the result as JSON
	result := HeaderJson{
		Version: "1.0",
		Count:   len(movesWithScores),
		Moves:   movesWithScores,
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		// Unable to generate valid JSON
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
