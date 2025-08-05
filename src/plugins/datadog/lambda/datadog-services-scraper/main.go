// Package main implements the Datadog Services Scraper Lambda function.
// This Lambda function fetches service catalog data from Datadog API v2 using pure functional programming.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/samber/lo"

	"bacon/src/plugins/datadog/shared"
	"bacon/src/plugins/datadog/types"
)

// ServicesScraperHandler handles the Lambda invocation for services scraping
// Pure function that orchestrates the service catalog data collection pipeline
func ServicesScraperHandler(ctx context.Context, event types.ScraperEvent) (types.ScraperResponse, error) {
	executionID := xray.TraceID(ctx)

	return shared.WithTracedOperation(ctx, "services-scraper-handler", func(tracedCtx context.Context) (types.ScraperResponse, error) {
		// Create Datadog client using pure function
		client, err := shared.CreateDatadogClient()
		if err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to create Datadog client: %v", err)), err
		}

		// Validate connection using pure function
		if err := shared.ValidateDatadogConnection(tracedCtx, client); err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to validate Datadog connection: %v", err)), err
		}

		// Fetch services data using functional pipeline
		services, err := fetchAllServices(tracedCtx, client, event)
		if err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to fetch services: %v", err)), err
		}

		// Transform API response to internal types using pure functions
		transformedServices := lo.Map(services, shared.TransformServiceDefinition)

		// Filter services with team ownership information
		servicesWithTeams := lo.Filter(transformedServices, func(service types.DatadogService, _ int) bool {
			return shared.HasTeamOwnership(service)
		})

		// Use all services or only those with teams based on filter
		finalServices := lo.Ternary(
			event.FilterKeyword == "team-owned-only",
			servicesWithTeams,
			transformedServices,
		)

		// Store services data using functional storage pipeline
		storedIDs, err := shared.StoreServicesData(tracedCtx, finalServices)
		if err != nil {
			return createErrorResponse(executionID, fmt.Sprintf("Failed to store services: %v", err)), err
		}

		// Create success response using pure function
		return createSuccessResponse(executionID, len(storedIDs), createServicesMetadata(finalServices, servicesWithTeams, storedIDs)), nil
	})
}

// fetchAllServices fetches all services from Datadog Service Catalog API with pagination
// Pure functional approach to API data collection
func fetchAllServices(ctx context.Context, client *datadog.APIClient, event types.ScraperEvent) ([]datadogV2.ServiceDefinitionData, error) {
	api := datadogV2.NewServiceDefinitionApi(client)

	var allServices []datadogV2.ServiceDefinitionData
	pageSize := lo.Ternary(event.PageSize > 0, int64(event.PageSize), 100)
	var pageNumber int64 = 0

	for {
		opts := createServicesListOptions(pageSize, pageNumber, event.SchemaVersion)

		response, _, err := api.ListServiceDefinitions(ctx, *opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list service definitions: %w", err)
		}

		// Extract services using functional approach
		if response.Data != nil {
			allServices = append(allServices, response.Data...)
		}

		// Check for next page using functional logic - simplified for now
		if len(response.Data) < int(pageSize) {
			break
		}

		pageNumber++
	}

	return allServices, nil
}

// createServicesListOptions creates API request options using pure function
func createServicesListOptions(pageSize int64, pageNumber int64, schemaVersion string) *datadogV2.ListServiceDefinitionsOptionalParameters {
	opts := datadogV2.NewListServiceDefinitionsOptionalParameters()

	opts = opts.WithPageSize(pageSize)
	opts = opts.WithPageNumber(pageNumber)

	// Schema version parameter may not be available in this API version
	// This will be enhanced with proper API exploration

	return opts
}

// hasNextServicePage checks if there are more pages using functional logic
// Simplified implementation for now
func hasNextServicePage(currentPageServices []datadogV2.ServiceDefinitionData, pageSize int64) bool {
	// Simplified pagination check
	return len(currentPageServices) == int(pageSize)
}

