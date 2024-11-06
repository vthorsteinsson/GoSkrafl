// main.go
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.

// Example main program for exercising the skrafl module

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

// GameConstructor is a function that returns the type of Game we want
type GameConstructor func(boardType string) *skrafl.Game

// Generate a sequence of moves and responses
func simulateGame(gameConstructor GameConstructor, boardType string,
	robotA *skrafl.RobotWrapper, robotB *skrafl.RobotWrapper,
	verbose bool) (scoreA, scoreB int) {

	// Wrap fmt.Printf
	var p func(string, ...interface{}) (int, error)
	if verbose {
		p = fmt.Printf
	} else {
		p = func(format string, a ...interface{}) (int, error) { return 0, nil }
	}
	game := gameConstructor(boardType)
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

func movesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req skrafl.MovesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	skrafl.HandleMovesRequest(w, req)
}

func wordcheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req skrafl.WordCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	skrafl.HandleWordCheckRequest(w, req)
}

func runServer() {
	http.HandleFunc("/moves", movesHandler)
	http.HandleFunc("/wordcheck", wordcheckHandler)
	http.ListenAndServe(":8080", nil)
}

func main() {
	// Modify the following depending on the type of Game wanted
	dict := flag.String("d", "ice", "Dictionary to use (otcwl, sowpods, osps, ice)")
	boardType := flag.String("b", "standard", "Board type (standard, explo)")
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
	case "octwl":
		gameConstructor = skrafl.NewOtcwlGame
	case "sowpods":
		gameConstructor = skrafl.NewSowpodsGame
	case "osps":
		gameConstructor = skrafl.NewOspsGame
	case "ice":
		// Already set
	default:
		fmt.Printf("Unknown dictionary '%v'. Specify one of 'otcwl', 'sowpods', 'osps', or 'ice'.\n", *dict)
		os.Exit(1)
	}
	robotA := skrafl.NewHighScoreRobot()
	robotB := skrafl.NewHighScoreRobot()
	// robotB := skrafl.NewOneOfNBestRobot(10) // Picks one of 10 best moves
	var winsA, winsB int
	for i := 0; i < *num; i++ {
		scoreA, scoreB := simulateGame(gameConstructor, *boardType, robotA, robotB, !*quiet)
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
