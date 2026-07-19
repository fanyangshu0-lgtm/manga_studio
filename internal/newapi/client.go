package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type VideoRequest struct {
	Model    string         `json:"model"`
	Prompt   string         `json:"prompt"`
	Seconds  int            `json:"seconds"`
	Metadata map[string]any `json:"metadata"`
}

type VideoTask struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	VideoURL string `json:"videoUrl,omitempty"`
	Error    string `json:"error,omitempty"`
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

func (c *Client) SubmitVideo(ctx context.Context, input VideoRequest) (VideoTask, error) {
	var task VideoTask
	if err := c.doJSON(ctx, http.MethodPost, "/v1/video/generations", input, &task); err != nil {
		return VideoTask{}, err
	}
	if task.ID == "" {
		return VideoTask{}, errors.New("Seedance 未返回任务 ID")
	}
	return task, nil
}

func (c *Client) GetVideoTask(ctx context.Context, id string) (VideoTask, error) {
	var response struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Error    any    `json:"error"`
		VideoURL string `json:"video_url"`
		Output   struct {
			VideoURL string `json:"video_url"`
			URL      string `json:"url"`
		} `json:"output"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/video/generations/"+url.PathEscape(id), nil, &response); err != nil {
		return VideoTask{}, err
	}
	videoURL := response.VideoURL
	if videoURL == "" {
		videoURL = response.Output.VideoURL
	}
	if videoURL == "" {
		videoURL = response.Output.URL
	}
	return VideoTask{ID: response.ID, Status: strings.ToLower(response.Status), VideoURL: videoURL, Error: fmt.Sprint(response.Error)}, nil
}

func (c *Client) WaitVideo(ctx context.Context, id string, pollInterval, timeout time.Duration) (VideoTask, error) {
	waitContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		task, err := c.GetVideoTask(waitContext, id)
		if err != nil {
			return VideoTask{}, err
		}
		switch task.Status {
		case "completed", "succeeded":
			if task.VideoURL == "" {
				return VideoTask{}, errors.New("Seedance 任务完成但没有视频地址")
			}
			return task, nil
		case "failed", "cancelled", "canceled":
			return VideoTask{}, fmt.Errorf("Seedance 任务失败（状态 %s）", task.Status)
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-waitContext.Done():
			timer.Stop()
			return VideoTask{}, waitContext.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		payload, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
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
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("解析 New API 响应失败: %w", err)
	}
	return nil
}
