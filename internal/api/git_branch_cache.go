package api

import (
	"strings"
	"sync"
	"time"

	gitprovider "github.com/LiteyukiStudio/devops/internal/provider/git"
)

const gitBranchCacheTTL = 2 * time.Minute

type gitBranchCache struct {
	mu      sync.RWMutex
	entries map[string]gitBranchCacheEntry
}

type gitBranchCacheEntry struct {
	branches  []gitprovider.Branch
	expiresAt time.Time
}

func newGitBranchCache() *gitBranchCache {
	return &gitBranchCache{entries: map[string]gitBranchCacheEntry{}}
}

func (c *gitBranchCache) get(key string) ([]gitprovider.Branch, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.branches, true
}

func (c *gitBranchCache) set(key string, branches []gitprovider.Branch) {
	c.mu.Lock()
	c.entries[key] = gitBranchCacheEntry{branches: branches, expiresAt: time.Now().Add(gitBranchCacheTTL)}
	c.mu.Unlock()
}

func gitBranchCacheKey(accountID, owner, repo, ref string) string {
	return strings.Join([]string{
		strings.TrimSpace(accountID),
		strings.TrimSpace(owner),
		strings.TrimSpace(repo),
		strings.TrimSpace(ref),
	}, "\x00")
}

type gitBranchFilterResult struct {
	items        []gitprovider.Branch
	matchedTotal int
}

func filterGitBranches(branches []gitprovider.Branch, search string, limit int) gitBranchFilterResult {
	search = strings.ToLower(strings.TrimSpace(search))
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	capacity := len(branches)
	if capacity > limit {
		capacity = limit
	}
	filtered := make([]gitprovider.Branch, 0, capacity)
	matchedTotal := 0
	for _, branch := range branches {
		if search != "" && !strings.Contains(strings.ToLower(branch.Name), search) {
			continue
		}
		matchedTotal++
		if len(filtered) < limit {
			filtered = append(filtered, branch)
		}
	}
	return gitBranchFilterResult{items: filtered, matchedTotal: matchedTotal}
}
