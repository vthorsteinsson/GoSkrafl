// datastore.go
//
// Copyright (C) 2025 Vilhjálmur Þorsteinsson / Miðeind ehf.
//
// This file implements Google Cloud Datastore integration for riddle storage.

package skrafl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

// RiddleModel represents a riddle entity in Google Cloud Datastore
type RiddleModel struct {
	// The entity uses a composite key: "YYYY-MM-DD:locale" (e.g., "2025-01-15:is_IS")
	Date       string    `datastore:"date,noindex"`        // Date in YYYY-MM-DD format
	Locale     string    `datastore:"locale,noindex"`      // Locale code (e.g., "is_IS")
	RiddleJSON string    `datastore:"riddle_json,noindex"` // Complete riddle as JSON
	Created    time.Time `datastore:"created"`             // Indexed for time-based queries
	Version    int       `datastore:"version,noindex"`     // Schema version (currently 1)
}

// GetKey returns the composite key for this riddle
func (r *RiddleModel) GetKey() string {
	return fmt.Sprintf("%s:%s", r.Date, r.Locale)
}

// ParseKey splits a composite key into date and locale components
func ParseKey(key string) (date, locale string) {
	parts := strings.Split(key, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return key, "" // Fallback for malformed keys
}

// DatastoreClient handles all Google Cloud Datastore operations
type DatastoreClient struct {
	client    *datastore.Client
	namespace string
	kind      string
}

// NewDatastoreClient creates a new Datastore client
func NewDatastoreClient(projectID, namespace string) (*DatastoreClient, error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore client: %w", err)
	}

	return &DatastoreClient{
		client:    client,
		namespace: namespace,
		kind:      "RiddleModel",
	}, nil
}

// Close closes the Datastore client connection
func (dc *DatastoreClient) Close() error {
	if dc.client != nil {
		return dc.client.Close()
	}
	return nil
}

// makeKey creates a Datastore key with the appropriate namespace
func (dc *DatastoreClient) makeKey(keyName string) *datastore.Key {
	key := datastore.NameKey(dc.kind, keyName, nil)
	if dc.namespace != "" {
		key.Namespace = dc.namespace
	}
	return key
}

// SaveRiddle saves a riddle to Datastore with a composite key
func (dc *DatastoreClient) SaveRiddle(ctx context.Context, riddle *RiddleModel, date, locale string) error {
	keyName := fmt.Sprintf("%s:%s", date, locale)
	key := dc.makeKey(keyName)

	// Set metadata
	riddle.Date = date
	riddle.Locale = locale
	riddle.Created = time.Now().UTC()
	riddle.Version = 1

	// Save to Datastore (upsert - will overwrite if exists)
	_, err := dc.client.Put(ctx, key, riddle)
	if err != nil {
		return fmt.Errorf("failed to save riddle %s: %w", keyName, err)
	}

	return nil
}

// GetRiddle retrieves a riddle for a specific date and locale
func (dc *DatastoreClient) GetRiddle(ctx context.Context, date, locale string) (*RiddleModel, error) {
	keyName := fmt.Sprintf("%s:%s", date, locale)
	return dc.GetRiddleByKey(ctx, keyName)
}

// GetRiddleByKey retrieves a riddle by its composite key
func (dc *DatastoreClient) GetRiddleByKey(ctx context.Context, keyName string) (*RiddleModel, error) {
	key := dc.makeKey(keyName)

	var riddle RiddleModel
	err := dc.client.Get(ctx, key, &riddle)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("failed to get riddle %s: %w", keyName, err)
	}

	// Parse key to set date and locale if not already set
	if riddle.Date == "" || riddle.Locale == "" {
		riddle.Date, riddle.Locale = ParseKey(keyName)
	}

	return &riddle, nil
}

// ListRiddlesInRange retrieves all riddles in a date range
func (dc *DatastoreClient) ListRiddlesInRange(ctx context.Context, startDate, endDate string) ([]*RiddleModel, error) {
	// Query for all riddles with keys in the date range
	query := datastore.NewQuery(dc.kind).
		FilterField("__key__", ">=", dc.makeKey(startDate+":")).
		FilterField("__key__", "<=", dc.makeKey(endDate+":\xff")).
		Order("__key__")

	if dc.namespace != "" {
		query = query.Namespace(dc.namespace)
	}

	var riddles []*RiddleModel
	keys, err := dc.client.GetAll(ctx, query, &riddles)
	if err != nil {
		return nil, fmt.Errorf("failed to list riddles in range %s to %s: %w", startDate, endDate, err)
	}

	// Set date and locale from keys
	for i, key := range keys {
		riddles[i].Date, riddles[i].Locale = ParseKey(key.Name)
	}

	return riddles, nil
}

// ListRiddlesForDate retrieves all riddles for a specific date (all locales)
func (dc *DatastoreClient) ListRiddlesForDate(ctx context.Context, date string) ([]*RiddleModel, error) {
	return dc.ListRiddlesInRange(ctx, date, date)
}

// DeleteRiddle deletes a riddle for a specific date and locale
func (dc *DatastoreClient) DeleteRiddle(ctx context.Context, date, locale string) error {
	keyName := fmt.Sprintf("%s:%s", date, locale)
	key := dc.makeKey(keyName)

	err := dc.client.Delete(ctx, key)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return fmt.Errorf("failed to delete riddle %s: %w", keyName, err)
	}

	return nil
}

// SetFromRiddle sets the RiddleJSON field from a Riddle struct
func (rm *RiddleModel) SetFromRiddle(riddle *Riddle) error {
	// Marshal the riddle to JSON
	riddleJSON, err := json.Marshal(riddle)
	if err != nil {
		return fmt.Errorf("failed to marshal riddle to JSON: %w", err)
	}

	rm.RiddleJSON = string(riddleJSON)
	return nil
}
