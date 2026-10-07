// Package agent runs the tool-calling loop that turns a brief into a shortlist.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/epidemicsound/soundtrack-agent/internal/content"
)

type Pick struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Reason string `json:"reason"`
}

type Agent struct {
	llm     anthropic.Client
	model   string
	content *content.Client
}

func New(llm anthropic.Client, model string, content *content.Client) *Agent {
	return &Agent{llm: llm, model: model, content: content}
}

// Run drives the conversation until the model submits a shortlist.
func (a *Agent) Run(ctx context.Context, system, prompt string) ([]Pick, error) {
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	for {
		resp, err := a.llm.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(a.model),
			MaxTokens: 8192,
			System:    []anthropic.TextBlockParam{{Text: system}},
			Tools:     tools,
			Messages:  messages,
		})
		if err != nil {
			return nil, err
		}
		messages = append(messages, resp.ToParam())

		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			use, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}
			input := json.RawMessage(use.JSON.Input.Raw())
			if use.Name == "submit_shortlist" {
				return parseShortlist(input)
			}
			results = append(results, anthropic.NewToolResultBlock(use.ID, a.runTool(ctx, use.Name, input), false))
		}

		if resp.StopReason != anthropic.StopReasonToolUse {
			return nil, errors.New("agent stopped without submitting a shortlist")
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
}

func (a *Agent) runTool(ctx context.Context, name string, input json.RawMessage) string {
	result, err := a.callTool(ctx, name, input)
	if err != nil {
		log.Printf("tool %s failed: %v", name, err)
		return "[]"
	}
	return result
}

func parseShortlist(input json.RawMessage) ([]Pick, error) {
	var in struct {
		Tracks []Pick `json:"tracks"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}
	return in.Tracks, nil
}
