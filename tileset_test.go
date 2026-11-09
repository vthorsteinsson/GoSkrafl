// tileset_test.go
//
// Copyright (C) 2026 Vilhjálmur Þorsteinsson / Miðeind ehf.
//
// Tests for the tile sets and their explicit selection in /moves requests

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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTileSets(t *testing.T) {
	// Each named tile set: its expected size, and two blanks
	sizes := map[string]int{
		"icelandic_old":      104,
		"icelandic_new":      100,
		"english_classic":    100,
		"english_new":        100,
		"polish":             100,
		"norwegian_original": 100,
		"norwegian_new":      100,
	}
	for name, size := range sizes {
		named, ok := tileSetByName(name)
		if !ok {
			t.Errorf("Tile set %v not found", name)
			continue
		}
		ts := named.tileSet
		if ts.Size != size || len(ts.Tiles) != size {
			t.Errorf("Tile set %v has %v tiles, expected %v", name, len(ts.Tiles), size)
		}
		blanks := 0
		for _, tile := range ts.Tiles {
			if tile.Letter == '?' {
				blanks++
			}
			if score, ok := ts.Scores[tile.Letter]; !ok || score != tile.Score {
				t.Errorf("Tile set %v: tile %c has no matching score", name, tile.Letter)
			}
		}
		if blanks != 2 {
			t.Errorf("Tile set %v has %v blanks, expected 2", name, blanks)
		}
	}
	if _, ok := tileSetByName("klingon"); ok {
		t.Errorf("Unknown tile set name accepted")
	}
}

// movesFor runs a /moves request and returns the HTTP status and,
// if it succeeded, the best move and its score
func movesFor(t *testing.T, req MovesRequest) (int, string, int) {
	w := httptest.NewRecorder()
	HandleMovesRequest(w, req)
	if w.Code != http.StatusOK {
		return w.Code, "", 0
	}
	var reply struct {
		Moves []struct {
			Co string `json:"co"`
			W  string `json:"w"`
			Sc int    `json:"sc"`
		} `json:"moves"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &reply); err != nil {
		t.Fatalf("Malformed /moves reply: %v", err)
	}
	if len(reply.Moves) == 0 {
		t.Fatalf("No moves in /moves reply")
	}
	return w.Code, reply.Moves[0].Co + " " + reply.Moves[0].W, reply.Moves[0].Sc
}

func icelandicMovesRequest(tileSet string) MovesRequest {
	board := make([]string, BoardSize)
	for i := range board {
		board[i] = "..............."
	}
	board[7] = ".....hestur...."
	return MovesRequest{
		Locale:    "is_IS",
		BoardType: "standard",
		Board:     board,
		Rack:      "aðeinsp",
		Limit:     1,
		TileSet:   tileSet,
	}
}

func TestMovesRequestTileSet(t *testing.T) {
	// No tile set given: the default for the locale (the new Icelandic one)
	code, defaultMove, defaultScore := movesFor(t, icelandicMovesRequest(""))
	if code != http.StatusOK {
		t.Fatalf("Default tile set: status %v", code)
	}
	code, newMove, newScore := movesFor(t, icelandicMovesRequest("icelandic_new"))
	if code != http.StatusOK || newMove != defaultMove || newScore != defaultScore {
		t.Errorf("icelandic_new should match the default: %v %v vs %v %v",
			newMove, newScore, defaultMove, defaultScore)
	}
	// The old bag has other letter values: the best move scores differently
	code, oldMove, oldScore := movesFor(t, icelandicMovesRequest("icelandic_old"))
	if code != http.StatusOK {
		t.Fatalf("icelandic_old: status %v", code)
	}
	if oldMove == defaultMove && oldScore == defaultScore {
		t.Errorf("icelandic_old scored like the new bag: %v %v", oldMove, oldScore)
	}
	// Unknown tile sets, and tile sets of another language, are rejected
	for _, name := range []string{"klingon", "english_new", "polish"} {
		if code, _, _ := movesFor(t, icelandicMovesRequest(name)); code != http.StatusBadRequest {
			t.Errorf("Tile set %v with locale is_IS: status %v, expected 400", name, code)
		}
	}
	// The classic English tile set on the standard board (with blanks)
	req := icelandicMovesRequest("english_classic")
	req.Locale = "en_US"
	req.Board[7] = ".....horse....."
	req.Rack = "aeinst?"
	if code, _, _ := movesFor(t, req); code != http.StatusOK {
		t.Errorf("english_classic with en_US: status %v", code)
	}
}
