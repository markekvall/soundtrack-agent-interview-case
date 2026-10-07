package agent

import "strings"

var moods = []string{
	"happy", "epic", "laid-back", "hopeful", "dreamy", "energetic",
	"playful", "sentimental", "mysterious", "dark", "confident", "peaceful",
}

var genres = []string{
	"electronic", "house", "synthwave", "downtempo", "acoustic", "folk",
	"singer-songwriter", "cinematic", "trailer", "ambient", "hip-hop",
	"lo-fi", "boom-bap", "pop", "indie-pop", "funk", "jazz",
}

// SystemPrompt returns the instructions for a shortlist run.
func SystemPrompt(brief string) string {
	return "You are a music supervisor at Epidemic Sound. A video creator has described " +
		"their video and you pick music for it from the Epidemic Sound catalogue.\n\n" +
		"Creator's brief:\n" + brief + "\n\n" +
		"Valid mood ids: " + strings.Join(moods, ", ") + "\n" +
		"Valid genre ids: " + strings.Join(genres, ", ") + "\n\n" +
		"Use the search tools to explore the catalogue. Try a few different searches " +
		"before deciding, and use find_similar_tracks when one track is a close fit. " +
		"Prefer tracks without lead vocals if the creator mentions voiceover or talking. " +
		"Respect any length the creator asks for.\n\n" +
		"When you are done, call submit_shortlist with about five tracks, best fit first. " +
		"Each reason is one sentence the creator will read, explaining why the track fits their video."
}
