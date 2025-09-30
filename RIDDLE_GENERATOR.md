# Riddle Generator for Google Cloud Datastore

This document specifies the design and implementation plan for a Go program that generates riddles in bulk and stores them in Google Cloud Datastore for use by the Netskrafl/Gáta Dagsins (Riddle of the Day) feature.

## 1. Overview

The riddle generator is a command-line utility that:
- Generates high-quality crossword riddles for a specified date range
- Stores riddles in Google Cloud Datastore using the same schema as Netskrafl
- Supports multiple locales per date (Icelandic, English, Polish, Norwegian)
- Provides options for dry-run and quality thresholds
- Integrates with Google Cloud authentication
- Automatically overwrites existing riddles when regenerating

## 2. Command-Line Interface

### Usage
```bash
# Generate riddles for a date range for a single locale
./main -generate-riddles -start-date 2025-01-01 -end-date 2025-12-31 -locale is_IS

# Generate riddles for multiple locales (comma-separated)
./main -generate-riddles \
  -start-date 2025-01-01 \
  -end-date 2025-01-31 \
  -locale is_IS,en_US,en_GB,pl_PL

# Generate riddles with custom parameters
./main -generate-riddles \
  -start-date 2025-01-01 \
  -end-date 2025-01-31 \
  -locale en_US \
  -project-id netskrafl-live \
  -namespace explo \
  -workers 16 \
  -time-limit 30 \
  -candidates 100 \
  -min-score 50

# Dry run to test without writing to database
./main -generate-riddles \
  -start-date 2025-01-01 \
  -end-date 2025-01-07 \
  -locale is_IS,en_US \
  -dry-run
```

### Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-generate-riddles` | bool | false | Enable riddle generation mode |
| `-start-date` | string | required | Start date in YYYY-MM-DD format |
| `-end-date` | string | required | End date in YYYY-MM-DD format |
| `-locale` | string | "is_IS" | Locale(s) for riddle generation (comma-separated for multiple) |
| `-project-id` | string | env:PROJECT_ID | Google Cloud project ID |
| `-namespace` | string | "" | Datastore namespace (e.g., "explo") |
| `-workers` | int | NumCPU | Number of worker goroutines per riddle |
| `-time-limit` | int | 20 | Time limit in seconds per riddle |
| `-candidates` | int | 100 | Number of candidates to generate per riddle |
| `-min-score` | int | 40 | Minimum acceptable best move score |
| `-dry-run` | bool | false | Test mode without database writes |
| `-p` | int | 8080 | Port for HTTP server mode |

## 3. Datastore Entity Design

### RiddleModel Entity

The entity stores the complete riddle as a JSON string, matching the format returned by the /riddle endpoint:

```go
type RiddleModel struct {
    // The entity uses a composite key: "YYYY-MM-DD:locale" (e.g., "2025-01-15:is_IS")
    // Date and Locale are stored as regular properties for querying purposes
    Date       string    `datastore:"date,noindex"`         // Date in YYYY-MM-DD format
    Locale     string    `datastore:"locale,noindex"`       // Locale code (e.g., "is_IS")
    RiddleJSON string    `datastore:"riddle_json,noindex"`  // Complete riddle as JSON
    Created    time.Time `datastore:"created"`              // Indexed for time-based queries
    Version    int       `datastore:"version,noindex"`      // Schema version (currently 1)
}

// Helper methods for Go implementation
func (r *RiddleModel) GetKey() string {
    return fmt.Sprintf("%s:%s", r.Date, r.Locale)
}

func ParseKey(key string) (date, locale string) {
    parts := strings.Split(key, ":")
    if len(parts) == 2 {
        return parts[0], parts[1]
    }
    return key, "" // Fallback for malformed keys
}
```

### Key Structure
- **Kind**: "RiddleModel"
- **Key Name**: Composite string in format "YYYY-MM-DD:locale" (e.g., "2025-01-15:is_IS")
- **Namespace**: Configurable (e.g., "explo" for Explo app)

### Multi-Locale Support
Each date can have multiple riddles, one per locale. The composite key ensures:
- Fast direct lookups for specific date+locale combinations
- Ability to query all riddles for a date using key prefix queries
- No conflicts between different locale riddles for the same date

