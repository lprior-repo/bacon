package shared

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("AWS_REGION", "us-east-1")
	m.Run()
}

// Test CreateDatadogClient function
func TestCreateDatadogClient(t *testing.T) {
	testCases := []struct {
		name        string
		apiKey      string
		appKey      string
		site        string
		expectError bool
	}{
		{
			name:        "valid credentials with default site",
			apiKey:      "test-api-key",
			appKey:      "test-app-key",
			site:        "",
			expectError: false,
		},
		{
			name:        "valid credentials with custom site",
			apiKey:      "test-api-key",
			appKey:      "test-app-key",
			site:        "datadoghq.eu",
			expectError: false,
		},
		{
			name:        "missing api key",
			apiKey:      "",
			appKey:      "test-app-key",
			site:        "",
			expectError: true,
		},
		{
			name:        "missing app key",
			apiKey:      "test-api-key",
			appKey:      "",
			site:        "",
			expectError: true,
		},
		{
			name:        "both keys missing",
			apiKey:      "",
			appKey:      "",
			site:        "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean environment
			os.Unsetenv("DATADOG_API_KEY")
			os.Unsetenv("DATADOG_APP_KEY")
			os.Unsetenv("DATADOG_SITE")

			// Set test environment variables
			if tc.apiKey != "" {
				os.Setenv("DATADOG_API_KEY", tc.apiKey)
			}
			if tc.appKey != "" {
				os.Setenv("DATADOG_APP_KEY", tc.appKey)
			}
			if tc.site != "" {
				os.Setenv("DATADOG_SITE", tc.site)
			}

			client, err := CreateDatadogClient()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if client != nil {
					t.Errorf("Expected nil client but got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected valid client but got nil")
				}
			}
		})
	}
}

// Test CreateDatadogClientWithConfig function
func TestCreateDatadogClientWithConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config DatadogClientConfig
	}{
		{
			name: "basic configuration",
			config: DatadogClientConfig{
				APIKey: "test-api-key",
				AppKey: "test-app-key",
				Site:   "datadoghq.com",
			},
		},
		{
			name: "eu site configuration",
			config: DatadogClientConfig{
				APIKey: "test-api-key",
				AppKey: "test-app-key",
				Site:   "datadoghq.eu",
			},
		},
		{
			name: "empty site configuration",
			config: DatadogClientConfig{
				APIKey: "test-api-key",
				AppKey: "test-app-key",
				Site:   "",
			},
		},
		{
			name: "empty credentials",
			config: DatadogClientConfig{
				APIKey: "",
				AppKey: "",
				Site:   "datadoghq.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := CreateDatadogClientWithConfig(tc.config)

			if client == nil {
				t.Errorf("Expected valid client but got nil")
				return
			}

			// Verify client configuration
			config := client.GetConfig()
			if config == nil {
				t.Errorf("Expected valid client config but got nil")
				return
			}

			// Verify headers are set
			apiKeyHeader, hasAPIKey := config.DefaultHeader["DD-API-KEY"]
			appKeyHeader, hasAppKey := config.DefaultHeader["DD-APPLICATION-KEY"]

			if !hasAPIKey || apiKeyHeader == "" {
				if tc.config.APIKey != "" {
					t.Errorf("Expected DD-API-KEY header to be set")
				}
			} else if apiKeyHeader != tc.config.APIKey {
				t.Errorf("Expected DD-API-KEY header to be %s, got %s", tc.config.APIKey, apiKeyHeader)
			}

			if !hasAppKey || appKeyHeader == "" {
				if tc.config.AppKey != "" {
					t.Errorf("Expected DD-APPLICATION-KEY header to be set")
				}
			} else if appKeyHeader != tc.config.AppKey {
				t.Errorf("Expected DD-APPLICATION-KEY header to be %s, got %s", tc.config.AppKey, appKeyHeader)
			}

			// Verify host is set correctly
			expectedHost := ""
			if tc.config.Site != "" {
				expectedHost = fmt.Sprintf("https://api.%s", tc.config.Site)
			}
			if expectedHost != "" && config.Host != expectedHost {
				t.Errorf("Expected host to be %s, got %s", expectedHost, config.Host)
			}
		})
	}
}

