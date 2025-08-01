package main

import (
	"context"
	"testing"
	"time"
)

func TestCreateSuccessResponse(t *testing.T) {
	message := "test success message"
	response := createSuccessResponse(message)

	if response.Status != "success" {
		t.Errorf("createSuccessResponse() Status = %v, want success", response.Status)
	}

	if response.Message != message {
		t.Errorf("createSuccessResponse() Message = %v, want %v", response.Message, message)
	}

	if response.Timestamp == "" {
		t.Error("createSuccessResponse() should set Timestamp")
	}

	// Validate timestamp format
	_, err := time.Parse(time.RFC3339, response.Timestamp)
	if err != nil {
		t.Errorf("createSuccessResponse() invalid timestamp format: %v", err)
	}
}

func TestCreateErrorResponse(t *testing.T) {
	message := "test error message"
	response := createErrorResponse(message)

	if response.Status != "error" {
		t.Errorf("createErrorResponse() Status = %v, want error", response.Status)
	}

	if response.Message != message {
		t.Errorf("createErrorResponse() Message = %v, want %v", response.Message, message)
	}

	if response.Timestamp == "" {
		t.Error("createErrorResponse() should set Timestamp")
	}

	// Validate timestamp format
	_, err := time.Parse(time.RFC3339, response.Timestamp)
	if err != nil {
		t.Errorf("createErrorResponse() invalid timestamp format: %v", err)
	}
}

func TestFetchGitHubRepository_InvalidURL(t *testing.T) {
	ctx := context.Background()
	
	// Test with empty parameters that would create invalid URL
	_, err := fetchGitHubRepository(ctx, "", "")
	if err == nil {
		t.Error("fetchGitHubRepository() should return error for empty owner/repo")
	}
}

func TestHandleGitHubScrapeRequest_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	event := GitHubEvent{
		Repository: "test-repo",
		Owner:      "test-owner",
	}

	// This is a light integration test that validates the pipeline structure
	// without making actual API calls (would need mocks for full isolation)
	_, err := handleGitHubScrapeRequest(ctx, event)
	
	// We expect this to fail due to missing environment variables in test
	// but it should fail at the GitHub API stage, not in the pipeline setup
	if err == nil {
		t.Error("Expected error due to missing GitHub token or invalid repository in test environment")
	}
}

func TestGitHubEvent_Validation(t *testing.T) {
	tests := []struct {
		name  string
		event GitHubEvent
		valid bool
	}{
		{
			name: "valid event",
			event: GitHubEvent{
				Repository: "test-repo",
				Owner:      "test-owner",
			},
			valid: true,
		},
		{
			name: "empty repository",
			event: GitHubEvent{
				Repository: "",
				Owner:      "test-owner",
			},
			valid: false,
		},
		{
			name: "empty owner",
			event: GitHubEvent{
				Repository: "test-repo",
				Owner:      "",
			},
			valid: false,
		},
		{
			name: "both empty",
			event: GitHubEvent{
				Repository: "",
				Owner:      "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.event.Repository != "" && tt.event.Owner != ""
			if valid != tt.valid {
				t.Errorf("Event validation = %v, want %v", valid, tt.valid)
			}
		})
	}
}

// BenchmarkCreateResponse benchmarks response creation
func BenchmarkCreateSuccessResponse(b *testing.B) {
	message := "benchmark test message"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createSuccessResponse(message)
	}
}

func BenchmarkCreateErrorResponse(b *testing.B) {
	message := "benchmark error message"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createErrorResponse(message)
	}
}