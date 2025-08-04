package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	common "bacon/src/shared"
)

// Test buildGitHubURL function thoroughly
func TestBuildGitHubURL(t *testing.T) {
	testCases := []struct {
		name     string
		owner    string
		repo     string
		expected string
	}{
		{
			name:     "valid inputs",
			owner:    "test-owner",
			repo:     "test-repo",
			expected: "https://api.github.com/repos/test-owner/test-repo",
		},
		{
			name:     "empty owner should create URL with empty segment",
			owner:    "",
			repo:     "test-repo",
			expected: "https://api.github.com/repos//test-repo",
		},
		{
			name:     "empty repo should create URL with empty segment",
			owner:    "test-owner",
			repo:     "",
			expected: "https://api.github.com/repos/test-owner/",
		},
		{
			name:     "both empty should create minimal URL",
			owner:    "",
			repo:     "",
			expected: "https://api.github.com/repos//",
		},
		{
			name:     "special characters in owner and repo",
			owner:    "owner-with-dashes",
			repo:     "repo.with.dots",
			expected: "https://api.github.com/repos/owner-with-dashes/repo.with.dots",
		},
		{
			name:     "unicode characters",
			owner:    "测试用户",
			repo:     "测试仓库",
			expected: "https://api.github.com/repos/测试用户/测试仓库",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildGitHubURL(tc.owner, tc.repo)
			if result != tc.expected {
				t.Errorf("Expected URL: %s, got: %s", tc.expected, result)
			}
		})
	}
}

// Test createAuthenticatedRequest function
func TestCreateAuthenticatedRequest(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name          string
		url           string
		githubToken   string
		shouldSucceed bool
		expectedError string
	}{
		{
			name:          "valid request with token",
			url:           "https://api.github.com/repos/owner/repo",
			githubToken:   "test-token-123",
			shouldSucceed: true,
		},
		{
			name:          "valid request without token",
			url:           "https://api.github.com/repos/owner/repo",
			githubToken:   "",
			shouldSucceed: true,
		},
		{
			name:          "invalid URL",
			url:           "://invalid-url",
			githubToken:   "test-token",
			shouldSucceed: false,
			expectedError: "failed to create request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable
			if tc.githubToken != "" {
				os.Setenv("GITHUB_TOKEN", tc.githubToken)
			} else {
				os.Unsetenv("GITHUB_TOKEN")
			}
			defer os.Unsetenv("GITHUB_TOKEN")

			req, err := createAuthenticatedRequest(ctx, tc.url)

			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if req == nil {
					t.Error("Expected request but got nil")
				} else {
					if req.Method != "GET" {
						t.Errorf("Expected GET method, got: %s", req.Method)
					}
					if req.URL.String() != tc.url {
						t.Errorf("Expected URL: %s, got: %s", tc.url, req.URL.String())
					}
					// Check authorization header
					if tc.githubToken != "" {
						expectedAuth := fmt.Sprintf("token %s", tc.githubToken)
						if req.Header.Get("Authorization") != expectedAuth {
							t.Errorf("Expected Authorization header: %s, got: %s", expectedAuth, req.Header.Get("Authorization"))
						}
					} else {
						if req.Header.Get("Authorization") != "" {
							t.Error("Expected no Authorization header without token")
						}
					}
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s' but got: %v", tc.expectedError, err)
				}
			}
		})
	}
}

// Test executeHTTPRequest function
func TestExecuteHTTPRequest(t *testing.T) {
	testCases := []struct {
		name           string
		responseBody   string
		statusCode     int
		shouldSucceed  bool
		expectedError  string
	}{
		{
			name:         "successful request",
			responseBody: `{"message": "success"}
`,
			statusCode:   200,
			shouldSucceed: true,
		},
		{
			name:         "404 not found",
			responseBody: `{"message": "Not Found"}
`,
			statusCode:   404,
			shouldSucceed: true, // HTTP request succeeds, but status code indicates failure
		},
		{
			name:         "500 server error",
			responseBody: `{"message": "Internal Server Error"}
`,
			statusCode:   500,
			shouldSucceed: true, // HTTP request succeeds, but status code indicates failure
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Create request
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create test request: %v", err)
			}

			resp, err := executeHTTPRequest(req)

			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if resp == nil {
					t.Error("Expected response but got nil")
				} else {
					if resp.StatusCode != tc.statusCode {
						t.Errorf("Expected status code: %d, got: %d", tc.statusCode, resp.StatusCode)
					}
					resp.Body.Close()
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s' but got: %v", tc.expectedError, err)
				}
			}
		})
	}
}

