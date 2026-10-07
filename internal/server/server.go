// Package server exposes the HTTP API used by the web app.
package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/epidemicsound/soundtrack-agent/internal/agent"
	"github.com/epidemicsound/soundtrack-agent/internal/content"
)

const userHeader = "X-User-ID"

type Server struct {
	db      *sql.DB
	agent   *agent.Agent
	content *content.Client
}

func New(db *sql.DB, agent *agent.Agent, content *content.Client) *Server {
	return &Server{db: db, agent: agent, content: content}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shortlist", s.handleShortlist)
	mux.HandleFunc("POST /api/usage", s.handleUsage)
	mux.HandleFunc("GET /api/tracks/{id}/preview", s.handlePreview)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

type shortlistItem struct {
	agent.Pick
	Used bool `json:"used"`
}

func (s *Server) handleShortlist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Brief string `json:"brief"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	brief := strings.TrimSpace(req.Brief)
	if brief == "" {
		writeError(w, http.StatusBadRequest, "brief is required")
		return
	}

	used := map[string]bool{}
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT DISTINCT track_id FROM usage_reports WHERE user_id = ?`, r.Header.Get(userHeader))
	if err != nil {
		log.Printf("shortlist: loading usage: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("shortlist: reading usage: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		used[id] = true
	}
	if err := rows.Err(); err != nil {
		log.Printf("shortlist: reading usage: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	picks, err := s.agent.Run(r.Context(), agent.SystemPrompt(brief), "Put together a shortlist for my video.")
	if err != nil {
		log.Printf("shortlist: agent run failed: %v", err)
		writeError(w, http.StatusBadGateway, "could not build a shortlist, please try again")
		return
	}

	items := make([]shortlistItem, len(picks))
	for i, p := range picks {
		items[i] = shortlistItem{Pick: p, Used: used[p.ID]}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracks": items})
}

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get(userHeader)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing "+userHeader+" header")
		return
	}
	var req struct {
		TrackID  string `json:"trackId"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TrackID == "" || req.Platform == "" {
		writeError(w, http.StatusBadRequest, "trackId and platform are required")
		return
	}

	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO usage_reports (user_id, track_id, platform) VALUES (?, ?, ?)`, userID, req.TrackID, req.Platform)
	if err != nil {
		log.Printf("usage: recording %s for %s: %v", req.TrackID, userID, err)
	}

	if err := s.content.ReportUsage(r.Context(), req.Platform, req.TrackID); err != nil {
		log.Printf("usage: reporting %s to content API: %v", req.TrackID, err)
		writeError(w, http.StatusBadGateway, "could not report usage, please try again")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	stream, err := s.content.Stream(r.Context(), r.PathValue("id"))
	if err != nil {
		log.Printf("preview: %v", err)
		writeError(w, http.StatusBadGateway, "preview unavailable")
		return
	}
	writeJSON(w, http.StatusOK, stream)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writing response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
