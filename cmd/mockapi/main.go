// Command mockapi serves a small fictional catalogue in the shape of the
// Epidemic Sound Partner Content API, so the backend can run without
// credentials. Titles and artists are made up.
package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed tracks.json
var tracksJSON []byte

type named struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type track struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	MainArtists     []named  `json:"mainArtists"`
	FeaturedArtists []named  `json:"featuredArtists"`
	BPM             int      `json:"bpm"`
	Length          int      `json:"length"`
	Moods           []named  `json:"moods"`
	Genres          []named  `json:"genres"`
	VocalType       string   `json:"vocalType"`
	HasVocals       bool     `json:"hasVocals"`
	IsExplicit      bool     `json:"isExplicit"`
	IsPreviewOnly   bool     `json:"isPreviewOnly"`
	Added           string   `json:"added"`
	Tags            []string `json:"tags,omitempty"`
}

type server struct {
	tracks      []track
	failureRate float64
	perMinute   int

	publicURL string
	secret    []byte

	mu       sync.Mutex
	requests []time.Time
	usage    []usageReport
}

type usageReport struct {
	EventType string   `json:"eventType"`
	Platform  string   `json:"platform"`
	TrackIDs  []string `json:"trackIds"`
}

func main() {
	var tracks []track
	if err := json.Unmarshal(tracksJSON, &tracks); err != nil {
		log.Fatal(err)
	}
	s := &server{
		tracks:      tracks,
		failureRate: envFloat("MOCK_FAILURE_RATE", 0),
		perMinute:   int(envFloat("MOCK_RATE_LIMIT", 60)),
		publicURL:   envOr("MOCK_PUBLIC_URL", "http://localhost:4010"),
		secret:      []byte(strconv.FormatInt(time.Now().UnixNano(), 10)),
	}

	api := http.NewServeMux()
	api.HandleFunc("GET /v0/tracks/search", s.search)
	api.HandleFunc("GET /v0/tracks/metadata", s.metadata)
	api.HandleFunc("GET /v0/tracks/{id}/similar", s.similar)
	api.HandleFunc("GET /v0/tracks/{id}/stream", s.stream)
	api.HandleFunc("POST /v0/usage", s.reportUsage)

	// Media URLs are signed, so they are served without bearer auth.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /media/{id}", s.media)
	mux.Handle("/", s.middleware(api))

	addr := ":" + envOr("MOCK_PORT", "4010")
	log.Printf("mock content API listening on %s (%d tracks)", addr, len(tracks))
	log.Fatal(http.ListenAndServe(addr, mux))
}

// middleware applies bearer auth and a per-minute rate limit like the real API,
// and adds some response latency.
func (s *server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
			return
		}
		if retryAfter, ok := s.allow(); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "Rate limit exceeded"})
			return
		}
		time.Sleep(time.Duration(50+rand.IntN(200)) * time.Millisecond)
		if rand.Float64() < s.failureRate {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"message": "Service temporarily unavailable"})
			return
		}
		log.Printf("%s %s", r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (s *server) allow() (retryAfter int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.requests = slices.DeleteFunc(s.requests, func(t time.Time) bool { return now.Sub(t) > time.Minute })
	if len(s.requests) >= s.perMinute {
		return int(time.Minute-now.Sub(s.requests[0]))/int(time.Second) + 1, false
	}
	s.requests = append(s.requests, now)
	return 0, true
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	terms := strings.Fields(strings.ToLower(q.Get("term")))
	bpmMin, _ := strconv.Atoi(q.Get("bpmMin"))
	bpmMax, err := strconv.Atoi(q.Get("bpmMax"))
	if err != nil {
		bpmMax = 1000
	}

	type scored struct {
		t     track
		score int
	}
	var hits []scored
	for _, t := range s.tracks {
		if !matchesAny(t.Moods, q["mood"]) || !matchesAny(t.Genres, q["genre"]) {
			continue
		}
		if t.BPM < bpmMin || t.BPM > bpmMax {
			continue
		}
		if vt := q["vocalType"]; len(vt) > 0 && !slices.Contains(vt, t.VocalType) {
			continue
		}
		score := 0
		haystack := strings.ToLower(t.Title + " " + t.MainArtists[0].Name + " " + strings.Join(t.Tags, " "))
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				score++
			}
		}
		if len(terms) > 0 && score == 0 {
			continue
		}
		hits = append(hits, scored{t, score})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })

	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 || limit > 60 {
		limit = 50
	}
	out := make([]track, 0, limit)
	for _, h := range hits[:min(limit, len(hits))] {
		out = append(out, h.t)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tracks":     out,
		"pagination": map[string]int{"page": 1, "limit": limit, "offset": 0},
		"links":      map[string]any{},
	})
}

func (s *server) metadata(w http.ResponseWriter, r *http.Request) {
	ids := r.URL.Query()["trackId"]
	out := []track{}
	for _, t := range s.tracks {
		if slices.Contains(ids, t.ID) {
			out = append(out, t)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracks": out})
}

func (s *server) similar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	i := slices.IndexFunc(s.tracks, func(t track) bool { return t.ID == id })
	if i < 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Track not found"})
		return
	}
	seed := s.tracks[i]

	type scored struct {
		t     track
		score int
	}
	var hits []scored
	for _, t := range s.tracks {
		if t.ID == seed.ID {
			continue
		}
		score := overlap(t.Moods, seed.Moods) + 2*overlap(t.Genres, seed.Genres)
		if d := t.BPM - seed.BPM; d > -15 && d < 15 {
			score++
		}
		if score > 1 {
			hits = append(hits, scored{t, score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	out := []track{}
	for _, h := range hits[:min(10, len(hits))] {
		out = append(out, h.t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tracks": out})
}

func (s *server) reportUsage(w http.ResponseWriter, r *http.Request) {
	var report usageReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil || report.Platform == "" || len(report.TrackIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid usage report"})
		return
	}
	for _, id := range report.TrackIDs {
		if !slices.ContainsFunc(s.tracks, func(t track) bool { return t.ID == id }) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Unknown track id " + id})
			return
		}
	}
	s.mu.Lock()
	s.usage = append(s.usage, report)
	n := len(s.usage)
	s.mu.Unlock()
	log.Printf("usage report #%d: %s %v", n, report.Platform, report.TrackIDs)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Usage reported"})
}

func matchesAny(have []named, want []string) bool {
	if len(want) == 0 {
		return true
	}
	for _, h := range have {
		if slices.Contains(want, h.ID) {
			return true
		}
	}
	return false
}

func overlap(a, b []named) int {
	n := 0
	for _, x := range a {
		for _, y := range b {
			if x.ID == y.ID {
				n++
			}
		}
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	f, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return fallback
	}
	return f
}