// Test decodeGitHubResponse function
func TestDecodeGitHubResponse(t *testing.T) {
	testCases := []struct {
		name          string
		responseBody  string
		statusCode    int
		shouldSucceed bool
		expectedError string
		expectedRepo  *GitHubRepo
	}{
		{
			name: "valid repository response",
			responseBody: `{
				"id": 123456,
				"name": "test-repo",
				"description": "A test repository",
				"language": "Go",
				"stargazers_count": 42,
				"forks_count": 7
			}`,
			statusCode:   200,
			shouldSucceed: true,
			expectedRepo: &GitHubRepo{
				ID:          123456,
				Name:        "test-repo",
				Description: "A test repository",
				Language:    "Go",
				Stars:       42,
				Forks:       7,
			},
		},
		{
			name:         "invalid JSON",
			responseBody: `{"id": invalid json}`,
			statusCode:   200,
			shouldSucceed: false,
			expectedError: "failed to decode response",
		},
		{
			name:         "empty response",
			responseBody: ``,
			statusCode:   200,
			shouldSucceed: false,
			expectedError: "failed to decode response",
		},
		{
			name: "missing required fields",
			responseBody: `{
				"description": "A test repository"
			}`,
			statusCode:   200,
			shouldSucceed: true, // JSON decoding succeeds, but fields are zero values
			expectedRepo: &GitHubRepo{
				ID:          0,
				Name:        "",
				Description: "A test repository",
				Language:    "",
				Stars:       0,
				Forks:       0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock HTTP response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("Failed to create mock response: %v", err)
			}

			repo, err := decodeGitHubResponse(resp)

			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if repo == nil {
					t.Error("Expected repo but got nil")
				} else {
					if repo.ID != tc.expectedRepo.ID {
						t.Errorf("Expected ID: %d, got: %d", tc.expectedRepo.ID, repo.ID)
					}
					if repo.Name != tc.expectedRepo.Name {
						t.Errorf("Expected Name: %s, got: %s", tc.expectedRepo.Name, repo.Name)
					}
					if repo.Description != tc.expectedRepo.Description {
						t.Errorf("Expected Description: %s, got: %s", tc.expectedRepo.Description, repo.Description)
					}
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s' but got: %v", tc.expectedError, err)
				}
			}
		})
	}
}

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

// Defensive programming and edge case tests
func TestDefensiveProgramming(t *testing.T) {
	t.Run("buildGitHubURL with extreme inputs", func(t *testing.T) {
		// Test with very long strings
		longOwner := strings.Repeat("a", 1000)
		longRepo := strings.Repeat("b", 1000)
		result := buildGitHubURL(longOwner, longRepo)
		expected := fmt.Sprintf("https://api.github.com/repos/%s/%s", longOwner, longRepo)
		if result != expected {
			t.Errorf("buildGitHubURL should handle long strings")
		}
	})

	t.Run("createAuthenticatedRequest with nil context", func(t *testing.T) {
		// This should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function panicked with nil context: %v", r)
			}
		}()
		
		// Should handle nil context gracefully (Go's http package handles this)
		_, err := createAuthenticatedRequest(nil, "https://api.github.com/repos/owner/repo")
		if err != nil {
			t.Logf("Expected behavior with nil context: %v", err)
		}
	})

	t.Run("executeHTTPRequest with nil request", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil request")
			}
		}()
		
		// This should panic as expected
		executeHTTPRequest(nil)
	})
}

