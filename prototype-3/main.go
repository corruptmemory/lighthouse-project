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
	DBPath     string `long:"db-path" default:":memory:" description:"SQLite database path"`
	APIKey     string `long:"api-key" description:"Anthropic API key"`
	WidgetsDir string `long:"widgets-dir" default:"widgets" description:"Directory for widget/dashboard markdown files"`
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

	tmpls, err := LoadTemplates()
	if err != nil {
		log.Fatalf("load templates: %v", err)
	}

	// Ensure widgets directory exists
	if err := os.MkdirAll(opts.WidgetsDir, 0o755); err != nil {
		log.Fatalf("create widgets dir: %v", err)
	}

	h := &Handlers{
		db:         db,
		tmpls:      tmpls,
		sessions:   NewSessionStore(),
		widgetsDir: opts.WidgetsDir,
		agentCfg: AgentConfig{
			APIKey:     opts.APIKey,
			DB:         db,
			WidgetsDir: opts.WidgetsDir,
		},
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Full page renders
	r.Get("/", h.HandleIndex)
	r.Get("/dashboard", h.HandleDashboardPage)

	// Order CRUD — HTML partials
	r.Get("/api/orders", h.HandleListOrders)
	r.Get("/api/orders/new", h.HandleNewOrder)
	r.Get("/api/orders/{id}", h.HandleGetOrder)
	r.Get("/api/orders/{id}/edit", h.HandleEditOrder)
	r.Post("/api/orders", h.HandleCreateOrder)
	r.Put("/api/orders/{id}", h.HandleUpdateOrder)

	// AI-powered order filter
	r.Post("/api/orders/ai-filter", h.HandleAIFilter)

	// AI bulk operations
	r.Post("/api/orders/ai-bulk-ops", h.HandleBulkOpsPropose)
	r.Post("/api/orders/ai-bulk-ops/execute", h.HandleBulkOpsExecute)

	// Product list for order form
	r.Get("/api/products", h.HandleListProducts)

	// AI chat + widgets
	r.Post("/api/chat", h.HandleChat)
	r.Get("/api/widgets", h.HandleListWidgets)
	r.Get("/api/widgets/{slug}", h.HandleGetWidgetBySlug)
	r.Get("/api/dashboards/{slug}", h.HandleDashboardView)

	// Static files
	r.Handle("/*", http.FileServer(http.Dir("static")))

	addr := fmt.Sprintf(":%d", opts.Port)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

type Handlers struct {
	db         *sql.DB
	tmpls      *Templates
	sessions   *SessionStore
	widgetsDir string
	agentCfg   AgentConfig
}
