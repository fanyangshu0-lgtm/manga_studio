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
