package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/joho/godotenv"

	"github.com/alaric/eino-learn/internal/config"
	"github.com/alaric/eino-learn/internal/game"
	"github.com/alaric/eino-learn/internal/genesis"

	_ "github.com/alaric/eino-learn/internal/role"
)

func main() {
	_ = godotenv.Load()

	if os.Getenv("CLAUDE_API_KEY") == "" && os.Getenv("GEMINI_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "Set CLAUDE_API_KEY and/or GEMINI_API_KEY environment variables.")
		os.Exit(1)
	}

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/game", handleGame)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Werewolf game server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func projectRoot() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..")
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	htmlPath := filepath.Join(projectRoot(), "web", "index.html")
	http.ServeFile(w, r, htmlPath)
}

var gameMu sync.Mutex
var gameRunning bool

func handleGame(w http.ResponseWriter, r *http.Request) {
	gameMu.Lock()
	if gameRunning {
		gameMu.Unlock()
		http.Error(w, "A game is already in progress", http.StatusConflict)
		return
	}
	gameRunning = true
	gameMu.Unlock()

	defer func() {
		gameMu.Lock()
		gameRunning = false
		gameMu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	ctx := r.Context()

	emit := func(event game.UIEvent) {
		data, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	emit(game.UIEvent{Type: "genesis_start"})

	modelCfg := config.ModelConfigFromEnv()
	cfg, err := genesis.Create(ctx, modelCfg)
	if err != nil {
		emit(game.UIEvent{Type: "error", Content: fmt.Sprintf("Genesis failed: %v", err)})
		return
	}

	engine, err := game.NewEngine(ctx, cfg, game.WithEmitter(emit), game.WithSilent())
	if err != nil {
		emit(game.UIEvent{Type: "error", Content: fmt.Sprintf("Failed to create game: %v", err)})
		return
	}

	if err := engine.Run(context.Background()); err != nil {
		emit(game.UIEvent{Type: "error", Content: fmt.Sprintf("Game error: %v", err)})
	}
}