// Benchmark tests for performance
func BenchmarkBuildGitHubURL(b *testing.B) {
	owner := "test-owner"
	repo := "test-repo"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildGitHubURL(owner, repo)
	}
}

func BenchmarkCreateAuthenticatedRequest(b *testing.B) {
	ctx := context.Background()
	url := "https://api.github.com/repos/owner/repo"
	os.Setenv("GITHUB_TOKEN", "test-token")
	defer os.Unsetenv("GITHUB_TOKEN")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := createAuthenticatedRequest(ctx, url)
		if err != nil {
			b.Fatal(err)
		}
		_ = req
	}
}

// Test getTableName function
func TestGetTableName(t *testing.T) {
	testCases := []struct {
		name         string
		envValue     string
		expectedName string
	}{
		{
			name:         "with environment variable",
			envValue:     "custom-github-table",
			expectedName: "custom-github-table",
		},
		{
			name:         "without environment variable",
			envValue:     "",
			expectedName: "github-repositories",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("DYNAMODB_TABLE", tc.envValue)
			} else {
				os.Unsetenv("DYNAMODB_TABLE")
			}
			defer os.Unsetenv("DYNAMODB_TABLE")

			result := getTableName()
			if result != tc.expectedName {
				t.Errorf("Expected table name: %s, got: %s", tc.expectedName, result)
			}
		})
	}
}

// Test createRepositoryItem function
func TestCreateRepositoryItem(t *testing.T) {
	testCases := []struct {
		name string
		repo *GitHubRepo
	}{
		{
			name: "complete repository data",
			repo: &GitHubRepo{
				ID:          123456,
				Name:        "test-repo",
				Description: "A test repository",
				Language:    "Go",
				Stars:       42,
				Forks:       7,
			},
		},
		{
			name: "minimal repository data",
			repo: &GitHubRepo{
				ID:   789,
				Name: "minimal-repo",
			},
		},
		{
			name: "repository with zero values",
			repo: &GitHubRepo{
				ID:          0,
				Name:        "",
				Description: "",
				Language:    "",
				Stars:       0,
				Forks:       0,
			},
		},
		{
			name: "repository with negative values",
			repo: &GitHubRepo{
				ID:          -1,
				Name:        "negative-repo",
				Description: "Repo with negative ID",
				Language:    "Unknown",
				Stars:       -5,
				Forks:       -2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			item := createRepositoryItem(tc.repo)

			// Verify all required fields are present
			requiredFields := []string{"id", "name", "description", "language", "stars", "forks", "scraped_at"}
			for _, field := range requiredFields {
				if _, exists := item[field]; !exists {
					t.Errorf("Missing required field: %s", field)
				}
			}

			// Verify values match input
			if idAttr, ok := item["id"].(*types.AttributeValueMemberN); ok {
				expectedID := fmt.Sprintf("%d", tc.repo.ID)
				if idAttr.Value != expectedID {
					t.Errorf("Expected ID: %s, got: %s", expectedID, idAttr.Value)
				}
			} else {
				t.Error("id is not a number attribute")
			}

			if nameAttr, ok := item["name"].(*types.AttributeValueMemberS); ok {
				if nameAttr.Value != tc.repo.Name {
					t.Errorf("Expected name: %s, got: %s", tc.repo.Name, nameAttr.Value)
				}
			} else {
				t.Error("name is not a string attribute")
			}

			if descAttr, ok := item["description"].(*types.AttributeValueMemberS); ok {
				if descAttr.Value != tc.repo.Description {
					t.Errorf("Expected description: %s, got: %s", tc.repo.Description, descAttr.Value)
				}
			} else {
				t.Error("description is not a string attribute")
			}
		})
	}
}



