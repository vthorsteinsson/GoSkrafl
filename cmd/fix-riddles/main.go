package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

func main() {
	// Load .env files
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")

	// Parse flags
	startDate := flag.String("start-date", "2025-11-01", "Start date (YYYY-MM-DD)")
	endDate := flag.String("end-date", "2025-11-30", "End date (YYYY-MM-DD)")
	projectID := flag.String("project-id", os.Getenv("PROJECT_ID"), "Google Cloud project ID")
	namespace := flag.String("namespace", "", "Datastore namespace")
	dryRun := flag.Bool("dry-run", false, "Test mode without database writes")
	flag.Parse()

	if *projectID == "" {
		log.Fatal("Project ID is required")
	}

	// Connect to Datastore
	client, err := skrafl.NewDatastoreClient(*projectID, *namespace)
	if err != nil {
		log.Fatalf("Failed to create datastore client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// List riddles
	fmt.Printf("Fetching riddles from %s to %s...\n", *startDate, *endDate)
	riddles, err := client.ListRiddlesInRange(ctx, *startDate, *endDate)
	if err != nil {
		log.Fatalf("Failed to list riddles: %v", err)
	}

	fmt.Printf("Found %d riddles. Checking for coordinate mismatches...\n", len(riddles))

	fixedCount := 0
	for _, r := range riddles {
		var riddle skrafl.Riddle
		if err := json.Unmarshal([]byte(r.RiddleJSON), &riddle); err != nil {
			log.Printf("[%s] Failed to unmarshal JSON: %v", r.GetKey(), err)
			continue
		}

		// Extract correct coordinate from Description
		// Description format: "COORD WORD" (e.g., "H8 WORD")
		parts := strings.Split(riddle.Solution.Description, " ")
		if len(parts) < 2 {
			log.Printf("[%s] Invalid description format: '%s'", r.GetKey(), riddle.Solution.Description)
			continue
		}
		correctCoord := parts[0]

		if riddle.Solution.Coord != correctCoord {
			fmt.Printf("[%s] Fixing coord: '%s' -> '%s'\n", r.GetKey(), riddle.Solution.Coord, correctCoord)

			// Update the struct
			riddle.Solution.Coord = correctCoord

			// Serialize back to JSON
			bytes, err := json.Marshal(riddle)
			if err != nil {
				log.Printf("[%s] Failed to marshal JSON: %v", r.GetKey(), err)
				continue
			}
			r.RiddleJSON = string(bytes)

			if !*dryRun {
				// Save to Datastore
				if err := client.SaveRiddle(ctx, r, r.Date, r.Locale); err != nil {
					log.Printf("[%s] Failed to save riddle: %v", r.GetKey(), err)
				} else {
					fixedCount++
				}
			} else {
				log.Printf("[%s] Dry run: JSON would become %s", r.GetKey(), r.RiddleJSON)
				fixedCount++
			}
		}
	}

	if *dryRun {
		fmt.Printf("Dry run complete. Would have fixed %d riddles.\n", fixedCount)
	} else {
		fmt.Printf("Operation complete. Fixed %d riddles.\n", fixedCount)
	}
}
