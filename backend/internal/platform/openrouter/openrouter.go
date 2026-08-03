// Package openrouter is the provider adapter for OpenRouter inference. It owns
// the HTTP transport, the fixed Zero Data Retention (ZDR) provider policy, and
// model-capability discovery. OpenRouter response DTOs are decoded and discarded
// inside this package; only the provider-neutral Model, Message, and Completion
// views cross the boundary.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://openrouter.ai/api/v1"

// Client is the OpenRouter inference adapter. Every request applies ZDR policy.
type Client struct {
	apiKey string
	http   *http.Client
}

// New constructs a client from an API key. An empty key yields a client that
// reports Configured() == false and errors on every network call.
func New(apiKey string) *Client {
	return &Client{apiKey: strings.TrimSpace(apiKey), http: &http.Client{Timeout: 60 * time.Second}}
}

// Configured reports whether an API key is present.
func (c *Client) Configured() bool { return c.apiKey != "" }

// Model is a provider-neutral capability view of one model endpoint. Purpose
// policy (which capabilities a use-case requires) is applied by callers.
type Model struct {
	ID                       string
	Name                     string
	SupportsTextInput        bool
	SupportsImageInput       bool
	SupportsStructuredOutput bool
	HasZDREndpoint           bool
}

// Image is one image part of a multimodal message. URL may be a data: URI.
type Image struct {
	URL string
}

// Message is a provider-neutral chat message. When Images is non-empty the
// message is encoded as multimodal content (text parts plus image parts).
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
	Images  []Image
}

// Schema requests strict structured JSON output validated against a JSON Schema.
type Schema struct {
	Name string
	Body map[string]any
}

// Usage is non-content generation metadata returned alongside a completion.
type Usage struct {
	GenerationID     string
	PromptTokens     int
	CompletionTokens int
}

// Completion is the model's returned content plus the model that produced it and
// its usage metadata.
type Completion struct {
	Content string
	Model   string
	Usage   Usage
}

// Complete runs one non-streaming chat completion under the fixed ZDR provider
// policy. A non-nil schema requests strict structured output. The caller supplies
// fully-formed messages; prompt construction is not this adapter's concern.
func (c *Client) Complete(ctx context.Context, modelID string, messages []Message, schema *Schema) (Completion, error) {
	if c.apiKey == "" {
		return Completion{}, fmt.Errorf("OpenRouter API key is not configured")
	}
	requestMessages := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		requestMessages = append(requestMessages, encodeMessage(message))
	}
	body := map[string]any{
		"model":    modelID,
		"messages": requestMessages,
		"stream":   false,
		"provider": map[string]any{
			"zdr":                true,
			"data_collection":    "deny",
			"require_parameters": true,
		},
	}
	if schema != nil {
		body["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": schema.Name, "strict": true, "schema": schema.Body}}
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Completion{}, fmt.Errorf("encode OpenRouter request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return Completion{}, fmt.Errorf("create OpenRouter request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return Completion{}, fmt.Errorf("send OpenRouter request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return Completion{}, fmt.Errorf("OpenRouter request returned %s", response.Status)
	}
	var payload struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return Completion{}, fmt.Errorf("decode OpenRouter response: %w", err)
	}
	if len(payload.Choices) != 1 || strings.TrimSpace(payload.Choices[0].Message.Content) == "" {
		return Completion{}, fmt.Errorf("OpenRouter response did not contain one complete message")
	}
	return Completion{
		Content: payload.Choices[0].Message.Content,
		Model:   modelID,
		Usage:   Usage{GenerationID: payload.ID, PromptTokens: payload.Usage.PromptTokens, CompletionTokens: payload.Usage.CompletionTokens},
	}, nil
}

// encodeMessage renders a message as OpenRouter chat content: a plain string when
// there are no images, otherwise an array of text and image_url parts.
func encodeMessage(message Message) map[string]any {
	if len(message.Images) == 0 {
		return map[string]any{"role": message.Role, "content": message.Content}
	}
	parts := make([]map[string]any, 0, len(message.Images)+1)
	if strings.TrimSpace(message.Content) != "" {
		parts = append(parts, map[string]any{"type": "text", "text": message.Content})
	}
	for _, image := range message.Images {
		parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": image.URL}})
	}
	return map[string]any{"role": message.Role, "content": parts}
}

// ListModels returns every model with its capability flags, intersected with the
// live ZDR endpoint list. It does not filter by capability; callers apply the
// per-purpose policy (for example requiring image input for vision).
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is not configured")
	}
	var modelsResponse struct {
		Data []struct {
			ID                  string   `json:"id"`
			Name                string   `json:"name"`
			SupportedParameters []string `json:"supported_parameters"`
			Architecture        struct {
				InputModalities []string `json:"input_modalities"`
			} `json:"architecture"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/models", &modelsResponse); err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := c.getJSON(ctx, "/endpoints/zdr", &raw); err != nil {
		return nil, err
	}
	zdrModels := findModelIDs(raw)
	models := make([]Model, 0, len(modelsResponse.Data))
	for _, model := range modelsResponse.Data {
		text, image := false, false
		for _, modality := range model.Architecture.InputModalities {
			switch modality {
			case "text":
				text = true
			case "image":
				image = true
			}
		}
		structured := false
		for _, parameter := range model.SupportedParameters {
			if parameter == "response_format" {
				structured = true
			}
		}
		models = append(models, Model{
			ID:                       model.ID,
			Name:                     model.Name,
			SupportsTextInput:        text,
			SupportsImageInput:       image,
			SupportsStructuredOutput: structured,
			HasZDREndpoint:           zdrModels[model.ID],
		})
	}
	return models, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create OpenRouter model request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("send OpenRouter model request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OpenRouter model request returned %s", response.Status)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(out); err != nil {
		return fmt.Errorf("decode OpenRouter model response: %w", err)
	}
	return nil
}

// findModelIDs walks the ZDR endpoints payload and collects any provider/model
// identifiers it contains, tolerating shape changes in the upstream response.
func findModelIDs(value any) map[string]bool {
	result := make(map[string]bool)
	var walk func(any)
	walk = func(current any) {
		switch item := current.(type) {
		case map[string]any:
			for key, nested := range item {
				if key == "model" || key == "model_id" || key == "id" {
					if text, ok := nested.(string); ok && strings.Contains(text, "/") {
						result[text] = true
					}
				}
				walk(nested)
			}
		case []any:
			for _, nested := range item {
				walk(nested)
			}
		}
	}
	walk(value)
	return result
}
