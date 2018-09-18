// main.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson

// Example main program for exercising the skrafl module

package main

import (
	"flag"
	"fmt"
	"os"

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

func main() {
	// Modify the following depending on the type of Game wanted
	gameConstructor := skrafl.NewIcelandicGame
	dict := flag.String("d", "ice", "Dictionary to use (twl06, sowpods, ice)")
	num := flag.Int("n", 10, "Number of games to simulate")
	quiet := flag.Bool("q", false, "Suppress output of game state and moves")
	flag.Parse()
	switch *dict {
	case "twl06":
		gameConstructor = skrafl.NewTwl06Game
	case "sowpods":
		gameConstructor = skrafl.NewSowpodsGame
	case "ice":

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
