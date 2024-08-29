package llm

import (
	"encoding/json"
	"fmt"
)

func RegisterZhiPuProvider(registry *ProviderRegistry) {
	registry.Register("zhipu", NewZhiPuProvider)
}

// ZhiPuProvider implements the Provider interface for OpenAI
type ZhiPuProvider struct {
	apiKey string
	model  string
}

func NewZhiPuProvider(apiKey, model string) Provider {
	return &ZhiPuProvider{
		apiKey: apiKey,
		model:  model,
	}
}

func (p *ZhiPuProvider) Name() string {
	return "zhipu"
}

func (p *ZhiPuProvider) Endpoint() string {
	return "https://open.bigmodel.cn/api/paas/v4/chat/completions"
}

func (p *ZhiPuProvider) Headers() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}
}

func (p *ZhiPuProvider) PrepareRequest(prompt string, options map[string]interface{}) ([]byte, error) {
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

func (p *ZhiPuProvider) ParseResponse(body []byte) (string, error) {
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
