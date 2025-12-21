package tensorzero

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nannyagent/nannyapi/internal/types"
)

// Client manages TensorZero API interactions
type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewClient creates a new TensorZero client
func NewClient() *Client {
	baseURL := os.Getenv("TENSORZERO_API_URL")
	if baseURL == "" {
		baseURL = "https://tensorzero-api.nannyai.dev"
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  os.Getenv("TENSORZERO_API_KEY"),
		client:  &http.Client{Timeout: 300}, // 5 minute timeout for long operations
	}
}

// CallChatCompletion calls TensorZero Core API for investigation
func (c *Client) CallChatCompletion(messages []types.ChatMessage) (*types.TensorZeroResponse, error) {
	url := fmt.Sprintf("%s/openai/v1/chat/completions", c.baseURL)

	payload := map[string]interface{}{
		"model":    "tensorzero::function_name::diagnose_and_heal_application",
		"messages": messages,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TensorZero API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TensorZero API error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var tzResp types.TensorZeroResponse
	if err := json.Unmarshal(bodyBytes, &tzResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &tzResp, nil
}

// RetrieveEpisode retrieves episode data from TensorZero ClickHouse
func (c *Client) RetrieveEpisode(episodeID string) (*map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/episode/%s", c.baseURL, episodeID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TensorZero API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TensorZero API error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var episode map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &episode); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &episode, nil
}

// RetrieveInference retrieves inference details from TensorZero ClickHouse
func (c *Client) RetrieveInference(inferenceID string) (*map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/inference/%s", c.baseURL, inferenceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call TensorZero API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TensorZero API error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var inference map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &inference); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &inference, nil
}
