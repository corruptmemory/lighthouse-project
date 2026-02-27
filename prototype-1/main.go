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
	Port   int    `long:"port" default:"8080" description:"HTTP server port"`
	APIKey string `long:"api-key" required:"true" description:"Anthropic API key"`
	DBPath string `long:"db-path" default:":memory:" description:"SQLite database path"`
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

	db, err := InitDB(opts.DBPath)
	if err != nil {
		log.Fatalf("database init: %v", err)
	}
	defer db.Close()

	sessions := NewSessionStore()
	agentCfg := AgentConfig{
		APIKey: opts.APIKey,
		DB:     db,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &Handlers{
		db:       db,
		sessions: sessions,
		agentCfg: agentCfg,
	}

	r.Post("/api/chat", h.HandleChat)
	r.Get("/api/widgets/{id}", h.HandleGetWidget)
	r.Handle("/*", http.FileServer(http.Dir("static")))

	addr := fmt.Sprintf(":%d", opts.Port)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

type Handlers struct {
	db       *sql.DB
	sessions *SessionStore
	agentCfg AgentConfig
}
