package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GeminiClient struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

func NewGemini(apiKey, model string) *GeminiClient {
	return &GeminiClient{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 12 * time.Second}}
}

func (g *GeminiClient) Ask(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", g.Model, g.APIKey)
	payload := generateContentRequest{Contents: []content{{Role: "user", Parts: []part{{Text: prompt}}}}}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := g.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini status %d", res.StatusCode)
	}
	var out generateContentResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}
