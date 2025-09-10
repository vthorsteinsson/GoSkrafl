// riddlegen.go
//
// Copyright (C) 2025 Vilhjálmur Þorsteinsson / Miðeind ehf.
//
// This file implements batch riddle generation for multiple dates and locales.

package skrafl

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// RiddleGeneratorConfig holds configuration for the batch riddle generator
type RiddleGeneratorConfig struct {
	ProjectID     string
	Namespace     string
	Workers       int
	TimeLimit     time.Duration
	NumCandidates int
	MinScore      int
	DryRun        bool
}

// RiddleGenerator orchestrates batch riddle generation
type RiddleGenerator struct {
	config   RiddleGeneratorConfig
	client   *DatastoreClient
	stats    *BatchStats
	statsMux sync.Mutex
}

// BatchStats tracks statistics across all riddle generation
type BatchStats struct {
	TotalAttempted int
	TotalGenerated int
	TotalReplaced  int
	TotalFailed    int
	ByLocale       map[string]*LocaleStats
	StartTime      time.Time
	EndTime        time.Time
}

// LocaleStats tracks statistics for a specific locale
type LocaleStats struct {
	Attempted      int
	Generated      int
	Replaced       int
	Failed         int
	TotalScore     int
	TotalBingos    int
	FailureReasons map[string]int
}

// GenerationJob represents a single date+locale combination to generate
type GenerationJob struct {
	Date   time.Time
	Locale string
}

// GenerationResult holds the result of generating a single riddle
type GenerationResult struct {
	Job      GenerationJob
	Riddle   *Riddle
	Stats    *Stats
	Replaced bool
	Error    error
}

// NewRiddleGenerator creates a new batch riddle generator
func NewRiddleGenerator(config RiddleGeneratorConfig) (*RiddleGenerator, error) {
	var client *DatastoreClient
	var err error

	if !config.DryRun {
		client, err = NewDatastoreClient(config.ProjectID, config.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create datastore client: %w", err)
		}
	}

	return &RiddleGenerator{
		config: config,
		client: client,
		stats: &BatchStats{
			ByLocale: make(map[string]*LocaleStats),
		},
	}, nil
}

// Close closes the generator and its resources
func (rg *RiddleGenerator) Close() error {
	if rg.client != nil {
		return rg.client.Close()
	}
	return nil
}

// GenerateForDateRange generates riddles for all dates in range and all specified locales
func (rg *RiddleGenerator) GenerateForDateRange(startDate, endDate time.Time, locales []string) error {
	rg.stats.StartTime = time.Now()

	// Generate list of jobs (date+locale combinations)
	jobs := rg.createJobs(startDate, endDate, locales)
	totalJobs := len(jobs)

	// Initialize locale stats
	for _, locale := range locales {
		rg.stats.ByLocale[locale] = &LocaleStats{
			FailureReasons: make(map[string]int),
		}
	}

	// Print header
	days := int(endDate.Sub(startDate).Hours()/24) + 1
	fmt.Printf("\nGenerating riddles for %s to %s (%d days)\n",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), days)
	fmt.Printf("Locales: %v, Workers: %d, Time limit: %ds per riddle\n\n",
		locales, rg.config.Workers, int(rg.config.TimeLimit.Seconds()))

	// Create worker pool
	jobChan := make(chan GenerationJob, totalJobs)
	resultChan := make(chan GenerationResult, totalJobs)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < rg.config.Workers; i++ {
		wg.Add(1)
		go rg.worker(&wg, jobChan, resultChan)
	}

	// Send all jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Start result processor
	doneChan := make(chan bool)
	go rg.processResults(resultChan, totalJobs, doneChan)

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Wait for result processor to finish
	<-doneChan

	rg.stats.EndTime = time.Now()
	rg.printSummary()

	return nil
}

// createJobs generates all date+locale combinations
func (rg *RiddleGenerator) createJobs(startDate, endDate time.Time, locales []string) []GenerationJob {
	var jobs []GenerationJob

	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		for _, locale := range locales {
			jobs = append(jobs, GenerationJob{
				Date:   date,
				Locale: locale,
			})
		}
	}

	return jobs
}

// worker processes generation jobs
func (rg *RiddleGenerator) worker(wg *sync.WaitGroup, jobChan <-chan GenerationJob, resultChan chan<- GenerationResult) {
	defer wg.Done()

	for job := range jobChan {
		result := rg.generateForDateLocale(job)
		resultChan <- result
	}
}

// generateForDateLocale generates a riddle for a specific date and locale
func (rg *RiddleGenerator) generateForDateLocale(job GenerationJob) GenerationResult {
	dateStr := job.Date.Format("2006-01-02")
	result := GenerationResult{Job: job}

	// Check if riddle already exists (to track replacements)
	var existingRiddle *RiddleModel
	if rg.client != nil {
		ctx := context.Background()
		existingRiddle, _ = rg.client.GetRiddle(ctx, dateStr, job.Locale)
	}
	result.Replaced = existingRiddle != nil

	// Get DAWG and tileset for locale
	dawg, tileSet, err := decodeLocale(job.Locale, "standard")
	if err != nil {
		result.Error = fmt.Errorf("invalid locale %s: %w", job.Locale, err)
		return result
	}

	// Set up generation parameters
	params := GenerationParams{
		Locale:        job.Locale,
		BoardType:     "standard",
		Dawg:          dawg,
		TileSet:       tileSet,
		TimeLimit:     rg.config.TimeLimit,
		NumWorkers:    rg.config.Workers,
		NumCandidates: rg.config.NumCandidates,
	}

	// Select appropriate heuristics for the locale
	heuristics := DefaultHeuristics
	if job.Locale == "is" || job.Locale == "is_IS" {
		heuristics = IcelandicHeuristics
	}
	heuristics.MinBestScore = rg.config.MinScore

	// Generate the riddle
	riddle, stats, err := GenerateRiddle(params, heuristics)
	if err != nil {
		result.Error = err
		result.Stats = stats
		return result
	}

	result.Riddle = riddle
	result.Stats = stats

	// Save to Datastore if not dry-run
	if !rg.config.DryRun && rg.client != nil {
		model := &RiddleModel{}
		if err := model.SetFromRiddle(riddle); err != nil {
			result.Error = fmt.Errorf("failed to convert riddle: %w", err)
			return result
		}

		ctx := context.Background()
		if err := rg.client.SaveRiddle(ctx, model, dateStr, job.Locale); err != nil {
			result.Error = fmt.Errorf("failed to save riddle: %w", err)
			return result
		}
	}

	return result
}

