package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	Endpoint string
	APIKey   string
	Model    string
	HTTP     *http.Client
}

func New(endpoint, apiKey, model string) *Client {
	return &Client{Endpoint: endpoint, APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 45 * time.Second}}
}

func (c *Client) Complete(ctx context.Context, msgs []Message) (string, error) {
	body, err := json.Marshal(map[string]any{"model": c.Model, "messages": msgs, "max_tokens": 800})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: upstream status %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: empty completion")
	}
	return out.Choices[0].Message.Content, nil
}
