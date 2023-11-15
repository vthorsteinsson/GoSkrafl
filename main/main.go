// main.go
// Copyright (C) 2023 Vilhjálmur Þorsteinsson / Miðeind ehf.

// Example main program for exercising the skrafl module

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"unicode"

	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

// GameConstructor is a function that returns the type of Game we want
type GameConstructor func() *skrafl.Game

// Generate a sequence of moves and responses
func simulateGame(gameConstructor GameConstructor,
	robotA *skrafl.RobotWrapper, robotB *skrafl.RobotWrapper,
	verbose bool) (scoreA, scoreB int) {

	// Wrap fmt.Printf
	var p func(string, ...interface{}) (int, error)
	if verbose {
		p = fmt.Printf
	} else {
		p = func(format string, a ...interface{}) (int, error) { return 0, nil }
	}
	game := gameConstructor()
	game.SetPlayerNames("Robot A", "Robot B")
	p("%v\n", game)
	for i := 0; ; i++ {
		state := game.State()
		var move skrafl.Move
		// Ask robotA or robotB to generate a move
		if i%2 == 0 {
			move = robotA.GenerateMove(state)
		} else {
			move = robotB.GenerateMove(state)
		}
		game.ApplyValid(move)
		p("%v\n", game)
		if game.IsOver() {
			p("Game over!\n\n")
			break
		}
	}
	scoreA, scoreB = game.Scores[0], game.Scores[1]
	return // scoreA, scoreB
}

// A class describing incoming requests
type SkraflRequest struct {
	Dictionary string   `json:"dictionary"`
	Board      []string `json:"board"`
	Rack       string   `json:"rack"`
	BagSize    int      `json:"bag_size"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req SkraflRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tileSet := skrafl.NewIcelandicTileSet
	dawg := skrafl.IcelandicDictionary

	switch req.Dictionary {
	case "twl06":
		tileSet = skrafl.EnglishTileSet
		dawg = skrafl.Twl06Dictionary
	case "sowpods":
		tileSet = skrafl.EnglishTileSet
		dawg = skrafl.SowpodsDictionary
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

	board := skrafl.NewBoard()
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
				t := &skrafl.Tile{
					Letter:  letter,
					Meaning: meaning,
					Score:   score,
				}
				board.Sq(r, c).Tile = t
			}
		}
	}

	// Parse the rack
	rack := skrafl.NewRack(req.Rack, tileSet)

	// Create a GameState object, then find the valid moves
	state := skrafl.NewState(
		dawg,
		tileSet,
		board,
		rack,
		req.BagSize,
	)

	// Generate all valid moves
	moves := state.GenerateMoves()

	// Return the result as JSON
	err = json.NewEncoder(w).Encode(moves)
	if err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func runServer() {
	http.HandleFunc("/moves", handler)
	http.ListenAndServe(":8080", nil)
}

func main() {
	// Modify the following depending on the type of Game wanted
	dict := flag.String("d", "ice", "Dictionary to use (twl06, sowpods, ice)")
	num := flag.Int("n", 10, "Number of games to simulate")
	quiet := flag.Bool("q", false, "Suppress output of game state and moves")
	server := flag.Bool("s", false, "Run as a HTTP server")
	flag.Parse()
	if server != nil && *server {
		// Run a HTTP server
		runServer()
		return
	}
	gameConstructor := skrafl.NewIcelandicGame
	switch *dict {
	case "twl06":
		gameConstructor = skrafl.NewTwl06Game
	case "sowpods":
		gameConstructor = skrafl.NewSowpodsGame
	case "ice":
		// Already set
	default:
		fmt.Printf("Unknown dictionary '%v'. Specify one of 'twl06', 'sowpods' or 'ice'.\n", *dict)
		os.Exit(1)
	}
	robotA := skrafl.NewHighScoreRobot()
	robotB := skrafl.NewHighScoreRobot()
	// robotB := skrafl.NewOneOfNBestRobot(10) // Picks one of 10 best moves
	var winsA, winsB int
	for i := 0; i < *num; i++ {
		scoreA, scoreB := simulateGame(gameConstructor, robotA, robotB, !*quiet)
		if scoreA > scoreB {
			winsA++
		} else {
			if scoreB > scoreA {
				winsB++
			}
		}
	}
	fmt.Printf("%v games were played using the '%v' dictionary.\n"+
		"Robot A won %v games, and Robot B won %v games; %v games were draws.\n",
		*num, *dict,
		winsA, winsB, *num-winsA-winsB)
}
