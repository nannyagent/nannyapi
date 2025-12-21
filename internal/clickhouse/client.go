package clickhouse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/nannyagent/nannyapi/internal/types"
)

// Client manages ClickHouse queries for TensorZero data
type Client struct {
	baseURL  string
	database string
	user     string
	password string
	client   *http.Client
}

// NewClient creates a new ClickHouse client
func NewClient() *Client {
	baseURL := os.Getenv("CLICKHOUSE_URL")
	if baseURL == "" {
		baseURL = "https://clickhouse.nannyai.dev"
	}

	return &Client{
		baseURL:  baseURL,
		database: getEnv("CLICKHOUSE_DATABASE", "tensorzero"),
		user:     getEnv("CLICKHOUSE_USER", "default"),
		password: os.Getenv("CLICKHOUSE_PASSWORD"),
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// FetchInferencesByEpisode fetches all inferences for a given episode_id
func (c *Client) FetchInferencesByEpisode(episodeID string) ([]types.ClickHouseInference, error) {
	query := fmt.Sprintf(`
		SELECT
			id,
			function_name,
			variant_name,
			timestamp,
			processing_time_ms
		FROM ChatInference
		WHERE episode_id = '%s'
		ORDER BY timestamp ASC
		FORMAT JSONCompact
	`, episodeID)

	result, err := c.executeQuery(query)
	if err != nil {
		return nil, err
	}

	var inferences []types.ClickHouseInference
	if len(result) > 0 {
		for _, row := range result {
			if len(row) >= 5 {
				var timestamp time.Time
				if ts, ok := row[3].(string); ok {
					timestamp, _ = time.Parse(time.RFC3339, ts)
				}

				inference := types.ClickHouseInference{
					ID:               row[0].(string),
					FunctionName:     row[1].(string),
					VariantName:      row[2].(string),
					Timestamp:        timestamp,
					ProcessingTimeMs: int64(row[4].(float64)),
				}
				inferences = append(inferences, inference)
			}
		}
	}

	return inferences, nil
}

// FetchInferenceDetails fetches detailed information about a specific inference
func (c *Client) FetchInferenceDetails(inferenceID string) (*types.InferenceDetailsResponse, error) {
	query := fmt.Sprintf(`
		SELECT
			ci.id,
			ci.function_name,
			ci.variant_name,
			ci.episode_id,
			ci.input,
			ci.output,
			ci.processing_time_ms,
			ci.timestamp,
			mi.id as model_inference_id,
			mi.model_name,
			mi.model_provider_name,
			mi.input_tokens,
			mi.output_tokens,
			mi.response_time_ms,
			mi.ttft_ms
		FROM ChatInference ci
		LEFT JOIN ModelInference mi ON ci.id = mi.inference_id
		WHERE ci.id = '%s'
		FORMAT JSONCompact
	`, inferenceID)

	result, err := c.executeQuery(query)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("inference not found: %s", inferenceID)
	}

	// Parse first row (main inference)
	row := result[0]
	if len(row) < 8 {
		return nil, fmt.Errorf("invalid inference data")
	}

	var timestamp time.Time
	if ts, ok := row[7].(string); ok {
		timestamp, _ = time.Parse(time.RFC3339, ts)
	}

	response := &types.InferenceDetailsResponse{
		ID:               row[0].(string),
		FunctionName:     row[1].(string),
		VariantName:      row[2].(string),
		EpisodeID:        row[3].(string),
		ProcessingTimeMs: int64(row[6].(float64)),
		Timestamp:        timestamp,
		ModelInferences:  []types.ModelInference{},
	}

	// Parse model inferences (all rows may have model inference details)
	for _, r := range result {
		if len(r) >= 15 && r[8] != nil {
			modelID, ok := r[8].(string)
			if ok && modelID != "" {
				modelInf := types.ModelInference{
					ID:               modelID,
					ModelName:        r[9].(string),
					ModelProvider:    r[10].(string),
					InputTokens:      int(r[11].(float64)),
					OutputTokens:     int(r[12].(float64)),
					ResponseTimeMs:   int64(r[13].(float64)),
					TimeToFirstToken: int64(r[14].(float64)),
				}
				response.ModelInferences = append(response.ModelInferences, modelInf)
			}
		}
	}

	return response, nil
}

// executeQuery executes a ClickHouse query and returns the result
func (c *Client) executeQuery(query string) ([][]interface{}, error) {
	urlStr := c.baseURL
	params := url.Values{}
	params.Set("user", c.user)
	params.Set("password", c.password)
	params.Set("database", c.database)

	if len(params.Encode()) > 0 {
		urlStr = fmt.Sprintf("%s/?%s", urlStr, params.Encode())
	}

	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ClickHouse error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse JSONCompact response
	var result [][]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}
