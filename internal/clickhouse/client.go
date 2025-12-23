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
// FAILS if CLICKHOUSE_URL, CLICKHOUSE_PASSWORD, TENSORZERO_API_URL, or TENSORZERO_API_KEY are not set
func NewClient() *Client {
	baseURL := os.Getenv("CLICKHOUSE_URL")
	if baseURL == "" {
		panic("CLICKHOUSE_URL environment variable is required")
	}

	password := os.Getenv("CLICKHOUSE_PASSWORD")
	if password == "" {
		panic("CLICKHOUSE_PASSWORD environment variable is required")
	}

	// Also verify TensorZero credentials are set (required for the system)
	if os.Getenv("TENSORZERO_API_URL") == "" {
		panic("TENSORZERO_API_URL environment variable is required")
	}
	if os.Getenv("TENSORZERO_API_KEY") == "" {
		panic("TENSORZERO_API_KEY environment variable is required")
	}

	return &Client{
		baseURL:  baseURL,
		database: getEnv("CLICKHOUSE_DATABASE", "tensorzero"),
		user:     getEnv("CLICKHOUSE_USER", "default"),
		password: password,
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
			processing_time_ms,
			input,
			output
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
			if len(row) >= 7 {
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

				// Parse Input JSON
				if len(row) > 5 && row[5] != nil {
					if inputStr, ok := row[5].(string); ok {
						var inputMap map[string]interface{}
						// Try direct unmarshal
						if err := json.Unmarshal([]byte(inputStr), &inputMap); err == nil {
							inference.Input = inputMap
						} else {
							// Try array wrapper (TensorZero format)
							var inputArray []map[string]interface{}
							if err := json.Unmarshal([]byte(inputStr), &inputArray); err == nil && len(inputArray) > 0 {
								if text, ok := inputArray[0]["text"].(string); ok {
									// Try to unmarshal the inner text
									var innerMap map[string]interface{}
									if err := json.Unmarshal([]byte(text), &innerMap); err == nil {
										inference.Input = innerMap
									}
								} else if content, ok := inputArray[0]["content"].(string); ok {
									// Sometimes it might be "content"
									var innerMap map[string]interface{}
									if err := json.Unmarshal([]byte(content), &innerMap); err == nil {
										inference.Input = innerMap
									}
								}
							}
						}
					} else if inputMap, ok := row[5].(map[string]interface{}); ok {
						inference.Input = inputMap
					}
				}

				// Parse Output JSON
				if len(row) > 6 && row[6] != nil {
					if outputStr, ok := row[6].(string); ok {
						var outputMap map[string]interface{}
						// Try direct unmarshal
						if err := json.Unmarshal([]byte(outputStr), &outputMap); err == nil {
							inference.Output = outputMap
						} else {
							// Try array wrapper (TensorZero format)
							var outputArray []map[string]interface{}
							if err := json.Unmarshal([]byte(outputStr), &outputArray); err == nil && len(outputArray) > 0 {
								if text, ok := outputArray[0]["text"].(string); ok {
									// Try to unmarshal the inner text
									var innerMap map[string]interface{}
									if err := json.Unmarshal([]byte(text), &innerMap); err == nil {
										inference.Output = innerMap
									}
								}
							}
						}
					} else if outputMap, ok := row[6].(map[string]interface{}); ok {
						inference.Output = outputMap
					}
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

	// Parse ClickHouse response - could be JSONCompact (array) or JSON object
	var result [][]interface{}

	// First, try to unmarshal as JSONCompact (array of arrays)
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// If that fails, try to parse as JSON object with "data" field
		var responseObj map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &responseObj); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Extract data array if present
		if dataField, ok := responseObj["data"]; ok {
			dataBytes, _ := json.Marshal(dataField)
			if err := json.Unmarshal(dataBytes, &result); err != nil {
				return nil, fmt.Errorf("failed to parse data field: %w", err)
			}
		} else {
			// No data field, return empty result
			result = [][]interface{}{}
		}
	}

	return result, nil
}