### Why Store Date and Locale as Properties?
Even though date and locale are encoded in the key, storing them as properties allows:
- Easier debugging and data inspection in Datastore console
- Potential future queries without parsing keys
- Compatibility with Python code that may expect these fields

## 4. Implementation Components

### 4.1 Main Program Extension (main/main.go)

Add riddle generation mode to the existing main program:

```go
func main() {
    // Existing flags...
    generateRiddles := flag.Bool("generate-riddles", false, "Generate riddles for date range")
    startDate := flag.String("start-date", "", "Start date (YYYY-MM-DD)")
    endDate := flag.String("end-date", "", "End date (YYYY-MM-DD)")
    // ... other riddle generation flags ...
    
    flag.Parse()
    
    if *generateRiddles {
        runRiddleGenerator(/* parsed flags */)
        return
    }
    
    // Existing main program logic...
}
```

### 4.2 Riddle Generator Module (riddlegen.go)

Core module for batch riddle generation:

```go
package skrafl

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"
)

type RiddleGenerator struct {
    config   RiddleGeneratorConfig
    client   *DatastoreClient
    stats    *BatchStats
    statsMux sync.Mutex
}

func NewRiddleGenerator(config RiddleGeneratorConfig) (*RiddleGenerator, error)
func (rg *RiddleGenerator) GenerateForDateRange(start, end time.Time, locales []string) error
func (rg *RiddleGenerator) generateForDateLocale(job GenerationJob) GenerationResult
func (rg *RiddleGenerator) Close() error
```

### 4.3 Datastore Integration (datastore.go)

Handles Google Cloud Datastore operations:

```go
package skrafl

import (
    "cloud.google.com/go/datastore"
)

type DatastoreClient struct {
    client    *datastore.Client
    namespace string
}

func NewDatastoreClient(projectID, namespace string) (*DatastoreClient, error)
func (dc *DatastoreClient) SaveRiddle(ctx context.Context, riddle *RiddleModel, date, locale string) error
func (dc *DatastoreClient) GetRiddle(date, locale string) (*RiddleModel, error)
func (dc *DatastoreClient) GetRiddleByKey(key string) (*RiddleModel, error)
func (dc *DatastoreClient) ListRiddlesForDate(date string) ([]*RiddleModel, error)
func (dc *DatastoreClient) ListRiddlesInRange(start, end string) ([]*RiddleModel, error)
func (dc *DatastoreClient) DeleteRiddle(date, locale string) error
```

## 5. Generation Algorithm

### 5.1 Date Range Processing

The generator automatically parallelizes riddle generation to maximize CPU utilization:

```
1. Parse and validate date range
2. Parse locale list (comma-separated)
3. Generate list of date+locale combinations to process
4. Create worker pool sized to number of CPU cores
5. Distribute date+locale pairs across workers:
   a. Each worker generates riddles for assigned date+locale combinations
   b. Generate riddle with locale-specific quality requirements
   c. Save to Datastore with composite key (unless dry-run)
   d. Report progress to main thread
6. Aggregate and display final statistics per locale
```

The system automatically determines optimal parallelization:
- For date+locale combinations: Uses all available CPU cores
- For riddle generation per combination: Also uses available cores via existing riddle engine
- Balances memory usage and CPU utilization

Example: Generating 30 days × 4 locales = 120 riddles will distribute across all cores

### 5.2 Quality Assurance

Each generated riddle must meet minimum quality criteria:
- Best move score ≥ configured minimum
- Sufficient move diversity (20+ valid moves)
- Solution word in common words dictionary (for Icelandic)
- Board complexity within acceptable range

### 5.3 Current Implementation Note

The current implementation attempts generation once per date+locale combination. The quality filtering happens within the `GenerateRiddle` function itself through the heuristics system. If generation fails, it's logged and the system continues with the next combination. Future enhancement could add retry logic with adjusted parameters.

## 6. Progress Tracking and Logging

### 6.1 Progress Output

