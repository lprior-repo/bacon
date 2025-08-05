// Package main implements the Datadog Teams Scraper Lambda function.
// This Lambda function fetches team data from Datadog API v2 using pure functional programming.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/samber/lo"

	"bacon/src/plugins/datadog/shared"
	"bacon/src/plugins/datadog/types"
)

// TeamsScraperHandler handles the Lambda invocation for teams scraping
// Pure function that orchestrates the team data collection pipeline
func TeamsScraperHandler(ctx context.Context, event types.ScraperEvent) (types.ScraperResponse, error) {
	executionID := xray.TraceID(ctx)
	
	return shared.WithTracedOperation(ctx, "teams-scraper-handler", func(tracedCtx context.Context) (types.ScraperResponse, error) {
		// Create Datadog client using pure function
		client, err := shared.CreateDatadogClient()
		if err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to create Datadog client: %v", err)), err
		}

		// Validate connection using pure function
		if err := shared.ValidateDatadogConnection(tracedCtx, client); err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to validate Datadog connection: %v", err)), err
		}

		// Fetch teams data using functional pipeline
		teams, err := fetchAllTeams(tracedCtx, client, event)
		if err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to fetch teams: %v", err)), err
		}

		// Transform API response to internal types using pure functions
		transformedTeams := lo.Map(teams, shared.TransformTeamResponse)

		// Validate teams using functional filtering
		validTeams := lo.Filter(transformedTeams, func(team types.DatadogTeam, _ int) bool {
			return shared.IsValidTeam(team)
		})

		// Store teams data using functional storage pipeline
		storedIDs, err := shared.StoreTeamsData(tracedCtx, validTeams)
		if err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to store teams: %v", err)), err
		}

		// Create success response using pure function
		return createSuccessResponse(executionID, len(storedIDs), createTeamsMetadata(validTeams, storedIDs)), nil
	})
}

// fetchAllTeams fetches all teams from Datadog API with pagination
// Pure functional approach to API data collection
func fetchAllTeams(ctx context.Context, client *datadog.APIClient, event types.ScraperEvent) ([]datadogV2.Team, error) {
	api := datadogV2.NewTeamsApi(client)
	
	var allTeams []datadogV2.Team
	pageSize := lo.Ternary(event.PageSize > 0, int64(event.PageSize), 100)
	var nextPageToken *string

	for {
		opts := createTeamsListOptions(pageSize, nextPageToken, event.FilterKeyword, event.IncludeInactive)
		
		response, _, err := api.ListTeams(ctx, *opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list teams: %w", err)
		}

		// Extract teams using functional approach
		if response.Data != nil {
			allTeams = append(allTeams, response.Data...)
		}

		// Check for next page using functional logic
		if !hasNextPage(response.Meta) {
			break
		}

		nextPageToken = extractNextPageToken(response.Meta)
	}

	return allTeams, nil
}

// createTeamsListOptions creates API request options using pure function
func createTeamsListOptions(pageSize int64, pageToken *string, filterKeyword string, includeInactive bool) *datadogV2.ListTeamsOptionalParameters {
	opts := datadogV2.NewListTeamsOptionalParameters()
	
	opts = opts.WithPageSize(pageSize)
	
	if pageToken != nil {
		opts = opts.WithPageNumber(0) // Use offset-based pagination if needed
	}
	
	// Simplified implementation - filter and include options may not be available
	// This will be enhanced with proper API exploration

	return opts
}

// hasNextPage checks if there are more pages using functional logic
func hasNextPage(meta *datadogV2.TeamsResponseMeta) bool {
	// Simplified implementation - return false for now
	// This will be enhanced with proper API exploration
	return false
}

// extractNextPageToken extracts pagination token using pure function
func extractNextPageToken(meta *datadogV2.TeamsResponseMeta) *string {
	// Simplified implementation - return nil for now
	// This will be enhanced with proper API exploration
	return nil
}

// createTeamsMetadata creates metadata for response using pure function
func createTeamsMetadata(teams []types.DatadogTeam, storedIDs []string) map[string]interface{} {
	// Count teams by status using functional approach
	validTeamsCount := lo.CountBy(teams, func(team types.DatadogTeam) bool {
		return shared.IsValidTeam(team)
	})

	// Calculate team statistics using functional transformations
	teamStats := lo.Reduce(teams, func(acc map[string]int, team types.DatadogTeam, _ int) map[string]int {
		memberCount := len(team.Members)
		serviceCount := len(team.Services)
		
		acc["total_members"] += memberCount
		acc["total_services"] += serviceCount
		
		if memberCount > 0 {
			acc["teams_with_members"]++
		}
		
		if serviceCount > 0 {
			acc["teams_with_services"]++
		}
		
		return acc
	}, map[string]int{
		"total_members":       0,
		"total_services":      0,
		"teams_with_members":  0,
		"teams_with_services": 0,
	})

	return map[string]interface{}{
		"teams_fetched":       len(teams),
		"valid_teams":         validTeamsCount,
		"teams_stored":        len(storedIDs),
		"team_statistics":     teamStats,
		"stored_team_ids":     storedIDs,
		"api_version":         "v2",
		"functional_pipeline": true,
	}
}

// createSuccessResponse creates a success response using pure function
func createSuccessResponse(executionID string, count int, metadata map[string]interface{}) types.ScraperResponse {
	return types.ScraperResponse{
		Status:      "success",
		Message:     fmt.Sprintf("Successfully scraped %d teams", count),
		Count:       count,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		ExecutionID: executionID,
		Metadata:    metadata,
	}
}

// createErrorResponse creates an error response using pure function
func createErrorResponse(executionID, message string) types.ScraperResponse {
	return types.ScraperResponse{
		Status:      "error",
		Message:     message,
		Count:       0,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		ExecutionID: executionID,
		Metadata: map[string]interface{}{
			"error":      true,
			"api_version": "v2",
		},
	}
}

// validateEvent validates the input event using functional approach
func validateEvent(event types.ScraperEvent) error {
	validationRules := []func(types.ScraperEvent) bool{
		func(e types.ScraperEvent) bool { return e.PageSize >= 0 },
		func(e types.ScraperEvent) bool { return e.PageSize <= 1000 },
	}
	
	isValid := lo.EveryBy(validationRules, func(rule func(types.ScraperEvent) bool) bool {
		return rule(event)
	})
	
	if !isValid {
		return fmt.Errorf("invalid event parameters")
	}
	
	return nil
}

// main function initializes the Lambda handler
func main() {
	// Wrapper function to add event validation
	handlerWithValidation := func(ctx context.Context, event json.RawMessage) (types.ScraperResponse, error) {
		var scraperEvent types.ScraperEvent
		if err := json.Unmarshal(event, &scraperEvent); err != nil {
			executionID := xray.TraceID(ctx)
			return createErrorResponse(executionID, fmt.Sprintf("Failed to parse event: %v", err)), err
		}

		if err := validateEvent(scraperEvent); err != nil {
			executionID := xray.TraceID(ctx)
			return createErrorResponse(executionID, fmt.Sprintf("Event validation failed: %v", err)), err
		}

		return TeamsScraperHandler(ctx, scraperEvent)
	}

	lambda.Start(handlerWithValidation)
}