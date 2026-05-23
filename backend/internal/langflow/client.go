// Package langflow provides a client for the Langflow pipeline runtime.
// Cinova delegates chat pipeline execution to Langflow, which provides visual
// pipeline observability and prompt editability without code changes.
package langflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dynolabs-io/cinova/backend/internal/models"
)

// Client calls the Langflow run API.
type Client struct {
	url    string
	flowID string
	apiKey string
	hc     *http.Client
}

// New creates a Langflow client. apiKey may be empty if Langflow is not protected.
func New(url, flowID, apiKey string) *Client {
	return &Client{
		url:    url,
		flowID: flowID,
		apiKey: apiKey,
		hc:     &http.Client{Timeout: 120 * time.Second},
	}
}

// PipelineInput is the structured context passed to the Langflow flow as input_value.
// The CinovaChatPipeline component JSON-decodes this from the message string.
type PipelineInput struct {
	Message   string               `json:"message"`
	Country   string               `json:"country"`
	SessionID string               `json:"session_id"`
	ConvID    string               `json:"conv_id"`
	UserID    string               `json:"user_id"`
	History   []models.ChatMessage `json:"history"`
}

// PipelineOutput is the JSON the Langflow component returns as its message text.
type PipelineOutput struct {
	Reply       string               `json:"reply"`
	Suggestions []models.MovieSummary `json:"suggestions"`
	ConvID      string               `json:"conv_id"`
}

// runRequest mirrors the Langflow /api/v1/run/{flow_id} body.
type runRequest struct {
	InputValue string                       `json:"input_value"`
	OutputType string                       `json:"output_type"`
	InputType  string                       `json:"input_type"`
	Tweaks     map[string]map[string]string `json:"tweaks,omitempty"`
}

// runResponse is the shape of the Langflow run response.
type runResponse struct {
	Outputs []struct {
		Outputs []struct {
			Results map[string]json.RawMessage `json:"results"`
		} `json:"outputs"`
	} `json:"outputs"`
}

// Run executes the Langflow chat pipeline flow synchronously and returns the result.
func (c *Client) Run(ctx context.Context, input PipelineInput) (*PipelineOutput, error) {
	// Encode the structured input as a JSON string — the component parses this.
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("langflow: marshal pipeline input: %w", err)
	}

	inputStr := string(inputJSON)
	body, err := json.Marshal(runRequest{
		InputValue: inputStr,
		OutputType: "text",
		InputType:  "text",
		// Pass the input explicitly as a tweak so the component receives it
		// regardless of the flow's default routing.
		Tweaks: map[string]map[string]string{
			"CinovaChatPipeline-0001": {"input_value": inputStr},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("langflow: marshal run request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/run/%s?stream=false", c.url, c.flowID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("langflow: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("langflow: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("langflow: status %d: %s", resp.StatusCode, b)
	}

	var runResp runResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("langflow: decode response: %w", err)
	}

	text, err := extractText(runResp)
	if err != nil {
		return nil, err
	}

	var out PipelineOutput
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("langflow: parse component output: %w (raw: %.200s)", err, text)
	}
	return &out, nil
}

// extractText navigates the Langflow run response to find the component's output text.
// Langflow v1.3.x nests the output as outputs[0].outputs[0].results["message"].
func extractText(resp runResponse) (string, error) {
	if len(resp.Outputs) == 0 || len(resp.Outputs[0].Outputs) == 0 {
		return "", fmt.Errorf("langflow: empty outputs in response")
	}
	results := resp.Outputs[0].Outputs[0].Results

	// Try "message" key (standard for ChatOutput and custom Message returns).
	if msgRaw, ok := results["message"]; ok {
		// Shape 1: {"text": "..."}
		var s1 struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(msgRaw, &s1) == nil && s1.Text != "" {
			return s1.Text, nil
		}
		// Shape 2: {"data": {"text": "..."}}
		var s2 struct {
			Data struct {
				Text string `json:"text"`
			} `json:"data"`
		}
		if json.Unmarshal(msgRaw, &s2) == nil && s2.Data.Text != "" {
			return s2.Data.Text, nil
		}
	}

	// Fallback: "text" key.
	if textRaw, ok := results["text"]; ok {
		var s string
		if json.Unmarshal(textRaw, &s) == nil && s != "" {
			return s, nil
		}
	}

	return "", fmt.Errorf("langflow: no text found in results (keys: %v)", resultKeys(results))
}

func resultKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
