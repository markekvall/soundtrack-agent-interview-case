// Command server runs the soundtrack agent backend.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/epidemicsound/soundtrack-agent/internal/agent"
	"github.com/epidemicsound/soundtrack-agent/internal/content"
	"github.com/epidemicsound/soundtrack-agent/internal/server"
	"github.com/epidemicsound/soundtrack-agent/internal/store"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY is not set")
	}

	db, err := store.Open(envOr("DB_PATH", "soundtrack.db"))
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	llm := anthropic.NewClient(option.WithAPIKey(apiKey))
	catalogue := content.NewClient(
		envOr("CONTENT_API_URL", "http://localhost:4010"),
		envOr("CONTENT_API_KEY", "local-dev-key"),
	)
	a := agent.New(llm, envOr("ANTHROPIC_MODEL", "claude-sonnet-5-5"), catalogue)

	addr := ":" + envOr("PORT", "8080")
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.New(db, a, catalogue).Routes()))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
