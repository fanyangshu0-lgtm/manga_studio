package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadVideoWritesAtomicallyAndHashesContent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("video-content"))
	}))
	defer upstream.Close()
	root := t.TempDir()
	service := New(root, 1024, upstream.Client())
	asset, err := service.DownloadVideo(context.Background(), upstream.URL, "project", "run", "node", "asset")
	require.NoError(t, err)
	assert.Equal(t, int64(len("video-content")), asset.Size)
	assert.NotEmpty(t, asset.SHA256)
	_, err = os.Stat(filepath.Join(root, filepath.FromSlash(asset.Path)))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, filepath.FromSlash(asset.Path)) + ".part")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestDownloadVideoRejectsNonVideoAndOversizedBodies(t *testing.T) {
	for _, test := range []struct{ name, contentType, body string }{
		{"mime", "text/plain", "not-video"},
		{"size", "video/mp4", "too-large"},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				_, _ = w.Write([]byte(test.body))
			}))
			defer upstream.Close()
			limit := int64(100)
			if test.name == "size" {
				limit = 2
			}
			_, err := New(t.TempDir(), limit, upstream.Client()).DownloadVideo(context.Background(), upstream.URL, "p", "r", "n", "a")
			assert.Error(t, err)
		})
	}
}
