package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

func main() {
	// Load .env files in order of precedence (later files override earlier ones)
	_ = godotenv.Load(".env")       // Load defaults (safe to commit)
	_ = godotenv.Load(".env.local") // Load local overrides (never commit)

	// Parse flags
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

	// Validate required flags
	if *startDate == "" || *endDate == "" {
		log.Fatal("Both -start-date and -end-date are required")
	}
	if *projectID == "" {
		log.Fatal("Project ID is required (set PROJECT_ID env var or use -project-id flag)")
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Invalid start date: %v", err)
	}
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("Invalid end date: %v", err)
	}
	if end.Before(start) {
		log.Fatalf("End date must be after start date")
	}

	// Parse locales
	locales := strings.Split(*locale, ",")
	for i, loc := range locales {
		locales[i] = strings.TrimSpace(loc)
	}

	// Determine number of workers
	numWorkers := *workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	// Create riddle generator config
	config := skrafl.RiddleGeneratorConfig{
		ProjectID:     *projectID,
		Namespace:     *namespace,
		Workers:       numWorkers,
		TimeLimit:     time.Duration(*timeLimit) * time.Second,
		NumCandidates: *candidates,
		MinScore:      *minScore,
		DryRun:        *dryRun,
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