```
Generating riddles for 2025-01-01 to 2025-01-31 (31 days)
Locales: is_IS, en_US, Workers: 16, Time limit: 20s per riddle

[2025-01-01:is_IS] Generating... Done (best score: 92, bingo: yes, attempts: 1)
[2025-01-01:en_US] Generating... Done (best score: 78, bingo: no, attempts: 1)
[2025-01-02:is_IS] Generating... Done (best score: 67, bingo: no, attempts: 2)
[2025-01-02:en_US] Generating... Done (best score: 85, bingo: yes, attempts: 1)
[2025-01-03:is_IS] Generating... Done (best score: 85, bingo: yes, attempts: 1) [replaced]
[2025-01-03:en_US] Generating... Failed (no suitable riddle after 3 attempts)
...

Summary by locale:
  is_IS:
  - Generated: 30 riddles
  - Failed: 1
  - Average best score: 74.3
  - Bingos: 12 (40.0%)
  
  en_US:
  - Generated: 29 riddles
  - Failed: 2
  - Average best score: 68.5
  - Bingos: 8 (27.6%)

Total: 59 riddles generated, 3 failed
Total time: 18m 45s
```

### 6.2 Detailed Logging

- Log file: `riddle_generation_YYYYMMDD_HHMMSS.log`
- Contents: Detailed statistics for each riddle, errors, and warnings

## 7. Error Handling

### 7.1 Recoverable Errors
- Individual riddle generation failure → Log and continue
- Temporary Datastore errors → Retry with exponential backoff
- Quality criteria not met → Try with adjusted parameters

### 7.2 Fatal Errors
- Invalid date range
- Datastore authentication failure
- Project/namespace not found
- Invalid locale

## 8. Testing Strategy

### 8.1 Unit Tests
- Date range parsing and validation
- Riddle quality evaluation
- Datastore entity serialization

### 8.2 Integration Tests
- Datastore connection and operations
- End-to-end generation for single date
- Batch generation with parallelism

### 8.3 Manual Testing
```bash
# Test with dry run
./main -generate-riddles -start-date 2025-01-01 -end-date 2025-01-01 -dry-run

# Test with local Datastore emulator
gcloud beta emulators datastore start
export DATASTORE_EMULATOR_HOST=localhost:8081
./main -generate-riddles -start-date 2025-01-01 -end-date 2025-01-07
```

## 9. Authentication and Deployment

### 9.1 Authentication Methods

The application automatically loads environment variables from `.env.local` file if present, then uses Google's Application Default Credentials (ADC) which finds credentials in this order:

1. **GOOGLE_APPLICATION_CREDENTIALS environment variable** (if set)
2. **gcloud auth application-default** credentials
3. **Google Cloud metadata service** (when running on GCP)
4. **gcloud auth** user credentials

#### Using .env.local file (Recommended for local development)

Create a `.env.local` file in the project root:
```bash
# .env.local
PROJECT_ID=my-gcp-project
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

Then simply run:
```bash
./main -generate-riddles -start-date 2025-01-01 -end-date 2025-01-31
```

#### Alternative methods for local development:
```bash
# Option 1: Service account (recommended for automation)
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
./main -generate-riddles -start-date 2025-01-01 -end-date 2025-01-31

# Option 2: User credentials (for development)
gcloud auth application-default login
./main -generate-riddles -start-date 2025-01-01 -end-date 2025-01-31
```

### 9.2 Service Account Setup

```bash
# Create service account
gcloud iam service-accounts create riddle-generator \
  --display-name="Riddle Generator Service Account"

# Grant necessary permissions
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:riddle-generator@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/datastore.user"

# Create and download key
gcloud iam service-accounts keys create riddle-generator-key.json \
  --iam-account=riddle-generator@PROJECT_ID.iam.gserviceaccount.com
```

### 9.3 Scheduled Generation

Use Cloud Scheduler or cron to run periodically:

```bash
# Generate riddles for next month for all supported locales
0 0 1 * * ./main -generate-riddles \
  -start-date $(date -d "+1 month" +%Y-%m-01) \
  -end-date $(date -d "+2 months -1 day" +%Y-%m-%d) \
  -locale is_IS,en_US,en_GB,pl_PL,nb_NO \
  -project-id netskrafl-live
