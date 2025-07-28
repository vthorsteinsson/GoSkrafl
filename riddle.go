// riddle.go
//
// Copyright (C) 2025 Vilhjálmur Þorsteinsson / Miðeind ehf.
//
// This file implements the riddle generation logic.

package skrafl

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// GenerationParams holds the parameters for riddle generation.
type GenerationParams struct {
	Locale        string
	BoardType     string
	Dawg          *Dawg    // The DAWG for the locale
	TileSet       *TileSet // The tile set for the locale
	TimeLimit     time.Duration
	NumWorkers    int
	NumCandidates int // Number of candidates to generate
}

// HeuristicConfig defines the parameters for what constitutes a "good" riddle.
type HeuristicConfig struct {
	MinTiles       int     // Minimum number of tiles on the board
	MaxTiles       int     // Maximum number of tiles on the board
	MinMoves       int     // Minimum number of valid tile moves available
	MinBestScore   int     // Minimum score for the best move
	MinWordLength  int     // Minimum length of the solution word
	BingoBonus     float64 // Bonus for bingo moves (all tiles used)
	ScoreGapBonus  float64 // Bonus factor for the gap between the best and second-best move scores
	NumCoversBonus float64 // Bonus factor for the number of tiles in the move
	SolutionFilter *Dawg   // Optional: A DAWG to filter solution words against
}

// DefaultHeuristics provides a baseline configuration.
var DefaultHeuristics = HeuristicConfig{
	MinTiles:       50,
	MaxTiles:       70,
	MinMoves:       16,
	MinBestScore:   30,
	MinWordLength:  3,
	BingoBonus:     15.0,
	ScoreGapBonus:  1.2,
	NumCoversBonus: 2.0,
	SolutionFilter: nil,
}

// IcelandicHeuristics adds a common word filter for Icelandic riddles.
func createIcelandicHeuristics() HeuristicConfig {
	h := DefaultHeuristics
	h.SolutionFilter = IcelandicCommonWordsDictionary
	return h
}

var IcelandicHeuristics = createIcelandicHeuristics()

// Solution holds the answer to the riddle.
type Solution struct {
	Move        string `json:"move"`
	Coord       string `json:"coord"`
	Score       int    `json:"score"`
	Description string `json:"description"`
}

// Analysis provides metrics about the riddle's move possibilities.
type Analysis struct {
	TotalMoves          int     `json:"totalMoves"`
	BestMoveScore       int     `json:"bestMoveScore"`
	SecondBestMoveScore int     `json:"secondBestMoveScore"`
	AverageScore        float64 `json:"averageScore"`
	IsBingo             bool    `json:"isBingo"`
}

// Riddle is the final structure returned by the API.
type Riddle struct {
	Board    []string `json:"board"`
	Rack     string   `json:"rack"`
	Solution Solution `json:"solution"`
	Analysis Analysis `json:"analysis"`
}

// RiddleCandidate holds a potential riddle and its evaluated metrics.
type RiddleCandidate struct {
	Riddle *Riddle
	Score  float64
}

// scoredMove is a helper struct to hold a move and its score for sorting.
type scoredMove struct {
	Move  *TileMove
	Score int
}

type Stats struct {
	Candidates int64 // Number of candidates generated
	// The following are rejection statistics
	NoValidMove      int // No valid move available
	GameEnded        int // Game already ended, no riddle possible
	ContextCancelled int // Context was cancelled before a riddle could be generated
	TooFewMoves      int // Unacceptable number of tile moves available
	TooManyMoves     int // Unacceptable number of tile moves available
	TooLowBestScore  int // Best move score too low
	TooShortWord     int // Best move word too short
	WordNotCommon    int // Solution word not in the common words dictionary
}

