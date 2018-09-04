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
func simulateGame(gameConstructor GameConstructor, robot *skrafl.RobotWrapper, verbose bool) {
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
		move := robot.GenerateMove(state)
		game.ApplyValid(move)
		p("%v\n", game)
		if game.IsOver() {
			p("Game over!\n")
			break
		}
	}
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
	robot := skrafl.NewHighScoreRobot()
	for i := 0; i < *num; i++ {
		simulateGame(gameConstructor, robot, !*quiet)
	}
}
