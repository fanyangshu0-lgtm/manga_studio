package assets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"manga-drama-studio/internal/domain"
)

type Service struct {
	root     string
	maxBytes int64
	http     *http.Client
}

func New(root string, maxBytes int64, client *http.Client) *Service {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	return &Service{root: root, maxBytes: maxBytes, http: client}
}

func (s *Service) DownloadVideo(ctx context.Context, sourceURL, projectID, runID, nodeID, assetID string) (domain.Asset, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return domain.Asset{}, errors.New("视频下载地址无效")
	}
	relative := filepath.ToSlash(filepath.Join(projectID, runID, nodeID, assetID+".mp4"))
	target := filepath.Join(s.root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.Asset{}, err
	}
	partial := target + ".part"
	_ = os.Remove(partial)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return domain.Asset{}, err
	}
	response, err := s.http.Do(request)
	if err != nil {
		return domain.Asset{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.Asset{}, fmt.Errorf("下载视频失败（状态 %d）", response.StatusCode)
	}
	mime := strings.ToLower(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(mime, "video/") && mime != "application/octet-stream" {
		return domain.Asset{}, errors.New("下载内容不是视频")
	}
	file, err := os.OpenFile(partial, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return domain.Asset{}, err
	}
	cleanup := func() { file.Close(); _ = os.Remove(partial) }
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, s.maxBytes+1))
	if copyErr != nil || written > s.maxBytes {
		cleanup()
		if copyErr != nil {
			return domain.Asset{}, copyErr
		}
		return domain.Asset{}, errors.New("视频文件超过下载大小限制")
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return domain.Asset{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(partial)
		return domain.Asset{}, err
	}
	if err := os.Rename(partial, target); err != nil {
		_ = os.Remove(partial)
		return domain.Asset{}, err
	}
	return domain.Asset{ID: assetID, ProjectID: projectID, RunID: runID, NodeID: nodeID, Kind: "video", Path: relative, MIME: mime, Size: written, SHA256: hex.EncodeToString(hash.Sum(nil)), CreatedAt: time.Now().UTC()}, nil
}

func (s *Service) Path(asset domain.Asset) (string, error) {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(asset.Path)))
	if err != nil {
		return "", err
	}
	if target != root && !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return "", errors.New("资产路径越界")
	}
	return target, nil
}

func (s *Service) Verify(asset domain.Asset) bool {
	path, err := s.Path(asset)
	if err != nil {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() != asset.Size {
		return false
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return hex.EncodeToString(hash.Sum(nil)) == asset.SHA256
}

func (s *Service) Target(projectID, runID, nodeID, assetID string) (string, string, error) {
	relative := filepath.ToSlash(filepath.Join(projectID, runID, nodeID, assetID+".mp4"))
	target := filepath.Join(s.root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", "", err
	}
	return relative, target, nil
}

func (s *Service) RegisterVideo(relative, target, assetID, projectID, runID, nodeID string) (domain.Asset, error) {
	file, err := os.Open(target)
	if err != nil {
		return domain.Asset{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return domain.Asset{}, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return domain.Asset{}, err
	}
	return domain.Asset{ID: assetID, ProjectID: projectID, RunID: runID, NodeID: nodeID, Kind: "video", Path: relative, MIME: "video/mp4", Size: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil)), CreatedAt: time.Now().UTC()}, nil
}
