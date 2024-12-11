// go-app/main.go
// App Engine main package for GoSkrafl server
// Copyright (C) 2024 Vilhjálmur Þorsteinsson / Miðeind ehf.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

// Bearer authorization token, if any
var ACCESS_KEY string

// Corresponding Authorization header (or "" if no auth required)
var AUTH_HEADER string

// Allowed access control (CORS) origins
var ALLOWED_ORIGINS string = "*" // Default to all origins allowed

func validate(w http.ResponseWriter, r *http.Request, req any) bool {
	// Set CORS headers
	header := w.Header()
	header.Set("Access-Control-Allow-Origin", ALLOWED_ORIGINS)
	header.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		// Returning false simply causes the handler to return the response headers
		return false
	}

	// We only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return false
	}
	// Check for a bearer authorization token,
	// which must match the environment variable
	// ACCESS_KEY, if present
	if AUTH_HEADER != "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != AUTH_HEADER {
			http.Error(w,
				fmt.Sprintf(
					"Authorization header mismatch: got '%s'",
					authHeader,
				),
				http.StatusUnauthorized,
			)
			return false
		}
	}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func movesHandler(w http.ResponseWriter, r *http.Request) {
	var req skrafl.MovesRequest
	if !validate(w, r, &req) {
		return
	}
	skrafl.HandleMovesRequest(w, req)
}

func wordcheckHandler(w http.ResponseWriter, r *http.Request) {
	var req skrafl.WordCheckRequest
	if !validate(w, r, &req) {
		return
	}
	skrafl.HandleWordCheckRequest(w, req)
}

func warmupHandler(w http.ResponseWriter, r *http.Request) {
	// No concrete action required
	log.Println("Warmup request received")
}

func main() {
	// Log to Google App Engine
	log.SetOutput(os.Stderr)
	log.Printf("Moves service starting, Go version %s", runtime.Version())
	// Figure out the authorization header, if required
	ACCESS_KEY := os.Getenv("ACCESS_KEY")
	if ACCESS_KEY != "" {
		AUTH_HEADER = "Bearer " + ACCESS_KEY
	}
	// Set up a dummy warmup handler
	http.HandleFunc("/_ah/warmup", warmupHandler)
	// Set up the actual service handlers
	http.HandleFunc("/moves", movesHandler)
	http.HandleFunc("/wordcheck", wordcheckHandler)
	// Establish the port number to listen on, defaulting to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	// Establish allowed CORS origins
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		log.Printf("Allowed CORS origins: %s", origins)
		ALLOWED_ORIGINS = origins
	} else {
		log.Printf("No ALLOWED_ORIGINS specified, allowing all")
	}
	// Start the server loop
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