// processResults collects and displays results as they come in
func (rg *RiddleGenerator) processResults(resultChan <-chan GenerationResult, totalJobs int, doneChan chan<- bool) {
	processed := 0

	for result := range resultChan {
		processed++
		rg.updateStats(result)
		rg.printProgress(result, processed, totalJobs)
	}

	doneChan <- true
}

// updateStats updates the batch statistics with a result
func (rg *RiddleGenerator) updateStats(result GenerationResult) {
	rg.statsMux.Lock()
	defer rg.statsMux.Unlock()

	localeStats := rg.stats.ByLocale[result.Job.Locale]
	localeStats.Attempted++
	rg.stats.TotalAttempted++

	if result.Error != nil {
		localeStats.Failed++
		rg.stats.TotalFailed++
		// Track failure reason
		reason := "unknown"
		if result.Stats != nil {
			if result.Stats.TooFewMoves > 0 {
				reason = "too_few_moves"
			} else if result.Stats.TooLowBestScore > 0 {
				reason = "too_low_score"
			} else if result.Stats.WordNotCommon > 0 {
				reason = "word_not_common"
			} else if result.Stats.DoubleTripleWord > 0 {
				reason = "double_triple_word"
			} else if result.Stats.NoValidMove > 0 {
				reason = "no_valid_move"
			}
		}
		localeStats.FailureReasons[reason]++
	} else {
		localeStats.Generated++
		rg.stats.TotalGenerated++

		if result.Replaced {
			localeStats.Replaced++
			rg.stats.TotalReplaced++
		}

		if result.Riddle != nil {
			localeStats.TotalScore += result.Riddle.Analysis.BestMoveScore
			if result.Riddle.Analysis.IsBingo {
				localeStats.TotalBingos++
			}
		}
	}
}

// printProgress prints progress for a single result
func (rg *RiddleGenerator) printProgress(result GenerationResult, processed, total int) {
	dateStr := result.Job.Date.Format("2006-01-02")
	progressStr := fmt.Sprintf("[%d/%d] [%s:%s] ", processed, total, dateStr, result.Job.Locale)

	if rg.config.DryRun {
		progressStr += "[DRY-RUN] "
	}

	if result.Error != nil {
		fmt.Printf("%sFailed (%v)\n", progressStr, result.Error)
	} else {
		riddle := result.Riddle
		bingoStr := ""
		if riddle.Analysis.IsBingo {
			bingoStr = ", bingo: yes"
		}
		replacedStr := ""
		if result.Replaced {
			replacedStr = " [replaced]"
		}
		fmt.Printf("%sDone (best score: %d%s)%s\n",
			progressStr, riddle.Analysis.BestMoveScore, bingoStr, replacedStr)
	}
}

// printSummary prints the final statistics summary
func (rg *RiddleGenerator) printSummary() {
	duration := rg.stats.EndTime.Sub(rg.stats.StartTime)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Summary by locale:")

	for locale, stats := range rg.stats.ByLocale {
		if stats.Attempted == 0 {
			continue
		}

		fmt.Printf("\n  %s:\n", locale)
		fmt.Printf("  - Generated: %d riddles\n", stats.Generated)
		if stats.Replaced > 0 {
			fmt.Printf("  - Replaced: %d existing riddles\n", stats.Replaced)
		}
		if stats.Failed > 0 {
			fmt.Printf("  - Failed: %d\n", stats.Failed)
			if len(stats.FailureReasons) > 0 {
				fmt.Printf("    Reasons: ")
				first := true
				for reason, count := range stats.FailureReasons {
					if !first {
						fmt.Printf(", ")
					}
					fmt.Printf("%s (%d)", reason, count)
					first = false
				}
				fmt.Println()
			}
		}

		if stats.Generated > 0 {
			avgScore := float64(stats.TotalScore) / float64(stats.Generated)
			bingoPercent := float64(stats.TotalBingos) * 100.0 / float64(stats.Generated)
			fmt.Printf("  - Average best score: %.1f\n", avgScore)
			fmt.Printf("  - Bingos: %d (%.1f%%)\n", stats.TotalBingos, bingoPercent)
		}
	}

	fmt.Printf("\nTotal: %d riddles generated, %d failed\n",
		rg.stats.TotalGenerated, rg.stats.TotalFailed)
	fmt.Printf("Total time: %s\n", duration.Round(time.Second))

	if rg.config.DryRun {
		fmt.Println("\n[DRY-RUN MODE: No riddles were saved to the database]")
	}
}
