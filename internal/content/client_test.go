package content

import "testing"

func TestSearchParamsQuery(t *testing.T) {
	p := SearchParams{
		Term:   "cooking",
		Moods:  []string{"happy", "laid-back"},
		Genres: []string{"folk"},
		BPMMin: 80,
		BPMMax: 110,
	}

	got := p.Query().Encode()
	want := "bpmMax=110&bpmMin=80&genre=folk&mood=happy&mood=laid-back&term=cooking"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}
