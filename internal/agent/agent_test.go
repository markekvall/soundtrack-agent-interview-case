package agent

import (
	"encoding/json"
	"testing"

	"github.com/epidemicsound/soundtrack-agent/internal/content"
)

func TestFilterByDuration(t *testing.T) {
	tracks := []content.Track{
		{ID: "short", Length: 90},
		{ID: "medium", Length: 150},
		{ID: "long", Length: 240},
	}

	got := filterByDuration(tracks, 100, 200)
	if len(got) != 1 || got[0].ID != "medium" {
		t.Errorf("filterByDuration() = %v, want only medium", got)
	}
}

func TestParseShortlist(t *testing.T) {
	input := json.RawMessage(`{"tracks":[
		{"id":"Gc8xN2dLm4","title":"Warm Kitchen","artist":"Olive Marsh","reason":"Cosy ukulele that sits under a voiceover."}
	]}`)

	picks, err := parseShortlist(input)
	if err != nil {
		t.Fatal(err)
	}
	want := Pick{ID: "Gc8xN2dLm4", Title: "Warm Kitchen", Artist: "Olive Marsh", Reason: "Cosy ukulele that sits under a voiceover."}
	if len(picks) != 1 || picks[0] != want {
		t.Errorf("parseShortlist() = %v, want [%v]", picks, want)
	}
}
