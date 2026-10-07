# Soundtrack Agent

An LLM agent that turns a creator's video brief ("calm acoustic background for a cooking tutorial, about two minutes") into a shortlist of tracks from the Epidemic Sound catalogue, with a one-line reason for each. The creator can preview tracks and mark one as used, which reports the usage to Epidemic Sound.

## For the interview

We'll spend about 40 minutes talking through this repo together. Please spend no more than an hour with it beforehand. Use whatever tools you normally would, AI assistants included, both while preparing and during the interview. You don't need to write any code, and you don't need any prior experience building agents. If you don't get time to prepare, that's fine, and it won't be held against you.

To prepare yourself for our discussion, think about:

- How a brief becomes a shortlist of tracks. Be ready to walk us through the flow.
- What you'd change before putting this in front of real users, and why.
- How you'd extend it. We'll give you a small feature to plan together, using whatever AI coding tool you normally work with, so have the repo ready on your machine.

If you need an API key for your AI tool, or a laptop for the interview, let us know and we'll provide one.

This is a prototype we put together quickly, so there's plenty here to question, challenge, or disagree with.

Start reading at `handleShortlist` in `internal/server/server.go` and follow a request from there.

## How it fits together

```
web/ (React) ──HTTP──▶ internal/server ──▶ internal/agent ──▶ Claude
                          │      │              │
                          │      └──────────────┴──▶ internal/content ──HTTP──▶ content API
                          ▼
                        SQLite
```

- `internal/server` holds the HTTP handlers.
- `internal/agent` runs a tool-calling loop with Claude. The model searches the catalogue, looks at track details and similar tracks, then calls `submit_shortlist`.
- `internal/content` is the client for the [Partner Content API](https://partner-content-api.epidemicsound.com/docs/spec.json).
- `internal/store` opens the SQLite database that keeps a local copy of usage reports.
- `web/` is a single-page React app.

`cmd/mockapi` stands in for the real Partner Content API with a small fictional catalogue, so the tracks and artists aren't real. You don't need to read it.

**Stack:** Go (standard library `net/http`), the Anthropic Go SDK, SQLite, and React with TypeScript on Vite.

## Running it

Running it is optional, and reading the code is enough. If you do want to run it, you need Docker and an Anthropic API key:

```bash
cp .env.example .env    # fill in ANTHROPIC_API_KEY
docker compose up       # mock API on :4010, backend on :8080, web app on :5173
```

Then open [http://localhost:5173](http://localhost:5173).

If you have Go 1.26 and Node 20 installed, `make dev` runs the same three processes without Docker, and `make test` runs the Go tests.
