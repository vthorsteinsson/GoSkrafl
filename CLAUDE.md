# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoSkrafl is a concurrent crossword game engine and auto-playing robot written in Go. It implements a complete SCRABBLE-like game with support for multiple languages and dictionaries (OTCWL2014, SOWPODS, OSPS37, Norwegian, Icelandic).

## Development Commands

### Building and Running
- `go build` - Build the project
- `go run main/main.go` - Run the example program with default settings
- `go run main/main.go -d sowpods -n 5` - Run 5 games with SOWPODS dictionary
- `go run main/main.go -s` - Run as HTTP server on port 8080
- `go test` - Run tests

### Testing
- `go test` - Run all tests
- `go test -v` - Run tests with verbose output

### Server Deployment (Google App Engine)
- `cd go-app && ./deploy.sh <version>` - Deploy to explo-dev project
- `cd go-app && ./deploy-live.sh <version>` - Deploy to explo-live project

## Architecture

### Core Components

**Game Engine (`game.go`)**
- `Game` struct: Main game container with board, racks, bag, and move history
- `GameState` struct: Snapshot of current game state for move generation
- Manages player turns, scoring, and game completion logic

**Board Management (`board.go`)**
- `Board` struct: 15x15 game board with premium squares
- `Square` and `Tile` structs for board representation
- Handles tile placement and board state validation

**Dictionary System (`dawg.go`)**
- DAWG (Directed Acyclic Word Graph) implementation for efficient word validation
- Embedded binary dictionaries in `dicts/` directory
- Support for multiple languages with different alphabets

**Robot Players (`robot.go`)**
- `Robot` interface for AI players
- `HighScoreRobot`: Always picks highest-scoring move
- `OneOfNBestRobot`: Randomly selects from N best moves
- Concurrent move generation using goroutines

**Move Generation (`movegen.go`)**
- Generates all valid moves for current game state
- Uses parallel processing with goroutines for performance
- Handles both horizontal and vertical word placement

**HTTP Server (`server.go`)**
- JSON API endpoints: `/moves` and `/wordcheck`
- Handles move generation requests and word validation
- CORS support for web clients

### Key Data Structures
- `Move`: Represents a game move (tile placement, pass, exchange)
- `Rack`: Player's current tiles
- `Bag`: Tile distribution and drawing logic
- `TileSet`: Language-specific tile values and distributions

### Concurrency
The engine uses Go's goroutines extensively for parallel move generation, allowing it to utilize all available CPU cores for optimal performance.

## Dictionary Support
To add new dictionaries:
1. Create word list as UTF-8 text file (one word per line, lowercase)
2. Use Netskrafl's DAWG builder to create `.bin.dawg` file
3. Place in `dicts/` directory
4. Add alphabet string and constructor function in `dawg.go`
