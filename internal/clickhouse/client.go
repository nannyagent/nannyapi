package clickhouse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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
// Uses CLICKHOUSE_URL which should contain all connection details:
// Format: https://user:password@host:port/database
// Or: https://host:port?user=xxx&password=xxx&database=xxx
// Legacy env vars (CLICKHOUSE_DATABASE, CLICKHOUSE_USER, CLICKHOUSE_PASSWORD) are still supported for backwards compatibility
func NewClient() *Client {
	clickhouseURL := os.Getenv("CLICKHOUSE_URL")
	if clickhouseURL == "" {
		panic("CLICKHOUSE_URL environment variable is required")
	}

	// Also verify TensorZero credentials are set (required for the system)
	if os.Getenv("TENSORZERO_API_URL") == "" {
		panic("TENSORZERO_API_URL environment variable is required")
	}
	if os.Getenv("TENSORZERO_API_KEY") == "" {
		panic("TENSORZERO_API_KEY environment variable is required")
	}

	// Parse the URL to extract components
	parsedURL, err := url.Parse(clickhouseURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid CLICKHOUSE_URL: %v", err))
	}

	// Extract user and password from URL userinfo or query params
	var user, password, database string

	// Check for userinfo in URL (https://user:pass@host)
	if parsedURL.User != nil {
		user = parsedURL.User.Username()
		password, _ = parsedURL.User.Password()
	}

	// Extract database from path (https://host/database)
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		database = strings.TrimPrefix(parsedURL.Path, "/")
	}

	// Check query params for additional settings (override userinfo if present)
	queryParams := parsedURL.Query()
	if qUser := queryParams.Get("user"); qUser != "" {
		user = qUser
	}
	if qPassword := queryParams.Get("password"); qPassword != "" {
		password = qPassword
	}
	if qDatabase := queryParams.Get("database"); qDatabase != "" {
		database = qDatabase
	}

	// Fall back to legacy env vars if not in URL
	if user == "" {
		user = getEnv("CLICKHOUSE_USER", "default")
	}
	if password == "" {
		password = os.Getenv("CLICKHOUSE_PASSWORD")
		if password == "" {
			panic("ClickHouse password is required. Set it in CLICKHOUSE_URL or CLICKHOUSE_PASSWORD")
		}
	}
	if database == "" {
		database = getEnv("CLICKHOUSE_DATABASE", "tensorzero")
	}

	// Build base URL without auth info for requests
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	return &Client{
		baseURL:  baseURL,
		database: database,
		user:     user,
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
	defer func() { _ = resp.Body.Close() }()

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
