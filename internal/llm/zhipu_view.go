package llm

import (
	"encoding/json"
	"fmt"
)

func RegisterZhiPuViewProvider(registry *ProviderRegistry) {
	registry.Register("zhipu_view", NewZhiPuViewProvider)
}

// ZhiPuViewProvider implements the Provider interface for OpenAI
type ZhiPuViewProvider struct {
	apiKey string
	model  string
}

func NewZhiPuViewProvider(apiKey, model string) Provider {
	return &ZhiPuViewProvider{
		apiKey: apiKey,
		model:  model,
	}
}

func (p *ZhiPuViewProvider) Name() string {
	return "zhipu_view"
}

func (p *ZhiPuViewProvider) Endpoint() string {
	return "https://open.bigmodel.cn/api/paas/v4/images/generations"
}

func (p *ZhiPuViewProvider) Headers() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}
}

func (p *ZhiPuViewProvider) PrepareRequest(prompt string, options map[string]interface{}) ([]byte, error) {
	requestBody := map[string]interface{}{
		"model":  p.model,
		"prompt": prompt,
	}

	for k, v := range options {
		requestBody[k] = v
	}

	return json.Marshal(requestBody)
}

func (p *ZhiPuViewProvider) ParseResponse(body []byte) (string, error) {
	var response struct {
		Created int64 `json:"created"`
		Data    []struct {
			Url string `json:"url"`
		} `json:"data"`
		ContentFilter []struct {
			Role  string `json:"role"`
			Level int32  `json:"level"`
		} `json:"content_filter"`
	}

	err := json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	if len(response.Data) == 0 || response.Data[0].Url == "" {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Data[0].Url, nil
}
