package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rajat/localdiscovery/internal/ports"
)

type OpenAI struct {
	apiKey string
	client *http.Client
}

func NewOpenAI(apiKey string) *OpenAI {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return &OpenAI{apiKey: apiKey, client: &http.Client{Timeout: 20 * time.Second}}
}

func (o *OpenAI) Complete(ctx context.Context, system, user string) (string, error) {
	body := map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.2,
		"max_tokens":  400,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("openai status %d", res.StatusCode)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

var _ ports.LLMClient = (*OpenAI)(nil)