// generateCandidate creates a single riddle candidate.
func generateCandidate(
	ctx context.Context,
	params GenerationParams,
	heuristics HeuristicConfig,
	stats *Stats,
) (*RiddleCandidate, error) {
	// Create a new game with two high-score robots.
	p1 := NewHighScoreRobot()
	p2 := NewHighScoreRobot()

	game, err := NewGameForLocale(params.Locale, params.BoardType)
	if err != nil {
		return nil, err
	}
	game.PlayerNames[0] = "P1"
	game.PlayerNames[1] = "P2"

	// Play turns to populate the board until the count of tiles
	// is above a random number in the interval heuristics.MinTiles to heuristics.MaxTiles.
	minTiles := heuristics.MinTiles + rand.Intn(heuristics.MaxTiles-heuristics.MinTiles+1)
	moveIndex := 0
	for game.Board.NumTiles < minTiles {
		state := game.State()
		var move Move
		if moveIndex%2 == 0 {
			move = p1.GenerateMove(state)
		} else {
			move = p2.GenerateMove(state)
		}
		if move == nil {
			stats.NoValidMove++
			return nil, nil // No valid move available, can't generate a riddle
		}
		moveIndex++
		game.ApplyValid(move)

		if game.IsOver() {
			stats.GameEnded++
			return nil, nil // Game already ended, no riddle possible
		}

		// Check for context cancellation to allow for early exit after a full turn.
		select {
		case <-ctx.Done():
			stats.ContextCancelled++
			return nil, ctx.Err() // Exit if the context has been canceled.
		default:
			// Continue if not canceled.
		}
	}

	// The current state is our candidate.
	state := game.State()
	board := state.Board
	rack := state.Rack.AsString()
	moves := state.GenerateMoves()

	// Score and sort the moves.
	scoredMoves := make([]scoredMove, 0, len(moves))
	for _, m := range moves {
		// We are only interested in TileMoves for riddles
		if tm, ok := m.(*TileMove); ok {
			scoredMoves = append(scoredMoves, scoredMove{Move: tm, Score: tm.Score(state)})
		}
	}

	numMoves := len(scoredMoves)
	if numMoves < heuristics.MinMoves {
		stats.TooFewMoves++
		return nil, nil // Unacceptable number of tile moves available
	}

	sort.Slice(scoredMoves, func(i, j int) bool {
		return scoredMoves[i].Score > scoredMoves[j].Score
	})

	bestMove := scoredMoves[0]
	if bestMove.Score < heuristics.MinBestScore {
		stats.TooLowBestScore++
		return nil, nil // Best move score too low
	}
	tm := bestMove.Move
	cleanWord := tm.CleanWord()
	cleanRunes := []rune(cleanWord)
	if len(cleanRunes) < heuristics.MinWordLength {
		stats.TooShortWord++
		return nil, nil // Best move word too short
	}

	// If a solution filter is configured, apply it now.
	// This is e.g. used to ensure that the solution word is a fairly common word.
	if heuristics.SolutionFilter != nil {
		if !heuristics.SolutionFilter.Find(cleanWord) {
			stats.WordNotCommon++
			return nil, nil // Solution word not in the common words dictionary.
		}
	}

	secondBestScore := bestMove.Score
	if numMoves > 1 {
		secondBestScore = scoredMoves[1].Score
	}

	totalScore := 0
	for _, sm := range scoredMoves {
		totalScore += sm.Score
	}

	isBingo := len(tm.Covers) == RackSize

	analysis := Analysis{
		TotalMoves:          numMoves,
		BestMoveScore:       bestMove.Score,
		SecondBestMoveScore: secondBestScore,
		AverageScore:        float64(totalScore) / float64(numMoves),
		IsBingo:             isBingo,
	}

	solution := Solution{
		Move:        tm.Word, // Note: includes '?' for blank tiles
		Coord:       Coord(tm.TopLeft.Row, tm.TopLeft.Col, tm.Horizontal),
		Score:       bestMove.Score,
		Description: tm.String(),
	}

	riddle := &Riddle{
		Board:    board.ToStrings(),
		Rack:     rack,
		Solution: solution,
		Analysis: analysis,
	}

	// Calculate the final ranking score for this candidate.
	rankScore := float64(bestMove.Score)
	rankScore += float64(len(tm.Covers)) * heuristics.NumCoversBonus
	rankScore += float64(bestMove.Score-secondBestScore) * heuristics.ScoreGapBonus
	if isBingo {
		rankScore += heuristics.BingoBonus
	}

	return &RiddleCandidate{
		Riddle: riddle,
		Score:  rankScore,
	}, nil
}

// GenerateRiddle orchestrates the generation and selection of the best riddle.
func GenerateRiddle(params GenerationParams, heuristics HeuristicConfig) (*Riddle, *Stats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), params.TimeLimit)
	defer cancel()

	var wg sync.WaitGroup
	candidateChan := make(chan *RiddleCandidate, 100)

	stats := &Stats{}

	// Spawn a configurable number of workers.
	numWorkers := params.NumWorkers
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for atomic.LoadInt64(&stats.Candidates) < int64(params.NumCandidates) {
				select {
				case <-ctx.Done():
					return
				default:
					candidate, err := generateCandidate(ctx, params, heuristics, stats)
					if err == nil && candidate != nil {
						candidateChan <- candidate
						atomic.AddInt64(&stats.Candidates, 1)
					}
				}
			}
		}()
	}

	// This goroutine will wait for all workers to finish and then close the channel.
	go func() {
		wg.Wait()
		close(candidateChan)
	}()

	// Collect and rank candidates as they come in.
	var bestCandidates []*RiddleCandidate
	for candidate := range candidateChan {
		bestCandidates = append(bestCandidates, candidate)
	}
	numCandidates := len(bestCandidates)

	// Log the rejection stats
	if numCandidates == 0 {
		return nil, nil, fmt.Errorf("could not generate a suitable riddle in the allotted time")
	}

	// Sort by our final rank score.
	sort.Slice(bestCandidates, func(i, j int) bool {
		return bestCandidates[i].Score > bestCandidates[j].Score
	})

	// Return the best scoring riddle.
	return bestCandidates[0].Riddle, stats, nil
}