```

### 9.4 Monitoring

- Track generation success rate
- Monitor average generation time per riddle
- Alert on consecutive failures
- Track riddle quality metrics over time

## 10. Implementation Status

### ✅ Completed
- [x] Add command-line flags to main.go (including -p port flag)
- [x] Create riddlegen.go with full implementation
- [x] Implement date range parsing and validation
- [x] Create datastore.go module with complete Datastore integration
- [x] Implement RiddleModel entity with composite keys
- [x] Add save/retrieve/list operations
- [x] Integrate with existing riddle generation engine
- [x] Implement quality checks via heuristics
- [x] Implement parallel processing with worker pools
- [x] Add comprehensive progress tracking and statistics
- [x] Support for multi-locale generation
- [x] Dry-run mode for testing

### 🔄 Ready for Testing
- [ ] Test with Datastore emulator
- [ ] Write unit and integration tests
- [ ] Test in staging environment

### 📋 Deployment Tasks
- [ ] Set up service accounts
- [ ] Deploy to production
- [ ] Set up monitoring
- [ ] Configure Cloud Scheduler for automated generation

## 11. Python Compatibility

### Reading Multi-Locale Riddles in Python

The composite key structure allows Python code to easily access riddles by date and locale:

```python
from google.cloud import ndb
import json
from datetime import date

class RiddleModel(ndb.Model):
    """Riddle entity compatible with Go-generated data."""
    date = ndb.StringProperty(indexed=False)
    locale = ndb.StringProperty(indexed=False)
    riddle_json = ndb.TextProperty(indexed=False)
    created = ndb.DateTimeProperty(indexed=True)
    version = ndb.IntegerProperty(indexed=False, default=1)

    @classmethod
    def get_riddle(cls, date_str: str, locale: str) -> Optional['RiddleModel']:
        """Get a riddle for a specific date and locale."""
        key = f"{date_str}:{locale}"
        return cls.get_by_id(key)

    @classmethod
    def get_riddles_for_date(cls, date_str: str) -> List['RiddleModel']:
        """Get all riddles for a specific date (all locales)."""
        # Query by key prefix - requires key range query
        query = cls.query()
        query = query.filter(cls.key >= ndb.Key(cls, f"{date_str}:"))
        query = query.filter(cls.key < ndb.Key(cls, f"{date_str}:ÿ"))
        return query.fetch()

    @property
    def riddle(self):
        """Parse and return the riddle data."""
        return json.loads(self.riddle_json) if self.riddle_json else None

    @property
    def date_str(self):
        """Extract date from the key."""
        key_name = self.key.id()
        return key_name.split(':')[0] if ':' in key_name else key_name

# Usage examples:
# Get Icelandic riddle for January 15, 2025
riddle = RiddleModel.get_riddle("2025-01-15", "is_IS")
if riddle:
    data = riddle.riddle
    print(f"Best move: {data['solution']['move']} for {data['solution']['score']} points")

# Get all riddles for a date
riddles = RiddleModel.get_riddles_for_date("2025-01-15")
for r in riddles:
    print(f"Locale: {r.locale}, Created: {r.created}")
```

### Key Format Consistency

Both Go and Python must use the same composite key format:
- **Format**: `"YYYY-MM-DD:locale"`
- **Examples**: `"2025-01-15:is_IS"`, `"2025-01-15:en_US"`, `"2025-01-15:pl_PL"`
- **Delimiter**: Colon (`:`) separates date from locale

## 12. Dependencies

Add to go.mod:
```go
require (
    cloud.google.com/go/datastore v1.15.0
    github.com/joho/godotenv v1.5.1
    // existing dependencies...
)
```

## 13. Configuration File (Optional)

For complex deployments, support configuration file:

```yaml
# riddle-config.yaml
generation:
  locales:
    - locale: is_IS
      minScore: 50
      solutionFilter: common_words
    - locale: en_US
      minScore: 40
  defaults:
    workers: 16
    timeLimit: 20
    candidates: 100

datastore:
  projectId: my-gcp-project
  namespace: my-namespace
  
schedule:
  - locale: is_IS
    frequency: daily
    advanceDays: 7
  - locale: en_US
    frequency: weekly
    advanceDays: 30
```

## 14. Future Enhancements

- Web UI for manual riddle review and editing
- A/B testing different heuristics
- Machine learning for quality prediction
- Automatic quality improvement over time
- Multi-locale generation in single run
- Riddle difficulty levels (easy/medium/hard)
- Theme-based riddles (holidays, events)
- User feedback integration for quality improvement