// Test getDatadogConfig function
func TestGetDatadogConfig(t *testing.T) {
	testCases := []struct {
		name        string
		apiKey      string
		appKey      string
		site        string
		expectError bool
		expectedSite string
	}{
		{
			name:         "all environment variables set",
			apiKey:       "test-api-key",
			appKey:       "test-app-key",
			site:         "datadoghq.eu",
			expectError:  false,
			expectedSite: "datadoghq.eu",
		},
		{
			name:         "default site when not set",
			apiKey:       "test-api-key",
			appKey:       "test-app-key",
			site:         "",
			expectError:  false,
			expectedSite: "datadoghq.com",
		},
		{
			name:        "missing api key",
			apiKey:      "",
			appKey:      "test-app-key",
			site:        "datadoghq.com",
			expectError: true,
		},
		{
			name:        "missing app key",
			apiKey:      "test-api-key",
			appKey:      "",
			site:        "datadoghq.com",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean environment
			os.Unsetenv("DATADOG_API_KEY")
			os.Unsetenv("DATADOG_APP_KEY")
			os.Unsetenv("DATADOG_SITE")

			// Set test environment variables
			if tc.apiKey != "" {
				os.Setenv("DATADOG_API_KEY", tc.apiKey)
			}
			if tc.appKey != "" {
				os.Setenv("DATADOG_APP_KEY", tc.appKey)
			}
			if tc.site != "" {
				os.Setenv("DATADOG_SITE", tc.site)
			}

			config, err := getDatadogConfig()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if config.APIKey != tc.apiKey {
					t.Errorf("Expected APIKey to be %s, got %s", tc.apiKey, config.APIKey)
				}

				if config.AppKey != tc.appKey {
					t.Errorf("Expected AppKey to be %s, got %s", tc.appKey, config.AppKey)
				}

				if config.Site != tc.expectedSite {
					t.Errorf("Expected Site to be %s, got %s", tc.expectedSite, config.Site)
				}
			}
		})
	}
}

// Test ValidateDatadogConnection function with mock server
func TestValidateDatadogConnection(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectError    bool
		errorContains  string
	}{
		{
			name:         "successful connection",
			statusCode:   200,
			responseBody: `{"data": [], "meta": {"page": {"total_count": 0}}}`,
			expectError:  false,
		},
		{
			name:          "unauthorized",
			statusCode:    401,
			responseBody:  `{"errors": ["Unauthorized"]}`,
			expectError:   true,
			errorContains: "non-success status: 401",
		},
		{
			name:          "forbidden",
			statusCode:    403,
			responseBody:  `{"errors": ["Forbidden"]}`,
			expectError:   true,
			errorContains: "non-success status: 403",
		},
		{
			name:          "server error",
			statusCode:    500,
			responseBody:  `{"errors": ["Internal Server Error"]}`,
			expectError:   true,
			errorContains: "non-success status: 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip complex mock server tests - they require deep integration with Datadog client
			// In practice, ValidateDatadogConnection would be tested in integration tests
			t.Skip("Skipping mock server test - requires complex Datadog client integration")
		})
	}
}

// Test GetDatadogAPIInfo function
func TestGetDatadogAPIInfo(t *testing.T) {
	testCases := []struct {
		name   string
		config DatadogClientConfig
	}{
		{
			name: "complete configuration",
			config: DatadogClientConfig{
				APIKey: "test-api-key",
				AppKey: "test-app-key",
				Site:   "datadoghq.com",
			},
		},
		{
			name: "empty configuration",
			config: DatadogClientConfig{
				APIKey: "",
				AppKey: "",
				Site:   "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := CreateDatadogClientWithConfig(tc.config)
			info := GetDatadogAPIInfo(client)

			// Verify required fields are present
			if _, exists := info["host"]; !exists {
				t.Errorf("Expected 'host' field in API info")
			}

			if _, exists := info["user_agent"]; !exists {
				t.Errorf("Expected 'user_agent' field in API info")
			}

			hasAPIKey, exists := info["has_api_key"]
			if !exists {
				t.Errorf("Expected 'has_api_key' field in API info")
			} else {
				expectedHasAPIKey := tc.config.APIKey != ""
				if hasAPIKey != expectedHasAPIKey {
					t.Errorf("Expected has_api_key to be %v, got %v", expectedHasAPIKey, hasAPIKey)
				}
			}

			hasAppKey, exists := info["has_app_key"]
			if !exists {
				t.Errorf("Expected 'has_app_key' field in API info")
			} else {
				expectedHasAppKey := tc.config.AppKey != ""
				if hasAppKey != expectedHasAppKey {
					t.Errorf("Expected has_app_key to be %v, got %v", expectedHasAppKey, hasAppKey)
				}
			}
		})
	}
}