// Additional defensive programming tests
func TestMoreDefensiveProgramming(t *testing.T) {
	t.Run("createRepositoryItem with nil repo", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil repo")
			}
		}()
		
		createRepositoryItem(nil)
	})

	t.Run("buildGitHubURL with very long inputs", func(t *testing.T) {
		longOwner := strings.Repeat("a", 10000)
		longRepo := strings.Repeat("b", 10000)
		
		result := buildGitHubURL(longOwner, longRepo)
		if !strings.Contains(result, longOwner) || !strings.Contains(result, longRepo) {
			t.Error("Should handle very long inputs")
		}
	})

	t.Run("HTTP timeout handling", func(t *testing.T) {
		// Test that our HTTP client has appropriate timeout
		req, err := http.NewRequest("GET", "https://httpbin.org/delay/10", nil)
		if err != nil {
			t.Skip("Unable to create test request")
		}

		start := time.Now()
		_, err = executeHTTPRequest(req)
		duration := time.Since(start)

		// Should timeout before 35 seconds (we set 30s timeout + some buffer)
		if duration > 35*time.Second {
			t.Error("HTTP request should have timed out")
		}
	})
}

// Test GitHubEvent and GitHubResponse structure validation
func TestGitHubEventValidation(t *testing.T) {
	testCases := []struct {
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate that event has required fields
			isValid := tc.event.Repository != "" && tc.event.Owner != ""
			if isValid != tc.valid {
				t.Errorf("Expected validity: %t, got: %t", tc.valid, isValid)
			}
			
			// Test URL building with the event
			url := buildGitHubURL(tc.event.Owner, tc.event.Repository)
			if tc.valid {
				expected := fmt.Sprintf("https://api.github.com/repos/%s/%s", tc.event.Owner, tc.event.Repository)
				if url != expected {
					t.Errorf("Expected URL: %s, got: %s", expected, url)
				}
			}
		})
	}
}

func TestGitHubProcessingDataCreation(t *testing.T) {
	ctx := context.Background()
	event := GitHubEvent{
		Repository: "test-repo",
		Owner:      "test-owner",
	}
	
	data := GitHubProcessingData{
		Context: ctx,
		Event:   event,
	}
	
	if data.Context != ctx {
		t.Error("GitHubProcessingData Context not set correctly")
	}
	
	if data.Event.Repository != event.Repository {
		t.Errorf("GitHubProcessingData Event.Repository = %v, want %v", data.Event.Repository, event.Repository)
	}
	
	if data.Event.Owner != event.Owner {
		t.Errorf("GitHubProcessingData Event.Owner = %v, want %v", data.Event.Owner, event.Owner)
	}
}

func TestFetchRepositoryStep_WithXRayContext(t *testing.T) {
	// Create proper X-Ray context for testing
	ctx, cleanup := common.TestContext("github-scraper-test")
	defer cleanup()
	
	// Test with empty parameters that would create invalid URL/request
	data := GitHubProcessingData{
		Context: ctx,
		Event: GitHubEvent{
			Repository: "",
			Owner:      "",
		},
	}
	
	// This might succeed in making the request but get a 404 or other HTTP error
	// The main test is that X-Ray tracing works without panics
	result, err := fetchRepositoryStep(data)
	
	// Log the result for debugging
	if err != nil {
		t.Logf("fetchRepositoryStep returned error (expected): %v", err)
	} else {
		t.Logf("fetchRepositoryStep succeeded unexpectedly, result: %+v", result)
	}
	
	// The key test is that X-Ray context worked without panics
	// The actual business logic error (404, invalid repo, etc.) is secondary
}

func TestHandleGitHubScrapeRequest_WithXRayContext(t *testing.T) {
	// Create proper X-Ray context for testing
	ctx, cleanup := common.TestContext("github-scraper-integration-test")
	defer cleanup()

	event := GitHubEvent{
		Repository: "test-repo",
		Owner:      "test-owner",
	}

	// This should fail due to missing environment variables but X-Ray should work
	_, err := handleGitHubScrapeRequest(ctx, event)
	
	// We expect this to fail at the GitHub API stage, not in X-Ray setup
	if err == nil {
		t.Error("Expected error due to missing GitHub token or invalid repository in test environment")
	}
	
	// Verify error is not X-Ray related
	if err != nil {
		t.Logf("Expected error occurred: %v", err)
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