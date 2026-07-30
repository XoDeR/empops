package frankfurter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL string
}

func New() *Client {
	return &Client{httpClient: &http.Client{Timeout: 10 * time.Second}, baseURL: "https://api.frankfurter.app"}
}

func (c *Client) Rate(ctx context.Context, date, from, to string) (float64, error) {
	from, to = strings.ToUpper(from), strings.ToUpper(to)
	endpoint := fmt.Sprintf("%s/%s?from=%s&to=%s", c.baseURL, url.PathEscape(date), url.QueryEscape(from), url.QueryEscape(to))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil { return 0, err }
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil { return 0, fmt.Errorf("unable to retrieve exchange rate: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return 0, fmt.Errorf("unable to retrieve exchange rate") }
	var payload struct{ Rates map[string]float64 `json:"rates"` }
	if json.NewDecoder(resp.Body).Decode(&payload) != nil || payload.Rates[to] <= 0 {
		return 0, fmt.Errorf("unable to retrieve exchange rate")
	}
	return payload.Rates[to], nil
}
