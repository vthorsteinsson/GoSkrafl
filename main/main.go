// main.go
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.

// Example main program for exercising the skrafl module

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

// GameConstructor is a function that returns the type of Game we want
type GameConstructor func(boardType string) (*skrafl.Game, error)

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
	game, err := gameConstructor(boardType)
	if err != nil {
		p("Error creating game: %v\n", err)
		return 0, 0
	}
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

func riddleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req skrafl.RiddleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	skrafl.HandleGenerateRiddle(w, req)
}

func runServer(port int) {
	http.HandleFunc("/moves", movesHandler)
	http.HandleFunc("/wordcheck", wordcheckHandler)
	http.HandleFunc("/riddle", riddleHandler)
	fmt.Printf("Starting HTTP server on port %d...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func runRiddleGenerator(
	startDate, endDate string,
	localeList string,
	projectID string,
	namespace string,
	workers int,
	timeLimit int,
	candidates int,
	minScore int,
	dryRun bool,
) {
	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		log.Fatalf("Invalid start date: %v", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		log.Fatalf("Invalid end date: %v", err)
	}
	if end.Before(start) {
		log.Fatalf("End date must be after start date")
	}

	// Parse locales
	locales := strings.Split(localeList, ",")
	for i, loc := range locales {
		locales[i] = strings.TrimSpace(loc)
	}

	// Create riddle generator config
	config := skrafl.RiddleGeneratorConfig{
		ProjectID:     projectID,
		Namespace:     namespace,
		Workers:       workers,
		TimeLimit:     time.Duration(timeLimit) * time.Second,
		NumCandidates: candidates,
		MinScore:      minScore,
		DryRun:        dryRun,
	}

	// Create and run the generator
	generator, err := skrafl.NewRiddleGenerator(config)
	if err != nil {
		log.Fatalf("Failed to create riddle generator: %v", err)
	}
	defer generator.Close()

	if err := generator.GenerateForDateRange(start, end, locales); err != nil {
		log.Fatalf("Failed to generate riddles: %v", err)
	}
}

func main() {
	// Load .env files in order of precedence (later files override earlier ones)
	_ = godotenv.Load(".env")       // Load defaults (safe to commit)
	_ = godotenv.Load(".env.local") // Load local overrides (never commit)
	
	// Existing flags
	server := flag.Bool("s", false, "Run as a HTTP server")
	port := flag.Int("p", 8080, "Port for HTTP server")
	dict := flag.String("d", "ice", "Dictionary to use (otcwl, sowpods, osps, ice)")
	boardType := flag.String("b", "standard", "Board type (standard, explo)")
	num := flag.Int("n", 10, "Number of games to simulate")
	quiet := flag.Bool("q", false, "Suppress output of game state and moves")

	// Riddle generation flags
	generateRiddles := flag.Bool("generate-riddles", false, "Generate riddles for date range")
	startDate := flag.String("start-date", "", "Start date (YYYY-MM-DD)")
	endDate := flag.String("end-date", "", "End date (YYYY-MM-DD)")
	locale := flag.String("locale", "is_IS", "Locale(s) for riddle generation (comma-separated)")
	projectID := flag.String("project-id", os.Getenv("PROJECT_ID"), "Google Cloud project ID")
	namespace := flag.String("namespace", "", "Datastore namespace")
	workers := flag.Int("workers", 0, "Number of workers (0 = NumCPU)")
	timeLimit := flag.Int("time-limit", 20, "Time limit in seconds per riddle")
	candidates := flag.Int("candidates", 100, "Number of candidates to generate per riddle")
	minScore := flag.Int("min-score", 40, "Minimum acceptable best move score")
	dryRun := flag.Bool("dry-run", false, "Test mode without database writes")

	flag.Parse()

	// Handle server mode
	if *server {
		if *generateRiddles {
			log.Fatal("Cannot use -s (server) and -generate-riddles together")
		}
		// Run a HTTP server
		runServer(*port)
		return
	}

	// Handle riddle generation mode
	if *generateRiddles {
		if *startDate == "" || *endDate == "" {
			log.Fatal("Both -start-date and -end-date are required for riddle generation")
		}
		if *projectID == "" {
			log.Fatal("Project ID is required (set PROJECT_ID env var or use -project-id flag)")
		}
		numWorkers := *workers
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU()
		}
		runRiddleGenerator(
			*startDate, *endDate,
			*locale,
			*projectID,
			*namespace,
			numWorkers,
			*timeLimit,
			*candidates,
			*minScore,
			*dryRun,
		)
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
