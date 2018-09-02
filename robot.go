// robot.go
// Copyright (C) 2018 Vilhjálmur Þorsteinsson
// This file implements a SCRABBLE(tm) playing robot,
// and is a part of the Go 'skrafl' package.

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

import "sort"

// Robot is an interface for automatic players that implement
// a playing strategy to pick a move given a list of legal tile
// moves.
type Robot interface {
	PickMove(state *GameState, moves []Move) Move
}

// RobotWrapper wraps a Robot implementation
type RobotWrapper struct {
	Robot
}

// GenerateMove generates a list of legal tile moves, then
// asks the wrapped robot to pick one of them to play
func (rw *RobotWrapper) GenerateMove(state *GameState) Move {
	moves := state.GenerateMoves()
	return rw.PickMove(state, moves)
}

// HighScoreRobot implements a simple strategy: it always picks
// the highest-scoring move available, or exchanges all tiles
// if there is no valid tile move, or passes if exchange is not
// allowed.
type HighScoreRobot struct {
}

// Implement a strategy for sorting move lists by score

type byScore struct {
	state *GameState
	moves []Move
}

func (list byScore) Len() int {
	return len(list.moves)
}

func (list byScore) Swap(i, j int) {
	list.moves[i], list.moves[j] = list.moves[j], list.moves[i]
}

func (list byScore) Less(i, j int) bool {
	// We want descending order, so we reverse the comparison
	return list.moves[i].Score(list.state) > list.moves[j].Score(list.state)
}

// PickMove for a HighScoreRobot picks the highest scoring move available,
// or an exchange move, or a pass move as a last resort
func (robot *HighScoreRobot) PickMove(state *GameState, moves []Move) Move {
	if len(moves) > 0 {
		// Sort by score and return the highest scoring move
		sort.Sort(byScore{state, moves})
		return moves[0]
	}
	// No valid tile moves
	if !state.exchangeForbidden {
		// Exchange all tiles, since that is allowed
		return NewExchangeMove(state.Rack.AsString())
	}
	// Exchange forbidden: Return a pass move
	return NewPassMove()
}

// NewHighScoreRobot returns a fresh instance of a HighestScoreRobot
func NewHighScoreRobot() *RobotWrapper {
	return &RobotWrapper{&HighScoreRobot{}}
}
