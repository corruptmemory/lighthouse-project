package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Port       int    `long:"port" default:"8080" description:"HTTP server port"`
	APIKey     string `long:"api-key" required:"true" description:"Anthropic API key"`
	DBPath     string `long:"db-path" default:":memory:" description:"SQLite database path"`
	WidgetsDir string `long:"widgets-dir" default:"./widgets" description:"Directory for widget and dashboard markdown files"`
}

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	// Ensure widgets directory exists
	if err := os.MkdirAll(opts.WidgetsDir, 0o755); err != nil {
		log.Fatalf("create widgets dir: %v", err)
	}

	db, err := InitDB(opts.DBPath)
	if err != nil {
		log.Fatalf("database init: %v", err)
	}
	defer db.Close()

	sessions := NewSessionStore()
	agentCfg := AgentConfig{
		APIKey:     opts.APIKey,
		DB:         db,
		WidgetsDir: opts.WidgetsDir,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &Handlers{
		db:         db,
		sessions:   sessions,
		agentCfg:   agentCfg,
		widgetsDir: opts.WidgetsDir,
	}

	r.Post("/api/chat", h.HandleChat)
	r.Get("/api/widgets/{slug}", h.HandleGetWidgetBySlug)
	r.Get("/api/dashboards/{slug}", h.HandleDashboard)
	r.Get("/api/widgets", h.HandleListWidgets)
	r.Handle("/*", http.FileServer(http.Dir("static")))

	addr := fmt.Sprintf(":%d", opts.Port)
	log.Printf("Listening on %s (widgets dir: %s)", addr, opts.WidgetsDir)
	log.Fatal(http.ListenAndServe(addr, r))
}

type Handlers struct {
	db         *sql.DB
	sessions   *SessionStore
	agentCfg   AgentConfig
	widgetsDir string
}
