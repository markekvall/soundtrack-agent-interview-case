package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/epidemicsound/soundtrack-agent/internal/content"
)

var tools = []anthropic.ToolUnionParam{
	{OfTool: &anthropic.ToolParam{
		Name:        "search_tracks",
		Description: anthropic.String("Search the catalogue. All filters are optional. Durations are in seconds."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"term":         map[string]any{"type": "string", "description": "Free-text keywords, e.g. 'cooking ukulele'"},
				"moods":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"genres":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"bpm_min":      map[string]any{"type": "integer"},
				"bpm_max":      map[string]any{"type": "integer"},
				"min_duration": map[string]any{"type": "integer"},
				"max_duration": map[string]any{"type": "integer"},
			},
		},
	}},
	{OfTool: &anthropic.ToolParam{
		Name:        "get_track",
		Description: anthropic.String("Get full details for one track."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{"track_id": map[string]any{"type": "string"}},
			Required:   []string{"track_id"},
		},
	}},
	{OfTool: &anthropic.ToolParam{
		Name:        "find_similar_tracks",
		Description: anthropic.String("Find tracks that sound similar to the given track."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{"track_id": map[string]any{"type": "string"}},
			Required:   []string{"track_id"},
		},
	}},
	{OfTool: &anthropic.ToolParam{
		Name:        "submit_shortlist",
		Description: anthropic.String("Submit the final shortlist for the creator. Call this exactly once, at the end."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"tracks": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":     map[string]any{"type": "string"},
							"title":  map[string]any{"type": "string"},
							"artist": map[string]any{"type": "string"},
							"reason": map[string]any{"type": "string"},
						},
						"required": []string{"id", "title", "artist", "reason"},
					},
				},
			},
			Required: []string{"tracks"},
		},
	}},
}

type searchInput struct {
	Term        string   `json:"term"`
	Moods       []string `json:"moods"`
	Genres      []string `json:"genres"`
	BPMMin      int      `json:"bpm_min"`
	BPMMax      int      `json:"bpm_max"`
	MinDuration int      `json:"min_duration"`
	MaxDuration int      `json:"max_duration"`
}

type trackInput struct {
	TrackID string `json:"track_id"`
}

func (a *Agent) callTool(ctx context.Context, name string, input json.RawMessage) (string, error) {
	var result any
	switch name {
	case "search_tracks":
		var in searchInput
		if err := json.Unmarshal(input, &in); err != nil {
			return "", err
		}
		tracks, err := a.content.Search(ctx, content.SearchParams{
			Term:   in.Term,
			Moods:  in.Moods,
			Genres: in.Genres,
			BPMMin: in.BPMMin,
			BPMMax: in.BPMMax,
		})
		if err != nil {
			return "", err
		}
		result = filterByDuration(tracks, in.MinDuration, in.MaxDuration)

	case "get_track":
		var in trackInput
		if err := json.Unmarshal(input, &in); err != nil {
			return "", err
		}
		track, err := a.content.Track(ctx, in.TrackID)
		if err != nil {
			return "", err
		}
		result = track

	case "find_similar_tracks":
		var in trackInput
		if err := json.Unmarshal(input, &in); err != nil {
			return "", err
		}
		tracks, err := a.content.Similar(ctx, in.TrackID)
		if err != nil {
			return "", err
		}
		result = tracks

	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}

	out, err := json.Marshal(result)
	return string(out), err
}

// filterByDuration keeps tracks whose length in seconds falls within
// [min, max]. A zero bound is ignored.
func filterByDuration(tracks []content.Track, min, max int) []content.Track {
	out := make([]content.Track, 0, len(tracks))
	for _, t := range tracks {
		if min > 0 && t.Length < min {
			continue
		}
		if max > 0 && t.Length > max {
			continue
		}
		out = append(out, t)
	}
	return out
}
