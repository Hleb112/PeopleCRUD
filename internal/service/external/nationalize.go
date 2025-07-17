package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Country struct {
	CountryID   string  `json:"country_id"`
	Probability float64 `json:"probability"`
}

type NationalizeResponse struct {
	Name    string    `json:"name"`
	Country []Country `json:"country"`
}

type NationalizeClient struct {
	client  *http.Client
	baseURL string
}

func NewNationalizeClient() *NationalizeClient {
	return &NationalizeClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.nationalize.io",
	}
}

func (c *NationalizeClient) GetNationality(ctx context.Context, name string) (*string, error) {
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

	var nationalizeResp NationalizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&nationalizeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(nationalizeResp.Country) == 0 {
		return nil, nil
	}

	// Возвращаем страну с наибольшей вероятностью
	bestCountry := nationalizeResp.Country[0]
	for _, country := range nationalizeResp.Country {
		if country.Probability > bestCountry.Probability {
			bestCountry = country
		}
	}

	if bestCountry.Probability < 0.1 {
		return nil, nil
	}

	return &bestCountry.CountryID, nil
}
