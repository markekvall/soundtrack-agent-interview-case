package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"time"
)

const streamTTL = 10 * time.Minute

// stream returns a short-lived signed URL, like the real API's
// /v0/tracks/{id}/stream.
func (s *server) stream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !slices.ContainsFunc(s.tracks, func(t track) bool { return t.ID == id }) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Track not found"})
		return
	}
	exp := time.Now().Add(streamTTL)
	url := fmt.Sprintf("%s/media/%s?exp=%d&sig=%s", s.publicURL, id, exp.Unix(), s.sign(id, exp.Unix()))
	writeJSON(w, http.StatusOK, map[string]string{"url": url, "expires": exp.UTC().Format(time.RFC3339)})
}

// media serves a placeholder tone that pulses at the track's BPM.
func (s *server) media(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	exp, _ := strconv.ParseInt(r.URL.Query().Get("exp"), 10, 64)
	if time.Now().Unix() > exp || !hmac.Equal([]byte(r.URL.Query().Get("sig")), []byte(s.sign(id, exp))) {
		http.Error(w, "link expired or invalid", http.StatusForbidden)
		return
	}
	i := slices.IndexFunc(s.tracks, func(t track) bool { return t.ID == id })
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Write(tone(s.tracks[i].BPM))
}

func (s *server) sign(id string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "%s:%d", id, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// tone renders 8 seconds of 8-bit mono WAV.
func tone(bpm int) []byte {
	const rate, seconds = 8000, 8
	samples := make([]byte, rate*seconds)
	beat := 60 / float64(bpm)
	for i := range samples {
		t := float64(i) / rate
		envelope := math.Exp(-6 * math.Mod(t, beat) / beat)
		samples[i] = byte(128 + 60*envelope*math.Sin(2*math.Pi*220*t))
	}

	var b bytes.Buffer
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(36+len(samples)))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate), uint16(1), uint16(8)} {
		binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(len(samples)))
	b.Write(samples)
	return b.Bytes()
}
