package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

func (h *Handlers) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	session, created := h.sessions.GetOrCreate(req.SessionID)
	if created {
		log.Printf("New session: %s", session.ID)
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send session ID first
	writeSSE(w, flusher, "session", map[string]string{"session_id": session.ID})

	// Run agent loop with SSE emission
	emit := func(eventType string, data interface{}) {
		writeSSE(w, flusher, eventType, data)
	}

	if err := runAgentLoop(r.Context(), h.agentCfg, session, req.Message, emit); err != nil {
		log.Printf("Agent error: %v", err)
		writeSSE(w, flusher, "error", map[string]string{"error": err.Error()})
	}
}

// HandleGetWidgetBySlug loads a widget markdown from disk, executes its SQL,
// injects fresh data, and serves the composed HTML.
func (h *Handlers) HandleGetWidgetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	widget, err := loadWidget(h.widgetsDir, slug)
	if err != nil {
		log.Printf("Widget load error (%s): %v", slug, err)
		http.NotFound(w, r)
		return
	}

	html, err := renderWidget(h.db, widget)
	if err != nil {
		log.Printf("Widget render error (%s): %v", slug, err)
		http.Error(w, fmt.Sprintf("Error rendering widget: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleDashboard loads a dashboard markdown, builds a page with iframes for each widget.
func (h *Handlers) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	dashboard, err := loadDashboard(h.widgetsDir, slug)
	if err != nil {
		log.Printf("Dashboard load error (%s): %v", slug, err)
		http.NotFound(w, r)
		return
	}

	html := renderDashboardHTML(dashboard)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleListWidgets returns a JSON list of all saved widgets.
func (h *Handlers) HandleListWidgets(w http.ResponseWriter, r *http.Request) {
	widgets, err := listWidgets(h.widgetsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing widgets: %v", err), http.StatusInternalServerError)
		return
	}

	type widgetInfo struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}

	var result []widgetInfo
	for _, wg := range widgets {
		result = append(result, widgetInfo{Slug: wg.Slug, Title: wg.Title})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}
