package versioncheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

type Service struct {
	repoOwner      string
	repoName       string
	currentVersion string
	httpClient     *http.Client

	mu     sync.RWMutex
	latest string
}

func New(repoOwner, repoName, currentVersion string) *Service {
	return &Service{
		repoOwner:      repoOwner,
		repoName:       repoName,
		currentVersion: currentVersion,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func (s *Service) Refresh(ctx context.Context) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest", s.repoOwner, s.repoName,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.Error("version check: build request failed", logger.KV{"error": err})
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error("version check: request failed", logger.KV{"error": err})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("version check: unexpected status", logger.KV{"status": resp.StatusCode})
		return
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		logger.Error("version check: decode failed", logger.KV{"error": err})
		return
	}
	if release.TagName == "" {
		return
	}

	s.mu.Lock()
	s.latest = release.TagName
	s.mu.Unlock()
}

func (s *Service) LatestVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
}

func (s *Service) CurrentVersion() string {
	return s.currentVersion
}

func (s *Service) HasUpdate() bool {
	s.mu.RLock()
	latest := s.latest
	s.mu.RUnlock()

	if latest == "" {
		return false
	}

	cur, ok := parseVersion(s.currentVersion)
	if !ok {
		return false
	}
	lat, ok := parseVersion(latest)
	if !ok {
		return false
	}

	for i := range cur {
		if lat[i] != cur[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