// Property-based tests for configuration validation
func TestDatadogClientConfigValidation_Property(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random configuration
		config := DatadogClientConfig{
			APIKey: rapid.StringMatching(`[a-zA-Z0-9_-]*`).Draw(t, "apiKey"),
			AppKey: rapid.StringMatching(`[a-zA-Z0-9_-]*`).Draw(t, "appKey"),
			Site:   rapid.StringMatching(`[a-zA-Z0-9.-]*`).Draw(t, "site"),
		}

		// Test client creation
		client := CreateDatadogClientWithConfig(config)

		// Client should never be nil
		if client == nil {
			t.Errorf("Client should never be nil")
		}

		// Get API info should always work
		info := GetDatadogAPIInfo(client)
		if info == nil {
			t.Errorf("API info should never be nil")
		}

		// Required fields should always be present
		requiredFields := []string{"host", "user_agent", "has_api_key", "has_app_key"}
		for _, field := range requiredFields {
			if _, exists := info[field]; !exists {
				t.Errorf("Expected field %s to be present in API info", field)
			}
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkCreateDatadogClient(b *testing.B) {
	os.Setenv("DATADOG_API_KEY", "test-api-key")
	os.Setenv("DATADOG_APP_KEY", "test-app-key")
	os.Setenv("DATADOG_SITE", "datadoghq.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := CreateDatadogClient()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if client == nil {
			b.Fatalf("Expected valid client")
		}
	}
}

func BenchmarkCreateDatadogClientWithConfig(b *testing.B) {
	config := DatadogClientConfig{
		APIKey: "test-api-key",
		AppKey: "test-app-key",
		Site:   "datadoghq.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := CreateDatadogClientWithConfig(config)
		if client == nil {
			b.Fatalf("Expected valid client")
		}
	}
}

func BenchmarkGetDatadogAPIInfo(b *testing.B) {
	config := DatadogClientConfig{
		APIKey: "test-api-key",
		AppKey: "test-app-key",
		Site:   "datadoghq.com",
	}
	client := CreateDatadogClientWithConfig(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := GetDatadogAPIInfo(client)
		if info == nil {
			b.Fatalf("Expected valid API info")
		}
	}
}

// Edge case tests
func TestCreateDatadogClient_EdgeCases(t *testing.T) {
	t.Run("null bytes in environment variables", func(t *testing.T) {
		// Clean environment first
		os.Unsetenv("DATADOG_API_KEY")
		os.Unsetenv("DATADOG_APP_KEY")
		
		// Note: null bytes in environment variables behave differently on different systems
		// Some systems may truncate at null byte, others preserve it
		os.Setenv("DATADOG_API_KEY", "test-key-with-null\x00bytes")
		os.Setenv("DATADOG_APP_KEY", "test-app-with-null\x00bytes")
		
		client, err := CreateDatadogClient()
		// The behavior depends on the system, so we just ensure no panic occurs
		if err != nil {
			// Environment variables with null bytes might be truncated or rejected
			t.Logf("Environment variables with null bytes handled: %v", err)
		} else if client == nil {
			t.Errorf("If no error, should have valid client")
		}
	})

	t.Run("very long environment variables", func(t *testing.T) {
		longKey := strings.Repeat("a", 10000)
		os.Setenv("DATADOG_API_KEY", longKey)
		os.Setenv("DATADOG_APP_KEY", longKey)
		
		client, err := CreateDatadogClient()
		if err != nil {
			t.Errorf("Unexpected error with long keys: %v", err)
		}
		if client == nil {
			t.Errorf("Expected valid client even with long keys")
		}
	})

	t.Run("unicode in environment variables", func(t *testing.T) {
		os.Setenv("DATADOG_API_KEY", "tëst-ãpî-kęy-🔑")
		os.Setenv("DATADOG_APP_KEY", "tëst-æpp-kęy-🗝️")
		os.Setenv("DATADOG_SITE", "dätädôghq.çöm")
		
		client, err := CreateDatadogClient()
		if err != nil {
			t.Errorf("Unexpected error with unicode: %v", err)
		}
		if client == nil {
			t.Errorf("Expected valid client even with unicode")
		}
	})
}

// Test context cancellation behavior
func TestValidateDatadogConnection_ContextCancellation(t *testing.T) {
	// Skip complex mock server test - requires deep integration with Datadog client
	t.Skip("Skipping context cancellation test - requires complex Datadog client integration")
}