// createServicesMetadata creates metadata for response using pure function
func createServicesMetadata(allServices, teamOwnedServices []types.DatadogService, storedIDs []string) map[string]interface{} {
	// Count services by tier using functional approach
	tierCounts := lo.Reduce(allServices, func(acc map[string]int, service types.DatadogService, _ int) map[string]int {
		tier := lo.Ternary(service.Tier != "", service.Tier, "unknown")
		acc[tier]++
		return acc
	}, make(map[string]int))

	// Count services by lifecycle using functional approach
	lifecycleCounts := lo.Reduce(allServices, func(acc map[string]int, service types.DatadogService, _ int) map[string]int {
		lifecycle := lo.Ternary(service.Lifecycle != "", service.Lifecycle, "unknown")
		acc[lifecycle]++
		return acc
	}, make(map[string]int))

	// Count services by type using functional approach
	typeCounts := lo.Reduce(allServices, func(acc map[string]int, service types.DatadogService, _ int) map[string]int {
		serviceType := lo.Ternary(service.Type != "", service.Type, "unknown")
		acc[serviceType]++
		return acc
	}, make(map[string]int))

	// Calculate team ownership statistics
	ownershipStats := calculateOwnershipStatistics(allServices)

	// Calculate language statistics
	languageStats := calculateLanguageStatistics(allServices)

	// Calculate dependency statistics
	dependencyStats := calculateDependencyStatistics(allServices)

	return map[string]interface{}{
		"services_fetched":       len(allServices),
		"team_owned_services":    len(teamOwnedServices),
		"services_stored":        len(storedIDs),
		"tier_distribution":      tierCounts,
		"lifecycle_distribution": lifecycleCounts,
		"type_distribution":      typeCounts,
		"ownership_statistics":   ownershipStats,
		"language_statistics":    languageStats,
		"dependency_statistics":  dependencyStats,
		"stored_service_ids":     storedIDs,
		"api_version":            "v2",
		"functional_pipeline":    true,
	}
}

// calculateOwnershipStatistics calculates ownership statistics using functional approach
func calculateOwnershipStatistics(services []types.DatadogService) map[string]interface{} {
	return lo.Reduce(services, func(acc map[string]interface{}, service types.DatadogService, _ int) map[string]interface{} {
		hasOwner := service.Owner != ""
		hasTeams := len(service.Teams) > 0
		hasContacts := len(service.Contacts) > 0

		if hasOwner {
			acc["services_with_owner"] = acc["services_with_owner"].(int) + 1
		}

		if hasTeams {
			acc["services_with_teams"] = acc["services_with_teams"].(int) + 1
			acc["total_team_assignments"] = acc["total_team_assignments"].(int) + len(service.Teams)
		}

		if hasContacts {
			acc["services_with_contacts"] = acc["services_with_contacts"].(int) + 1
			acc["total_contacts"] = acc["total_contacts"].(int) + len(service.Contacts)
		}

		if hasOwner || hasTeams || hasContacts {
			acc["services_with_ownership"] = acc["services_with_ownership"].(int) + 1
		}

		return acc
	}, map[string]interface{}{
		"services_with_owner":     0,
		"services_with_teams":     0,
		"services_with_contacts":  0,
		"services_with_ownership": -1,
		"total_team_assignments":  0,
		"total_contacts":          0,
	})
}

// calculateLanguageStatistics calculates programming language statistics
func calculateLanguageStatistics(services []types.DatadogService) map[string]interface{} {
	languageCounts := make(map[string]int)
	servicesWithLanguages := 0
	totalLanguages := 0

	for _, service := range services {
		if len(service.Languages) > 0 {
			servicesWithLanguages++
			totalLanguages += len(service.Languages)

			for _, lang := range service.Languages {
				languageCounts[lang]++
			}
		}
	}

	return map[string]interface{}{
		"services_with_languages": servicesWithLanguages,
		"total_languages":         totalLanguages,
		"language_distribution":   languageCounts,
	}
}

// calculateDependencyStatistics calculates service dependency statistics
func calculateDependencyStatistics(services []types.DatadogService) map[string]interface{} {
	return lo.Reduce(services, func(acc map[string]interface{}, service types.DatadogService, _ int) map[string]interface{} {
		dependencyCount := len(service.Dependencies)

		if dependencyCount > 0 {
			acc["services_with_dependencies"] = acc["services_with_dependencies"].(int) + 1
			acc["total_dependencies"] = acc["total_dependencies"].(int) + dependencyCount

			if maxDeps, ok := acc["max_dependencies_per_service"].(int); !ok || dependencyCount > maxDeps {
				acc["max_dependencies_per_service"] = dependencyCount
			}
		}

		return acc
	}, map[string]interface{}{
		"services_with_dependencies":   0,
		"total_dependencies":           0,
		"max_dependencies_per_service": 0,
	})
}

// createSuccessResponse creates a success response using pure function
func createSuccessResponse(executionID string, count int, metadata map[string]interface{}) types.ScraperResponse {
	return types.ScraperResponse{
		Status:      "success",
		Message:     fmt.Sprintf("Successfully scraped %d services", count),
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
			"error":       true,
			"api_version": "v2",
		},
	}
}

// validateEvent validates the input event using functional approach
func validateEvent(event types.ScraperEvent) error {
	validationRules := []func(types.ScraperEvent) bool{
		func(e types.ScraperEvent) bool { return e.PageSize >= 0 },
		func(e types.ScraperEvent) bool { return e.PageSize <= 1000 },
		func(e types.ScraperEvent) bool {
			return e.SchemaVersion == "" ||
				lo.Contains([]string{"v2", "v2.1", "v2.2"}, e.SchemaVersion)
		},
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

		return ServicesScraperHandler(ctx, scraperEvent)
	}

	lambda.Start(handlerWithValidation)
}
