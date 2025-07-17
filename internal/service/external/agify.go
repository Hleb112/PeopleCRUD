package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AgifyResponse struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Count int    `json:"count"`
}

type AgifyClient struct {
	client  *http.Client
	baseURL string
}

func NewAgifyClient() *AgifyClient {
	return &AgifyClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.agify.io",
	}
}

func (c *AgifyClient) GetAge(ctx context.Context, name string) (*int, error) {
	url := fmt.Sprintf("%s?name=%s", c.baseURL, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var agifyResp AgifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&agifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if agifyResp.Age == 0 {
		return nil, nil
	}

	return &agifyResp.Age, nil
}
