package llm

import (
	"encoding/json"
	"fmt"
)

func RegisterTongYiProvider(registry *ProviderRegistry) {
	registry.Register("tongyi", NewTongYiProvider)
}

// TongYiProvider implements the Provider interface for OpenAI
type TongYiProvider struct {
	apiKey string
	model  string
}

func NewTongYiProvider(apiKey, model string) Provider {
	return &TongYiProvider{
		apiKey: apiKey,
		model:  model,
	}
}

func (p *TongYiProvider) Name() string {
	return "tongyi"
}

func (p *TongYiProvider) Endpoint() string {
	return "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
}

func (p *TongYiProvider) Headers() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}
}

func (p *TongYiProvider) PrepareRequest(prompt string, options map[string]interface{}) ([]byte, error) {
	requestBody := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	for k, v := range options {
		requestBody[k] = v
	}

	return json.Marshal(requestBody)
}

func (p *TongYiProvider) ParseResponse(body []byte) (string, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Choices[0].Message.Content, nil
}
