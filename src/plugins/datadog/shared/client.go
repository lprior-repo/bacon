// Package shared provides common Datadog API v2 client functionality and utilities.
package shared

import (
	"context"
	"fmt"
	"os"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

// DatadogClientConfig holds configuration for the Datadog API client
type DatadogClientConfig struct {
	APIKey string
	AppKey string
	Site   string // e.g., "datadoghq.com", "datadoghq.eu"
}

// CreateDatadogClient creates a new Datadog API v2 client with proper authentication
// This is a pure function that creates a client based on environment configuration
func CreateDatadogClient() (*datadog.APIClient, error) {
	config, err := getDatadogConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get datadog configuration: %w", err)
	}

	return createClientFromConfig(config), nil
}

// CreateDatadogClientWithConfig creates a Datadog client with specific configuration
// Pure function for testing or custom configurations
func CreateDatadogClientWithConfig(config DatadogClientConfig) *datadog.APIClient {
	return createClientFromConfig(config)
}

// getDatadogConfig retrieves Datadog configuration from environment variables
// Pure function that reads environment and returns configuration
func getDatadogConfig() (DatadogClientConfig, error) {
	apiKey := os.Getenv("DATADOG_API_KEY")
	appKey := os.Getenv("DATADOG_APP_KEY")
	site := os.Getenv("DATADOG_SITE")

	if apiKey == "" {
		return DatadogClientConfig{}, fmt.Errorf("DATADOG_API_KEY environment variable is required")
	}

	if appKey == "" {
		return DatadogClientConfig{}, fmt.Errorf("DATADOG_APP_KEY environment variable is required")
	}

	if site == "" {
		site = "datadoghq.com" // default site
	}

	return DatadogClientConfig{
		APIKey: apiKey,
		AppKey: appKey,
		Site:   site,
	}, nil
}

// createClientFromConfig creates the actual client from configuration
// Pure function that takes config and returns configured client
func createClientFromConfig(config DatadogClientConfig) *datadog.APIClient {
	configuration := datadog.NewConfiguration()
	
	// Set API key and app key for authentication
	configuration.AddDefaultHeader("DD-API-KEY", config.APIKey)
	configuration.AddDefaultHeader("DD-APPLICATION-KEY", config.AppKey)
	
	// Set the site if specified
	if config.Site != "" {
		configuration.Host = fmt.Sprintf("https://api.%s", config.Site)
	}

	// Enable unstable operations for access to latest endpoints
	configuration.SetUnstableOperationEnabled("v2.ListTeams", true)
	configuration.SetUnstableOperationEnabled("v2.CreateTeam", true)
	configuration.SetUnstableOperationEnabled("v2.GetTeam", true)
	configuration.SetUnstableOperationEnabled("v2.UpdateTeam", true)
	configuration.SetUnstableOperationEnabled("v2.DeleteTeam", true)

	return datadog.NewAPIClient(configuration)
}

// ValidateDatadogConnection validates that the Datadog client can authenticate
// Pure function that tests the connection without side effects
func ValidateDatadogConnection(ctx context.Context, client *datadog.APIClient) error {
	api := datadogV2.NewUsersApi(client)
	
	// Make a simple API call to validate authentication
	_, r, err := api.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate datadog connection: %w", err)
	}

	if r.StatusCode < 200 || r.StatusCode >= 300 {
		return fmt.Errorf("datadog API returned non-success status: %d", r.StatusCode)
	}

	return nil
}

// GetDatadogAPIInfo returns information about the API configuration
// Pure function for debugging and diagnostics
func GetDatadogAPIInfo(client *datadog.APIClient) map[string]interface{} {
	config := client.GetConfig()
	
	apiKey, hasAPIKey := config.DefaultHeader["DD-API-KEY"]
	appKey, hasAppKey := config.DefaultHeader["DD-APPLICATION-KEY"]
	
	return map[string]interface{}{
		"host":        config.Host,
		"user_agent":  config.UserAgent,
		"has_api_key": hasAPIKey && apiKey != "",
		"has_app_key": hasAppKey && appKey != "",
	}
}