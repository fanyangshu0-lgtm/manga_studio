package newapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatJSONUsesApprovedModelAndBearerToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		var body struct {
			Model          string `json:"model"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "deepseek-v4-pro", body.Model)
		assert.Equal(t, "json_object", body.ResponseFormat.Type)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"测试\"}"}}]}`))
	}))
	defer upstream.Close()

	var output struct {
		Title string `json:"title"`
	}
	err := New(upstream.URL+"/v1/", "sk-test", upstream.Client()).ChatJSON(context.Background(), "deepseek-v4-pro", []Message{{Role: "user", Content: "write"}}, &output)
	require.NoError(t, err)
	assert.Equal(t, "测试", output.Title)
}

func TestChatJSONAcceptsFencedJSONAndReturnsSafeUpstreamErrors(t *testing.T) {
	t.Run("fenced json", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"ok\\\":true}\\n```\"}}]}"))
		}))
		defer upstream.Close()
		var output struct {
			OK bool `json:"ok"`
		}
		require.NoError(t, New(upstream.URL, "secret-token", upstream.Client()).ChatJSON(context.Background(), "model", nil, &output))
		assert.True(t, output.OK)
	})

	t.Run("safe error", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Request-Id", "req-123")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`secret-token upstream private response`))
		}))
		defer upstream.Close()
		var output any
		err := New(upstream.URL, "secret-token", upstream.Client()).ChatJSON(context.Background(), "model", nil, &output)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "502")
		assert.Contains(t, err.Error(), "req-123")
		assert.NotContains(t, err.Error(), "secret-token")
		assert.NotContains(t, err.Error(), "private response")
	})
}

func TestSubmitAndGetVideoTask(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/video/generations":
			var body VideoRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "doubao-seedance-2-0-fast-260128", body.Model)
			assert.Equal(t, true, body.Metadata["generate_audio"])
			_, _ = w.Write([]byte(`{"id":"task-1","status":"queued"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/video/generations/task-1":
			_, _ = w.Write([]byte(`{"id":"task-1","status":"succeeded","output":{"video_url":"https://cdn.example/video.mp4"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	client := New(upstream.URL, "sk-test", upstream.Client())
	task, err := client.SubmitVideo(context.Background(), VideoRequest{Model: "doubao-seedance-2-0-fast-260128", Prompt: "镜头", Seconds: "5", Metadata: map[string]any{"generate_audio": true}})
	require.NoError(t, err)
	assert.Equal(t, "task-1", task.ID)
	task, err = client.GetVideoTask(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example/video.mp4", task.VideoURL)
}
