// go-app/main.go
// App Engine main package for GoSkrafl server
// Copyright (C) 2023 Vilhjálmur Þorsteinsson / Miðeind ehf.

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	skrafl "github.com/vthorsteinsson/GoSkrafl"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req skrafl.SkraflRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Not valid JSON
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	skrafl.HandleRequest(w, req)
}

func warmup(w http.ResponseWriter, r *http.Request) {
	// No concrete action required
	log.Println("Warmup request received")
}

func main() {
	http.HandleFunc("/moves", handler)
	http.HandleFunc("/_ah/warmup", warmup)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
