package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GenderizeResponse struct {
	Name        string  `json:"name"`
	Gender      string  `json:"gender"`
	Probability float64 `json:"probability"`
	Count       int     `json:"count"`
}

type GenderizeClient struct {
	client  *http.Client
	baseURL string
}

func NewGenderizeClient() *GenderizeClient {
	return &GenderizeClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.genderize.io",
	}
}

func (c *GenderizeClient) GetGender(ctx context.Context, name string) (*string, error) {
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

	var genderizeResp GenderizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&genderizeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if genderizeResp.Gender == "" || genderizeResp.Probability < 0.5 {
		return nil, nil
	}

	return &genderizeResp.Gender, nil
}
