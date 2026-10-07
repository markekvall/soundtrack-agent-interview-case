// Package content is a client for the Epidemic Sound Partner Content API.
package content

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Named struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Track struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	MainArtists     []Named `json:"mainArtists"`
	FeaturedArtists []Named `json:"featuredArtists"`
	BPM             int     `json:"bpm"`
	Length          int     `json:"length"`
	Moods           []Named `json:"moods"`
	Genres          []Named `json:"genres"`
	VocalType       string  `json:"vocalType"`
	HasVocals       bool    `json:"hasVocals"`
	IsExplicit      bool    `json:"isExplicit"`
	IsPreviewOnly   bool    `json:"isPreviewOnly"`
	Added           string  `json:"added"`
}

type Stream struct {
	URL     string    `json:"url"`
	Expires time.Time `json:"expires"`
}

type SearchParams struct {
	Term   string
	Moods  []string
	Genres []string
	BPMMin int
	BPMMax int
}

// Query encodes the params in the form /v0/tracks/search expects.
func (p SearchParams) Query() url.Values {
	q := url.Values{}
	if p.Term != "" {
		q.Set("term", p.Term)
	}
	for _, m := range p.Moods {
		q.Add("mood", m)
	}
	for _, g := range p.Genres {
		q.Add("genre", g)
	}
	if p.BPMMin > 0 {
		q.Set("bpmMin", strconv.Itoa(p.BPMMin))
	}
	if p.BPMMax > 0 {
		q.Set("bpmMax", strconv.Itoa(p.BPMMax))
	}
	return q
}

type Client struct {
	baseURL string
	apiKey  string
	cache   map[string][]Track
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		cache:   map[string][]Track{},
	}
}

func (c *Client) Search(ctx context.Context, p SearchParams) ([]Track, error) {
	query := p.Query().Encode()
	if tracks, ok := c.cache[query]; ok {
		return tracks, nil
	}

	var resp struct {
		Tracks []Track `json:"tracks"`
	}
	if err := c.get(ctx, "/v0/tracks/search?"+query, &resp); err != nil {
		return nil, err
	}
	c.cache[query] = resp.Tracks
	return resp.Tracks, nil
}

func (c *Client) Track(ctx context.Context, id string) (Track, error) {
	var resp struct {
		Tracks []Track `json:"tracks"`
	}
	q := url.Values{"trackId": {id}}
	if err := c.get(ctx, "/v0/tracks/metadata?"+q.Encode(), &resp); err != nil {
		return Track{}, err
	}
	if len(resp.Tracks) == 0 {
		return Track{}, fmt.Errorf("track %s not found", id)
	}
	return resp.Tracks[0], nil
}

func (c *Client) Similar(ctx context.Context, id string) ([]Track, error) {
	var resp struct {
		Tracks []Track `json:"tracks"`
	}
	if err := c.get(ctx, "/v0/tracks/"+url.PathEscape(id)+"/similar", &resp); err != nil {
		return nil, err
	}
	return resp.Tracks, nil
}

// Stream returns a short-lived URL for previewing a track.
func (c *Client) Stream(ctx context.Context, id string) (Stream, error) {
	var s Stream
	err := c.get(ctx, "/v0/tracks/"+url.PathEscape(id)+"/stream", &s)
	return s, err
}

// ReportUsage tells Epidemic Sound that the tracks were exported to platform.
func (c *Client) ReportUsage(ctx context.Context, platform string, trackIDs ...string) error {
	body, err := json.Marshal(map[string]any{
		"eventType": "EXPORTED",
		"platform":  platform,
		"trackIds":  trackIDs,
	})
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/v0/usage", bytes.NewReader(body), nil)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("content api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("content api: %s %s returned %d", method, path, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
