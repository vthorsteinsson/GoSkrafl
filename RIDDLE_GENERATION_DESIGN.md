# Riddle Generation Design Plan

This document outlines the design and implementation plan for adding a "Riddle of the Day" generation feature to the GoSkrafl server.

## 1. Design Philosophy

The core of the feature is to treat riddle generation as a large-scale search problem. We will leverage Go's concurrency to generate and evaluate thousands of potential game states (a board and a player's rack) in parallel to find a "gem" that makes for a fun and challenging riddle for a human player.

A "good" riddle is defined by a set of measurable **heuristics**. The quality of a riddle is determined by the set of possible moves that can be made from that state.

### Key Heuristics for a Quality Riddle

1.  **Board State:**
    *   **Tile Count:** The board must have an adequate number of tiles to be interesting. We'll target a range, for example, 20-50 tiles. This avoids boards that are too open or too cluttered.

2.  **Move Possibilities:**
    *   **Total Moves:** A healthy number of total possible moves (e.g., 20-300) indicates a balanced and engaging position.
    *   **Best Move Score:** The highest-scoring move should be significantly high to be rewarding.
    *   **Score Gap:** A notable difference in score between the best and second-best moves makes the solution clearer and more satisfying.
    *   **Score Spread:** A high standard deviation in move scores implies a complex decision-making process for the player.
    *   **"Bingo" Bonus:** The best move should ideally be a "bingo" (using all 7 tiles). These are inherently spectacular and will be heavily prioritized.

## 2. The Generation Process

The process will be managed by an orchestrator that runs for a configurable time limit (e.g., 15 seconds) to ensure a timely response.

1.  **Initiation:** An API call triggers the riddle generation process.
2.  **Spawn Workers:** The orchestrator spawns a large number of concurrent "candidate generator" goroutines.
3.  **Generate Candidates (Parallel):** Each worker independently creates a plausible game state by:
    *   Starting a new game.
    *   Simulating a partial game by having two `HighScoreRobot` players play against each other for a random number of turns (e.g., 5-15). This produces varied and realistic board layouts.
    *   The resulting board and the current player's rack form a **candidate riddle**.
4.  **Analyze Candidates (Parallel):** Each worker analyzes its candidate:
    *   It generates all possible moves from the candidate state.
    *   It calculates all the key heuristics (tile count, total moves, best score, second-best score, score gap, bingo status, etc.).
5.  **Filter and Rank:**
    *   The orchestrator collects the analyzed candidates from the workers via a channel.
    *   It immediately discards candidates that fail to meet minimum criteria (e.g., too few moves).
    *   It maintains a sorted list of the top N (e.g., 10) candidates found so far, ranked by a weighted scoring formula. A potential formula could be: `RankScore = (BestMoveScore * 1.5) + ScoreGap + (IsBingo * 50)`.
6.  **Conclusion:** When the time limit is reached, the orchestrator stops the workers, selects the top-ranked candidate from its list, and returns it as the final riddle.

## 3. API Design

A new API endpoint will be added to `server.go`.

**Endpoint:** `POST /riddle`

A `POST` request is appropriate as it's an action that results in the creation of a new resource.

**Request Body (JSON):**
```json
{
  "locale": "is_IS",
  "boardType": "standard",
  "timeLimitSeconds": 15
}
```
*   `locale`: (Required) A string like "is_IS", "en_US", "en_GB", etc. This will be passed to the existing `decodeLocale()` function to select the correct dictionary and tile set.
*   `boardType`: (Optional) "standard" or "explo". Defaults to "standard".
*   `timeLimitSeconds`: (Optional) Maximum time for generation. Defaults to 15.

**Success Response (200 OK):**
A JSON object representing the chosen riddle. All letters are lowercase. Blank tiles are represented by `'?'`.
```json
{
  "board": [
    "...", "...", "...", ... // 15x15 string array of the board
  ],
  "rack": "st?ngur", // 7-tile rack as a string, '?' is a blank
  "solution": {
    "move": "gæs",
    "square": "8H", // "A1"-"O15" for horizontal, "1A"-"15O" for vertical
    "score": 88,
    "description": "gæs(88)"
  },
  "analysis": {
    "totalMoves": 124,
    "bestMoveScore": 88,
    "secondBestMoveScore": 42,
    "averageScore": 18.5,
    "isBingo": true
  }
}
```

**Error Response (503 Service Unavailable):**
Returned if no suitable riddle can be found within the time limit.
```json
{
  "error": "Could not generate a suitable riddle in the allotted time."
}
```

## 4. Implementation Plan

1.  **Create `riddle.go`:**
    *   Define a public `Riddle` struct for the API response.
    *   Define an internal `RiddleCandidate` struct to hold the riddle and its calculated heuristics for ranking.
    *   Define a `HeuristicConfig` struct to make tuning parameters (min/max tiles, etc.) easy.
    *   Add a utility function to format move coordinates from `(row, col, across)` to the "A1"/"1A" string format.

2.  **Implement Candidate Generator:**
    *   Create `generateCandidate(ctx context.Context, params GenerationParams) (*RiddleCandidate, error)`.
    *   This function will implement the partial game simulation and move analysis logic.

3.  **Implement Orchestrator:**
    *   Create the main public function `GenerateRiddle(params GenerationParams) (*Riddle, error)`.
    *   This function will manage the concurrent generation process using a `context` for the timeout, a `sync.WaitGroup` for workers, and a channel for results. It will also contain the filtering and ranking logic.

4.  **Update `server.go`:**
    *   Define a `RiddleRequest` struct to parse the incoming JSON body.
    *   Add a new handler `handleGenerateRiddle(w http.ResponseWriter, r *http.Request)`.
    *   This handler will parse the request, call `GenerateRiddle`, and serialize the resulting `Riddle` struct (or an error) to JSON.

5.  **Update `main/main.go`:**
    *   Register the new `/riddle` route with the HTTP server.
