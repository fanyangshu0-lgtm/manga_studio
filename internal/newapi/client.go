package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 4 << 20

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type UpstreamError struct {
	StatusCode int
	RequestID  string
}

func (e *UpstreamError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("New API 请求失败（状态 %d，请求 ID %s）", e.StatusCode, e.RequestID)
	}
	return fmt.Sprintf("New API 请求失败（状态 %d）", e.StatusCode)
}

func New(baseURL, token string, client *http.Client) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	return &Client{baseURL: baseURL, token: token, http: client}
}

func (c *Client) ChatJSON(ctx context.Context, model string, messages []Message, output any) error {
	requestBody := struct {
		Model          string    `json:"model"`
		Messages       []Message `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}{Model: model, Messages: messages}
	requestBody.ResponseFormat.Type = "json_object"
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return &UpstreamError{StatusCode: response.StatusCode, RequestID: response.Header.Get("X-Request-Id")}
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes+1))
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("解析 New API 响应失败: %w", err)
	}
	if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return errors.New("New API 未返回内容")
	}
	content := strings.TrimSpace(envelope.Choices[0].Message.Content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(strings.TrimSpace(content), "```")
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), output); err != nil {
		return fmt.Errorf("模型返回的 JSON 无效: %w", err)
	}
	return nil
}
