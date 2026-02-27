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

func (h *Handlers) HandleGetWidget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	html, ok := getWidget(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
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
