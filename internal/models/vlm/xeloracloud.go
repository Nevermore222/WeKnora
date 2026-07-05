package vlm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Tencent/Xelora/internal/logger"
	"github.com/Tencent/Xelora/internal/models/utils"
	"github.com/google/uuid"
)

const xeloraCloudVLMPath = "/api/v1/chat/completions"

// XeloraCloudVLM implements VLM via the XeloraCloud API.
type XeloraCloudVLM struct {
	modelName       string
	remoteModelName string
	modelID         string
	appID           string
	apiKey          string
	baseURL         string
	client          *http.Client
}

// NewXeloraCloudVLM creates a XeloraCloud-backed VLM instance.
func NewXeloraCloudVLM(config *Config) (*XeloraCloudVLM, error) {
	if config.AppID == "" {
		return nil, fmt.Errorf("XeloraCloud VLM: AppID is required")
	}
	if config.AppSecret == "" {
		return nil, fmt.Errorf("XeloraCloud VLM: AppSecret is required")
	}
	remoteModelName := ""
	if config.Extra != nil {
		if v, ok := config.Extra["remote_model_name"]; ok {
			if vs, ok := v.(string); ok {
				remoteModelName = strings.TrimSpace(vs)
			}
		}
	}
	return &XeloraCloudVLM{
		modelName:       config.ModelName,
		remoteModelName: remoteModelName,
		modelID:         config.ModelID,
		appID:           config.AppID,
		apiKey:          config.AppSecret,
		baseURL:         strings.TrimRight(config.BaseURL, "/"),
		client:          &http.Client{Timeout: vlmHTTPTimeout()},
	}, nil
}

type xeloraCloudVLMContentPart struct {
	Type     string                      `json:"type"`
	Text     string                      `json:"text,omitempty"`
	ImageURL *xeloraCloudVLMImageURL    `json:"image_url,omitempty"`
}

type xeloraCloudVLMImageURL struct {
	URL string `json:"url"`
}

type xeloraCloudVLMMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type xeloraCloudVLMRequest struct {
	Model       string                   `json:"model"`
	Messages    []xeloraCloudVLMMessage `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	Stream      bool                     `json:"stream"`
}

type xeloraCloudVLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Predict sends images with a text prompt to the XeloraCloud API.
func (v *XeloraCloudVLM) Predict(ctx context.Context, imgBytesList [][]byte, prompt string) (string, error) {
	var parts []xeloraCloudVLMContentPart

	parts = append(parts, xeloraCloudVLMContentPart{
		Type: "text",
		Text: prompt,
	})

	for _, imgBytes := range imgBytesList {
		if len(imgBytes) > 0 {
			mimeType := detectImageMIME(imgBytes)
			b64 := base64.StdEncoding.EncodeToString(imgBytes)
			dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
			parts = append(parts, xeloraCloudVLMContentPart{
				Type: "image_url",
				ImageURL: &xeloraCloudVLMImageURL{
					URL: dataURI,
				},
			})
		}
	}

	reqBody := xeloraCloudVLMRequest{
		Model: v.effectiveModelName(),
		Messages: []xeloraCloudVLMMessage{
			{
				Role:    "user",
				Content: parts,
			},
		},
		MaxTokens:   defaultMaxToks,
		Temperature: float64(defaultTemp),
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("xeloracloud VLM: marshal: %w", err)
	}

	requestID := uuid.New().String()
	headers := utils.Sign(v.appID, v.apiKey, requestID, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+xeloraCloudVLMPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("xeloracloud VLM: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, hv := range headers {
		req.Header.Set(k, hv)
	}

	totalImageSize := 0
	for _, img := range imgBytesList {
		totalImageSize += len(img)
	}
	logger.Infof(ctx, "[VLM] Calling XeloraCloud API, model=%s, baseURL=%s, numImages=%d, totalImageSize=%d",
		v.effectiveModelName(), v.baseURL, len(imgBytesList), totalImageSize)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("xeloracloud VLM: do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("xeloracloud VLM: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("xeloracloud VLM: status %d: %s", resp.StatusCode, string(respBytes))
	}

	var vlmResp xeloraCloudVLMResponse
	if err := json.Unmarshal(respBytes, &vlmResp); err != nil {
		return "", fmt.Errorf("xeloracloud VLM: unmarshal: %w", err)
	}
	if len(vlmResp.Choices) == 0 {
		return "", fmt.Errorf("xeloracloud VLM: no choices in response")
	}

	content := vlmResp.Choices[0].Message.Content
	logger.Infof(ctx, "[VLM] XeloraCloud response received, len=%d", len(content))
	return content, nil
}

func (v *XeloraCloudVLM) effectiveModelName() string {
	if v.remoteModelName != "" {
		return v.remoteModelName
	}
	return v.modelName
}

func (v *XeloraCloudVLM) GetModelName() string { return v.modelName }
func (v *XeloraCloudVLM) GetModelID() string   { return v.modelID }
