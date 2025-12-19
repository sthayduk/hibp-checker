package hibp

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles HTTP requests to the HIBP API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new HIBP API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.pwnedpasswords.com/range",
	}
}

// QueryRangeRaw queries the HIBP API with a hash prefix and returns raw response
// This avoids building a map of all results, saving memory
func (c *Client) QueryRangeRaw(hashPrefix string) (string, error) {
	url := fmt.Sprintf("%s/%s?mode=ntlm", c.baseURL, hashPrefix)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to query HIBP API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HIBP API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}
