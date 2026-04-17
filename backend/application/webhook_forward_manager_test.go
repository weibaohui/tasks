package application

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWebhookGitHubManager_NewWebhookGitHubManager(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		expected  string
	}{
		{
			name:      "normal URL",
			serverURL: "https://example.com/api/v1",
			expected:  "https://example.com",
		},
		{
			name:      "URL without trailing",
			serverURL: "https://example.com",
			expected:  "https://example.com",
		},
		{
			name:      "URL with trailing slash",
			serverURL: "https://example.com/",
			expected:  "https://example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewWebhookGitHubManager(tt.serverURL)
			if mgr.serverURL != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, mgr.serverURL)
			}
		})
	}
}

func TestWebhookGitHubManager_UpdateServerURL(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://old.example.com")

	mgr.UpdateServerURL("https://new.example.com/api/v1")
	if mgr.serverURL != "https://new.example.com" {
		t.Errorf("expected https://new.example.com, got %s", mgr.serverURL)
	}

	mgr.UpdateServerURL("https://another.example.com/")
	if mgr.serverURL != "https://another.example.com/" {
		t.Errorf("expected https://another.example.com/, got %s", mgr.serverURL)
	}
}

func TestWebhookGitHubManager_BuildWebhookURL(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")

	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "short repo format",
			repo:     "owner/repo",
			expected: "https://example.com/api/v1/webhook/repos/owner/repo",
		},
		{
			name:     "full GitHub URL",
			repo:     "https://github.com/owner/repo",
			expected: "https://example.com/api/v1/webhook/repos/owner/repo",
		},
		{
			name:     "full GitHub URL with .git",
			repo:     "https://github.com/owner/repo.git",
			expected: "https://example.com/api/v1/webhook/repos/owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := mgr.BuildWebhookURL(tt.repo)
			if url != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, url)
			}
		})
	}
}

func TestWebhookGitHubManager_BuildWebhookURL_Concurrent(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")

	var wg sync.WaitGroup
	resultChan := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			url := mgr.BuildWebhookURL("owner/repo")
			resultChan <- url
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for url := range resultChan {
		expected := "https://example.com/api/v1/webhook/repos/owner/repo"
		if url != expected {
			t.Errorf("expected %s, got %s", expected, url)
		}
	}
}

func TestWebhookGitHubManager_UpdateServerURL_Concurrent(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			newURL := "https://server-" + string(rune('0'+idx%10)) + ".example.com"
			mgr.UpdateServerURL(newURL)
		}(i)
	}

	close(start)
	wg.Wait()

	if mgr.serverURL == "" {
		t.Error("serverURL should not be empty after concurrent updates")
	}
}

func TestWebhookGitHubManager_CheckAndUpdateWebhook(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")

	ctx := context.Background()

	needsUpdate, currentURL, err := mgr.CheckAndUpdateWebhook(ctx, "owner/non-existent-repo")
	if err != nil {
		t.Logf("expected error for non-existent repo: %v", err)
	}
	_ = needsUpdate
	_ = currentURL
}

func TestWebhookGitHubManager_ThreadSafety(t *testing.T) {
	mgr := NewWebhookGitHubManager("https://example.com")

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				mgr.UpdateServerURL("https://server-" + time.Now().Format(time.RFC3339Nano) + ".example.com")
				mgr.BuildWebhookURL("owner/repo")
			}
		}()
	}

	close(start)
	wg.Wait()
}

func TestNormalizeRepo(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"owner/repo", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo.git", "owner/repo"},
		{"git@github.com:owner/repo.git", "git@github.com:owner/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeRepo(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeRepo(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}
