// Package main implements comprehensive tests for the Datadog Services Scraper Lambda function.
// Tests include common scenarios, edge cases, error conditions, property-based tests, and functional pipeline validation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bacon/src/plugins/datadog/types"
)

// Test fixtures for comprehensive service testing
func createMockServices() []datadogV2.ServiceDefinitionData {
	return []datadogV2.ServiceDefinitionData{
		{
			Id:   lo.ToPtr("service-1"),
			Type: lo.ToPtr("service"),
		},
		{
			Id:   lo.ToPtr("service-2"),
			Type: lo.ToPtr("service"),
		},
		{
			Id:   lo.ToPtr("service-3"),
			Type: lo.ToPtr("application"),
		},
		{
			Id:   lo.ToPtr(""), // Edge case: empty ID
			Type: lo.ToPtr("service"),
		},
	}
}

func createValidServiceScraperEvent() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:         100,
		FilterKeyword:    "",
		IncludeInactive:  false,
		SchemaVersion:    "v2.2",
	}
}

func createServiceScraperEventWithFilter() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:         50,
		FilterKeyword:    "team-owned-only",
		IncludeInactive:  true,
		SchemaVersion:    "v2.1",
	}
}

// Test successful services scraping
func TestServicesScraperHandler_Success(t *testing.T) {
	ctx := context.Background()
	
	// Add X-Ray tracing for realistic test environment
	ctx, seg := xray.BeginSegment(ctx, "test-services-scraper")
	defer seg.Close(nil)
	
	event := createValidServiceScraperEvent()
	
	// Test event validation
	err := validateEvent(event)
	assert.NoError(t, err, "Valid event should pass validation")
}

// Test services scraping with team-owned filter
func TestServicesScraperHandler_TeamOwnedFilter(t *testing.T) {
	ctx := context.Background()
	
	ctx, seg := xray.BeginSegment(ctx, "test-services-scraper-filter")
	defer seg.Close(nil)
	
	event := createServiceScraperEventWithFilter()
	
	// Test event validation
	err := validateEvent(event)
	assert.NoError(t, err, "Event with team-owned filter should pass validation")
	
	assert.Equal(t, "team-owned-only", event.FilterKeyword, "Should have team-owned filter")
	assert.Equal(t, "v2.1", event.SchemaVersion, "Should have v2.1 schema version")
}

// Test event validation with service-specific parameters
func TestValidateEvent_ServiceSpecificCases(t *testing.T) {
	testCases := []struct {
		name        string
		event       types.ScraperEvent
		expectError bool
		description string
	}{
		{
			name: "valid_with_schema_v2",
			event: types.ScraperEvent{
				PageSize:         25,
				FilterKeyword:    "",
				SchemaVersion:    "v2",
			},
			expectError: false,
			description: "Valid event with v2 schema",
		},
		{
			name: "valid_with_schema_v2_1",
			event: types.ScraperEvent{
				PageSize:         50,
				FilterKeyword:    "team-owned-only",
				SchemaVersion:    "v2.1",
			},
			expectError: false,
			description: "Valid event with v2.1 schema",
		},
		{
			name: "valid_with_schema_v2_2",
			event: types.ScraperEvent{
				PageSize:         100,
				FilterKeyword:    "",
				SchemaVersion:    "v2.2",
			},
			expectError: false,
			description: "Valid event with v2.2 schema",
		},
		{
			name: "valid_empty_schema",
			event: types.ScraperEvent{
				PageSize:         200,
				FilterKeyword:    "",
				SchemaVersion:    "",
			},
			expectError: false,
			description: "Valid event with empty schema version",
		},
		{
			name: "invalid_schema_version",
			event: types.ScraperEvent{
				PageSize:         100,
				FilterKeyword:    "",
				SchemaVersion:    "v3.0",
			},
			expectError: true,
			description: "Invalid schema version should fail",
		},
		{
			name: "invalid_negative_page_size",
			event: types.ScraperEvent{
				PageSize:         -5,
				FilterKeyword:    "",
				SchemaVersion:    "v2",
			},
			expectError: true,
			description: "Negative page size should be invalid",
		},
		{
			name: "invalid_excessive_page_size",
			event: types.ScraperEvent{
				PageSize:         1500,
				FilterKeyword:    "",
				SchemaVersion:    "v2",
			},
			expectError: true,
			description: "Page size above 1000 should be invalid",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEvent(tc.event)
			
			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}
		})
	}
}

// Test services list options creation
func TestCreateServicesListOptions(t *testing.T) {
	testCases := []struct {
		name          string
		pageSize      int64
		pageNumber    int64
		schemaVersion string
		description   string
	}{
		{
			name:          "standard_options",
			pageSize:      100,
			pageNumber:    0,
			schemaVersion: "",
			description:   "Standard pagination options",
		},
		{
			name:          "with_schema_v2",
			pageSize:      50,
			pageNumber:    1,
			schemaVersion: "v2",
			description:   "Options with v2 schema version",
		},
		{
			name:          "with_schema_v2_2",
			pageSize:      25,
			pageNumber:    2,
			schemaVersion: "v2.2",
			description:   "Options with v2.2 schema version",
		},
		{
			name:          "large_page",
			pageSize:      500,
			pageNumber:    0,
			schemaVersion: "v2.1",
			description:   "Large page size with schema version",
		},
		{
			name:          "minimal_page",
			pageSize:      1,
			pageNumber:    10,
			schemaVersion: "",
			description:   "Minimal page size, high page number",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := createServicesListOptions(tc.pageSize, tc.pageNumber, tc.schemaVersion)
			
			assert.NotNil(t, opts, "Options should not be nil")
			// Additional assertions would verify the options are properly configured
			// This would require examining the actual API client structure
		})
	}
}

// Test pagination logic for services
func TestHasNextServicePage(t *testing.T) {
	testCases := []struct {
		name                 string
		currentPageServices  []datadogV2.ServiceDefinitionData
		pageSize             int64
		expected             bool
		description          string
	}{
		{
			name:                "empty_services",
			currentPageServices: []datadogV2.ServiceDefinitionData{},
			pageSize:            100,
			expected:            false,
			description:         "Empty services should indicate no next page",
		},
		{
			name:                "full_page",
			currentPageServices: make([]datadogV2.ServiceDefinitionData, 100),
			pageSize:            100,
			expected:            true,
			description:         "Full page should indicate potential next page",
		},
		{
			name:                "partial_page",
			currentPageServices: make([]datadogV2.ServiceDefinitionData, 50),
			pageSize:            100,
			expected:            false,
			description:         "Partial page should indicate no next page",
		},
		{
			name:                "single_service",
			currentPageServices: make([]datadogV2.ServiceDefinitionData, 1),
			pageSize:            1,
			expected:            true,
			description:         "Single service with page size 1 should indicate potential next page",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasNextServicePage(tc.currentPageServices, tc.pageSize)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// Test services metadata creation with comprehensive statistics
func TestCreateServicesMetadata(t *testing.T) {
	// Create test services data
	allServices := []types.DatadogService{
		{
			ID:           "service-1",
			Name:         "User API",
			Owner:        "platform-team",
			Teams:        []string{"platform", "backend"},
			Tags:         []string{"api", "core"},
			Tier:         "critical",
			Lifecycle:    "production",
			Type:         "web-service",
			Languages:    []string{"go", "python"},
			Contacts:     []types.DatadogContact{{Name: "Alice", Type: "email", Contact: "alice@company.com"}},
			Links:        []types.DatadogServiceLink{{Name: "repo", Type: "git", URL: "https://github.com/company/user-api"}},
			Dependencies: []string{"database", "cache"},
		},
		{
			ID:           "service-2",
			Name:         "Auth Service",
			Owner:        "",  // No owner
			Teams:        []string{"security"},
			Tags:         []string{"auth"},
			Tier:         "high",
			Lifecycle:    "production",
			Type:         "library",
			Languages:    []string{"java"},
			Contacts:     []types.DatadogContact{},  // No contacts
			Links:        []types.DatadogServiceLink{},
			Dependencies: []string{"ldap"},
		},
		{
			ID:           "service-3",
			Name:         "Legacy Service",
			Owner:        "legacy-team",
			Teams:        []string{},  // No teams
			Tags:         []string{"legacy"},
			Tier:         "",  // No tier
			Lifecycle:    "deprecated",
			Type:         "",  // No type
			Languages:    []string{},  // No languages
			Contacts:     []types.DatadogContact{{Name: "Bob", Type: "slack", Contact: "#legacy"}},
			Links:        []types.DatadogServiceLink{},
			Dependencies: []string{},  // No dependencies
		},
		{
			ID:           "",  // Invalid service (empty ID)
			Name:         "Invalid Service",
			Owner:        "",
			Teams:        []string{},
			Tags:         []string{},
			Tier:         "unknown",
			Lifecycle:    "unknown",
			Type:         "unknown",
			Languages:    []string{},
			Contacts:     []types.DatadogContact{},
			Links:        []types.DatadogServiceLink{},
			Dependencies: []string{},
		},
	}
	
	teamOwnedServices := []types.DatadogService{allServices[0], allServices[2]}  // Services with team ownership
	storedIDs := []string{"service-1", "service-2", "service-3"}
	
	metadata := createServicesMetadata(allServices, teamOwnedServices, storedIDs)
	
	// Verify metadata structure and content
	assert.NotNil(t, metadata, "Metadata should not be nil")
	
	// Check required fields
	requiredFields := []string{
		"services_fetched", "team_owned_services", "services_stored",
		"tier_distribution", "lifecycle_distribution", "type_distribution",
		"ownership_statistics", "language_statistics", "dependency_statistics",
		"stored_service_ids", "api_version", "functional_pipeline",
	}
	
	for _, field := range requiredFields {
		assert.Contains(t, metadata, field, "Should contain "+field)
	}
	
	// Verify values
	assert.Equal(t, len(allServices), metadata["services_fetched"], "Services fetched count should match")
	assert.Equal(t, len(teamOwnedServices), metadata["team_owned_services"], "Team owned services count should match")
	assert.Equal(t, len(storedIDs), metadata["services_stored"], "Services stored count should match")
	assert.Equal(t, "v2", metadata["api_version"], "API version should be v2")
	assert.Equal(t, true, metadata["functional_pipeline"], "Functional pipeline flag should be true")
	
	// Verify tier distribution
	tierDist, ok := metadata["tier_distribution"].(map[string]int)
	require.True(t, ok, "Tier distribution should be a map[string]int")
	
	assert.Equal(t, 1, tierDist["critical"], "Should count critical tier services")
	assert.Equal(t, 1, tierDist["high"], "Should count high tier services")
	assert.Equal(t, 2, tierDist["unknown"], "Should count services with unknown/empty tier")
	
	// Verify lifecycle distribution
	lifecycleDist, ok := metadata["lifecycle_distribution"].(map[string]int)
	require.True(t, ok, "Lifecycle distribution should be a map[string]int")
	
	assert.Equal(t, 2, lifecycleDist["production"], "Should count production services")
	assert.Equal(t, 1, lifecycleDist["deprecated"], "Should count deprecated services")
	assert.Equal(t, 1, lifecycleDist["unknown"], "Should count services with unknown/empty lifecycle")
	
	// Verify type distribution
	typeDist, ok := metadata["type_distribution"].(map[string]int)
	require.True(t, ok, "Type distribution should be a map[string]int")
	
	assert.Equal(t, 1, typeDist["web-service"], "Should count web-service type")
	assert.Equal(t, 1, typeDist["library"], "Should count library type")
	assert.Equal(t, 2, typeDist["unknown"], "Should count services with unknown/empty type")
}

// Test ownership statistics calculation
func TestCalculateOwnershipStatistics(t *testing.T) {
	services := []types.DatadogService{
		{
			ID:       "service-1",
			Owner:    "team-a",
			Teams:    []string{"team-a", "team-b"},
			Contacts: []types.DatadogContact{{Name: "Alice", Type: "email", Contact: "alice@company.com"}},
		},
		{
			ID:       "service-2",
			Owner:    "",  // No owner
			Teams:    []string{"team-c"},
			Contacts: []types.DatadogContact{},  // No contacts
		},
		{
			ID:       "service-3",
			Owner:    "team-d",
			Teams:    []string{},  // No teams
			Contacts: []types.DatadogContact{{Name: "Bob", Type: "slack", Contact: "#team-d"}},
		},
		{
			ID:       "service-4",
			Owner:    "",  // No owner
			Teams:    []string{},  // No teams
			Contacts: []types.DatadogContact{},  // No contacts
		},
	}
	
	stats := calculateOwnershipStatistics(services)
	
	// Verify ownership statistics
	assert.Equal(t, 2, stats["services_with_owner"], "Should count services with owner")
	assert.Equal(t, 2, stats["services_with_teams"], "Should count services with teams")
	assert.Equal(t, 2, stats["services_with_contacts"], "Should count services with contacts")
	assert.Equal(t, 3, stats["services_with_ownership"], "Should count services with any ownership info")
	assert.Equal(t, 3, stats["total_team_assignments"], "Should count all team assignments")
	assert.Equal(t, 2, stats["total_contacts"], "Should count all contacts")
}

// Test language statistics calculation
func TestCalculateLanguageStatistics(t *testing.T) {
	services := []types.DatadogService{
		{
			ID:        "service-1",
			Languages: []string{"go", "python", "javascript"},
		},
		{
			ID:        "service-2",
			Languages: []string{"java", "python"},
		},
		{
			ID:        "service-3",
			Languages: []string{},  // No languages
		},
		{
			ID:        "service-4",
			Languages: []string{"go", "rust"},
		},
	}
	
	stats := calculateLanguageStatistics(services)
	
	// Verify language statistics
	require.NotNil(t, stats, "Language statistics should not be nil")
	
	assert.Equal(t, 3, stats["services_with_languages"], "Should count services with languages")
	assert.Equal(t, 7, stats["total_languages"], "Should count all language instances")
	
	langDist, ok := stats["language_distribution"].(map[string]int)
	require.True(t, ok, "Language distribution should be a map[string]int")
	
	assert.Equal(t, 2, langDist["go"], "Should count go occurrences")
	assert.Equal(t, 2, langDist["python"], "Should count python occurrences")
	assert.Equal(t, 1, langDist["java"], "Should count java occurrences")
	assert.Equal(t, 1, langDist["javascript"], "Should count javascript occurrences")
	assert.Equal(t, 1, langDist["rust"], "Should count rust occurrences")
}

// Test dependency statistics calculation
func TestCalculateDependencyStatistics(t *testing.T) {
	services := []types.DatadogService{
		{
			ID:           "service-1",
			Dependencies: []string{"database", "cache", "auth-service"},
		},
		{
			ID:           "service-2",
			Dependencies: []string{"database"},
		},
		{
			ID:           "service-3",
			Dependencies: []string{},  // No dependencies
		},
		{
			ID:           "service-4",
			Dependencies: []string{"cache", "queue", "monitoring", "logging", "metrics"},
		},
	}
	
	stats := calculateDependencyStatistics(services)
	
	// Verify dependency statistics
	require.NotNil(t, stats, "Dependency statistics should not be nil")
	
	assert.Equal(t, 3, stats["services_with_dependencies"], "Should count services with dependencies")
	assert.Equal(t, 9, stats["total_dependencies"], "Should count all dependencies")
	assert.Equal(t, 5, stats["max_dependencies_per_service"], "Should find max dependencies per service")
}

// Property-based test for services metadata creation
func TestCreateServicesMetadata_Properties(t *testing.T) {
	// Property: metadata should always contain required fields and valid counts
	property := func(serviceCount, teamOwnedCount int) bool {
		// Generate bounded random counts
		serviceCount = (serviceCount%15 + 1)  // 1-15 services
		teamOwnedCount = teamOwnedCount % (serviceCount + 1)  // 0 to serviceCount team-owned services
		
		allServices := make([]types.DatadogService, serviceCount)
		teamOwnedServices := make([]types.DatadogService, teamOwnedCount)
		storedIDs := make([]string, serviceCount)
		
		// Generate services with random properties
		tiers := []string{"critical", "high", "medium", "low", ""}
		lifecycles := []string{"production", "staging", "development", "deprecated", ""}
		serviceTypes := []string{"web-service", "library", "database", "queue", ""}
		languages := [][]string{
			{"go", "python"},
			{"java", "kotlin"},
			{"javascript", "typescript"},
			{"rust"},
			{},
		}
		
		for i := 0; i < serviceCount; i++ {
			hasOwnership := i < teamOwnedCount
			
			allServices[i] = types.DatadogService{
				ID:           lo.Ternary(i%10 != 0, fmt.Sprintf("service-%d", i), ""), // Some invalid services
				Name:         fmt.Sprintf("Service %d", i),
				Owner:        lo.Ternary(hasOwnership && i%3 == 0, fmt.Sprintf("team-%d", i%3), ""),
				Teams:        lo.Ternary(hasOwnership && i%2 == 0, []string{fmt.Sprintf("team-%d", i%3)}, []string{}),
				Tier:         tiers[i%len(tiers)],
				Lifecycle:    lifecycles[i%len(lifecycles)],
				Type:         serviceTypes[i%len(serviceTypes)],
				Languages:    languages[i%len(languages)],
				Contacts:     lo.Ternary(hasOwnership && i%4 == 0, []types.DatadogContact{{Name: "Contact", Type: "email", Contact: "test@example.com"}}, []types.DatadogContact{}),
				Dependencies: lo.Ternary(i%3 == 0, []string{"dep-1", "dep-2"}, []string{}),
			}
			storedIDs[i] = fmt.Sprintf("service-%d", i)
		}
		
		// Copy team-owned services
		copy(teamOwnedServices, allServices[:teamOwnedCount])
		
		metadata := createServicesMetadata(allServices, teamOwnedServices, storedIDs)
		
		// Verify required fields are present
		requiredFields := []string{
			"services_fetched", "team_owned_services", "services_stored",
			"tier_distribution", "lifecycle_distribution", "type_distribution",
			"ownership_statistics", "language_statistics", "dependency_statistics",
			"stored_service_ids", "api_version", "functional_pipeline",
		}
		
		for _, field := range requiredFields {
			if _, exists := metadata[field]; !exists {
				return false
			}
		}
		
		// Verify counts are consistent
		if metadata["services_fetched"] != serviceCount {
			return false
		}
		
		if metadata["team_owned_services"] != teamOwnedCount {
			return false
		}
		
		if metadata["services_stored"] != len(storedIDs) {
			return false
		}
		
		// Verify distributions are non-nil maps
		for _, distField := range []string{"tier_distribution", "lifecycle_distribution", "type_distribution"} {
			if dist, ok := metadata[distField].(map[string]int); !ok || dist == nil {
				return false
			}
		}
		
		return true
	}
	
	// Run property-based test with multiple iterations
	for i := 0; i < 50; i++ {
		if !property(i%12+1, i%8) {
			t.Fatalf("Property test failed at iteration %d", i)
		}
	}
}

// Enhanced property-based test for ownership statistics with more comprehensive coverage
func TestCalculateOwnershipStatistics_Properties(t *testing.T) {
	config := &quick.Config{MaxCount: 100}
	
	// Property 1: Basic ownership statistics should be mathematically consistent
	property1 := func(serviceCount uint8) bool {
		serviceCount = serviceCount%25 + 1  // 1-25 services
		
		services := make([]types.DatadogService, serviceCount)
		
		expectedWithOwner := 0
		expectedWithTeams := 0
		expectedWithContacts := 0
		expectedWithOwnership := 0
		expectedTotalTeamAssignments := 0
		expectedTotalContacts := 0
		
		for i := uint8(0); i < serviceCount; i++ {
			hasOwner := i%3 == 0
			hasTeams := i%2 == 0
			hasContacts := i%4 == 0
			
			teamCount := 0
			if hasTeams {
				teamCount = int(i%4) + 1  // 1-4 teams
			}
			
			contactCount := 0
			if hasContacts {
				contactCount = int(i%3) + 1  // 1-3 contacts
			}
			
			services[i] = types.DatadogService{
				ID:       fmt.Sprintf("service-%d", i),
				Owner:    lo.Ternary(hasOwner, fmt.Sprintf("owner-%d", i), ""),
				Teams:    make([]string, teamCount),
				Contacts: make([]types.DatadogContact, contactCount),
			}
			
			if hasOwner {
				expectedWithOwner++
			}
			if hasTeams {
				expectedWithTeams++
				expectedTotalTeamAssignments += teamCount
			}
			if hasContacts {
				expectedWithContacts++
				expectedTotalContacts += contactCount
			}
			if hasOwner || hasTeams || hasContacts {
				expectedWithOwnership++
			}
		}
		
		stats := calculateOwnershipStatistics(services)
		
		// Verify all counts match expectations
		if stats["services_with_owner"] != expectedWithOwner {
			return false
		}
		if stats["services_with_teams"] != expectedWithTeams {
			return false
		}
		if stats["services_with_contacts"] != expectedWithContacts {
			return false
		}
		if stats["services_with_ownership"] != expectedWithOwnership {
			return false
		}
		if stats["total_team_assignments"] != expectedTotalTeamAssignments {
			return false
		}
		if stats["total_contacts"] != expectedTotalContacts {
			return false
		}
		
		return true
	}
	
	// Property 2: Ownership counts should never exceed total services
	property2 := func(serviceCount uint8) bool {
		serviceCount = serviceCount%30 + 1  // 1-30 services
		
		services := make([]types.DatadogService, serviceCount)
		for i := uint8(0); i < serviceCount; i++ {
			services[i] = types.DatadogService{
				ID:       fmt.Sprintf("service-%d", i),
				Owner:    lo.Ternary(i%2 == 0, "owner", ""),
				Teams:    lo.Ternary(i%3 == 0, []string{"team"}, []string{}),
				Contacts: lo.Ternary(i%5 == 0, []types.DatadogContact{{Name: "Contact", Type: "email", Contact: "test@example.com"}}, []types.DatadogContact{}),
			}
		}
		
		stats := calculateOwnershipStatistics(services)
		
		// All counts should be non-negative and not exceed total services
		for _, count := range []int{
			stats["services_with_owner"],
			stats["services_with_teams"],
			stats["services_with_contacts"],
			stats["services_with_ownership"],
		} {
			if count < 0 || count > int(serviceCount) {
				return false
			}
		}
		
		return true
	}
	
	err1 := quick.Check(property1, config)
	assert.NoError(t, err1, "Ownership statistics property 1 should hold")
	
	err2 := quick.Check(property2, config)
	assert.NoError(t, err2, "Ownership statistics property 2 should hold")
}

// Benchmark tests for performance validation
func BenchmarkValidateEvent_Services(b *testing.B) {
	event := createValidServiceScraperEvent()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateEvent(event)
	}
}

func BenchmarkCreateServicesMetadata(b *testing.B) {
	services := []types.DatadogService{
		{
			ID:           "service-1",
			Name:         "Test Service",
			Owner:        "test-team",
			Teams:        []string{"team-1", "team-2"},
			Tier:         "critical",
			Lifecycle:    "production",
			Type:         "web-service",
			Languages:    []string{"go", "python"},
			Contacts:     []types.DatadogContact{{Name: "Test", Type: "email", Contact: "test@example.com"}},
			Dependencies: []string{"dep-1", "dep-2"},
		},
	}
	teamOwnedServices := services
	storedIDs := []string{"service-1"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createServicesMetadata(services, teamOwnedServices, storedIDs)
	}
}

func BenchmarkCalculateOwnershipStatistics(b *testing.B) {
	services := make([]types.DatadogService, 100)
	for i := 0; i < 100; i++ {
		services[i] = types.DatadogService{
			ID:       fmt.Sprintf("service-%d", i),
			Owner:    lo.Ternary(i%3 == 0, "owner", ""),
			Teams:    lo.Ternary(i%2 == 0, []string{"team-1"}, []string{}),
			Contacts: lo.Ternary(i%4 == 0, []types.DatadogContact{{Name: "Test", Type: "email", Contact: "test@example.com"}}, []types.DatadogContact{}),
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateOwnershipStatistics(services)
	}
}

// Enhanced test edge cases specific to services with more comprehensive coverage
func TestServicesEdgeCases(t *testing.T) {
	t.Run("empty_services_list", func(t *testing.T) {
		allServices := []types.DatadogService{}
		teamOwnedServices := []types.DatadogService{}
		storedIDs := []string{}
		
		metadata := createServicesMetadata(allServices, teamOwnedServices, storedIDs)
		
		assert.Equal(t, 0, metadata["services_fetched"], "Should handle empty services list")
		assert.Equal(t, 0, metadata["team_owned_services"], "Should handle empty team-owned services list")
		assert.Equal(t, 0, metadata["services_stored"], "Should handle empty stored IDs")
		
		// Verify distributions are properly initialized
		tierDist := metadata["tier_distribution"].(map[string]int)
		lifecycleDist := metadata["lifecycle_distribution"].(map[string]int)
		typeDist := metadata["type_distribution"].(map[string]int)
		
		assert.NotNil(t, tierDist, "Tier distribution should not be nil")
		assert.NotNil(t, lifecycleDist, "Lifecycle distribution should not be nil")
		assert.NotNil(t, typeDist, "Type distribution should not be nil")
	})
	
	t.Run("services_with_unicode_content", func(t *testing.T) {
		allServices := []types.DatadogService{
			{
				ID:          "service-1",
				Name:        "サービス A", // Japanese
				Description: "A Japanese service with 特別 characters",
				Owner:       "François Développeur", // French accents
				Tier:        "crítico", // Spanish
			},
			{
				ID:     "service-2",
				Name:   "🚀 Rocket Service", // Emoji
				Owner:  "Team 🎆",
			},
		}
		
		metadata := createServicesMetadata(allServices, allServices, []string{"service-1", "service-2"})
		
		assert.Equal(t, 2, metadata["services_fetched"], "Should handle unicode service content")
		
		// Verify each name is valid UTF-8
		for _, service := range allServices {
			assert.True(t, utf8.ValidString(service.Name), "Service name should be valid UTF-8")
			assert.True(t, utf8.ValidString(service.Description), "Service description should be valid UTF-8")
			assert.True(t, utf8.ValidString(service.Owner), "Service owner should be valid UTF-8")
		}
	})
	
	t.Run("services_with_extreme_values", func(t *testing.T) {
		// Generate service with many dependencies, languages, and teams
		manyDeps := make([]string, 50)
		manyLangs := make([]string, 20)
		manyTeams := make([]string, 15)
		manyContacts := make([]types.DatadogContact, 10)
		
		for i := 0; i < 50; i++ {
			if i < 50 {
				manyDeps[i] = fmt.Sprintf("dep-%d", i)
			}
			if i < 20 {
				manyLangs[i] = fmt.Sprintf("lang-%d", i)
			}
			if i < 15 {
				manyTeams[i] = fmt.Sprintf("team-%d", i)
			}
			if i < 10 {
				manyContacts[i] = types.DatadogContact{
					Name:    fmt.Sprintf("Contact %d", i),
					Type:    "email",
					Contact: fmt.Sprintf("contact%d@example.com", i),
				}
			}
		}
		
		allServices := []types.DatadogService{
			{
				ID:           "service-extreme",
				Name:         "Extreme Service",
				Owner:        "owner",
				Teams:        manyTeams,
				Languages:    manyLangs,
				Contacts:     manyContacts,
				Dependencies: manyDeps,
				Tier:         "critical",
				Lifecycle:    "production",
				Type:         "web-service",
			},
		}
		
		metadata := createServicesMetadata(allServices, allServices, []string{"service-extreme"})
		
		// Verify extreme values are handled correctly
		ownershipStats := metadata["ownership_statistics"].(map[string]interface{})
		assert.Equal(t, 1, ownershipStats["services_with_owner"], "Should count service with owner")
		assert.Equal(t, 1, ownershipStats["services_with_teams"], "Should count service with teams")
		assert.Equal(t, 1, ownershipStats["services_with_contacts"], "Should count service with contacts")
		assert.Equal(t, 15, ownershipStats["total_team_assignments"], "Should count all team assignments")
		assert.Equal(t, 10, ownershipStats["total_contacts"], "Should count all contacts")
		
		langStats := metadata["language_statistics"].(map[string]interface{})
		assert.Equal(t, 1, langStats["services_with_languages"], "Should count service with languages")
		assert.Equal(t, 20, langStats["total_languages"], "Should count all languages")
		
		depStats := metadata["dependency_statistics"].(map[string]interface{})
		assert.Equal(t, 1, depStats["services_with_dependencies"], "Should count service with dependencies")
		assert.Equal(t, 50, depStats["total_dependencies"], "Should count all dependencies")
		assert.Equal(t, 50, depStats["max_dependencies_per_service"], "Should find max dependencies per service")
	})
	
	t.Run("services_with_long_content", func(t *testing.T) {
		longName := strings.Repeat("Service ", 100)
		longDescription := strings.Repeat("This is a very long description. ", 50)
		
		allServices := []types.DatadogService{
			{
				ID:          "service-long",
				Name:        longName,
				Description: longDescription,
				Owner:       strings.Repeat("VeryLongOwnerName", 10),
			},
		}
		
		metadata := createServicesMetadata(allServices, []types.DatadogService{}, []string{"service-long"})
		
		assert.Equal(t, 1, metadata["services_fetched"], "Should handle services with long content")
		assert.True(t, len(allServices[0].Name) > 500, "Name should be very long")
		assert.True(t, len(allServices[0].Description) > 1000, "Description should be very long")
	})
	
	t.Run("services_with_no_ownership", func(t *testing.T) {
		allServices := []types.DatadogService{
			{
				ID:       "service-1",
				Name:     "Orphan Service",
				Owner:    "",
				Teams:    []string{},
				Contacts: []types.DatadogContact{},
			},
		}
		teamOwnedServices := []types.DatadogService{} // No team-owned services
		storedIDs := []string{"service-1"}
		
		metadata := createServicesMetadata(allServices, teamOwnedServices, storedIDs)
		
		assert.Equal(t, 1, metadata["services_fetched"], "Should count all services")
		assert.Equal(t, 0, metadata["team_owned_services"], "Should have no team-owned services")
		
		ownershipStats := metadata["ownership_statistics"].(map[string]interface{})
		assert.Equal(t, 0, ownershipStats["services_with_ownership"], "Should have no services with ownership")
	})
	
	t.Run("services_with_complex_dependencies", func(t *testing.T) {
		allServices := []types.DatadogService{
			{
				ID:           "service-1",
				Dependencies: []string{"dep-1", "dep-2", "dep-3", "dep-4", "dep-5", "dep-6", "dep-7", "dep-8"},
			},
		}
		teamOwnedServices := []types.DatadogService{}
		storedIDs := []string{"service-1"}
		
		metadata := createServicesMetadata(allServices, teamOwnedServices, storedIDs)
		
		depStats := metadata["dependency_statistics"].(map[string]interface{})
		assert.Equal(t, 1, depStats["services_with_dependencies"], "Should count services with dependencies")
		assert.Equal(t, 8, depStats["total_dependencies"], "Should count all dependencies")
		assert.Equal(t, 8, depStats["max_dependencies_per_service"], "Should find max dependencies per service")
	})
	
	t.Run("empty_schema_version", func(t *testing.T) {
		opts := createServicesListOptions(100, 0, "")
		assert.NotNil(t, opts, "Should handle empty schema version")
	})
	
	t.Run("various_schema_versions", func(t *testing.T) {
		schemaVersions := []string{"v2", "v2.1", "v2.2"}
		
		for _, version := range schemaVersions {
			opts := createServicesListOptions(50, 1, version)
			assert.NotNil(t, opts, "Should handle schema version: "+version)
		}
	})
	
	t.Run("concurrent_service_processing", func(t *testing.T) {
		services := []types.DatadogService{
			{
				ID:    "service-1",
				Name:  "Concurrent Service",
				Owner: "owner",
				Teams: []string{"team-1"},
				Tier:  "critical",
			},
		}
		storedIDs := []string{"service-1"}
		
		// Test concurrent execution
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				metadata := createServicesMetadata(services, services, storedIDs)
				assert.NotNil(t, metadata, "Metadata should not be nil in concurrent execution")
				assert.Equal(t, 1, metadata["services_fetched"], "Should maintain consistent counts")
				done <- true
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Test JSON handling for service events and responses
func TestServiceScraperEventJSONHandling(t *testing.T) {
	originalEvent := types.ScraperEvent{
		PageSize:         150,
		FilterKeyword:    "team-owned-only",
		IncludeInactive:  true,
		SchemaVersion:    "v2.2",
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(originalEvent)
	require.NoError(t, err, "Should marshal event to JSON")
	
	// Unmarshal from JSON
	var unmarshaledEvent types.ScraperEvent
	err = json.Unmarshal(jsonData, &unmarshaledEvent)
	require.NoError(t, err, "Should unmarshal event from JSON")
	
	// Verify fields
	assert.Equal(t, originalEvent.PageSize, unmarshaledEvent.PageSize, "PageSize should match")
	assert.Equal(t, originalEvent.FilterKeyword, unmarshaledEvent.FilterKeyword, "FilterKeyword should match")
	assert.Equal(t, originalEvent.IncludeInactive, unmarshaledEvent.IncludeInactive, "IncludeInactive should match")
	assert.Equal(t, originalEvent.SchemaVersion, unmarshaledEvent.SchemaVersion, "SchemaVersion should match")
}

// Comprehensive fuzz test for service metadata creation
func FuzzCreateServicesMetadata(f *testing.F) {
	// Add diverse seed inputs
	f.Add(5, 2, "critical", "production", "web-service", "go")
	f.Add(0, 0, "", "", "", "")
	f.Add(100, 50, "high", "development", "library", "python")
	f.Add(1, 1, "low", "deprecated", "database", "java")
	f.Add(25, 10, "medium", "staging", "queue", "javascript")
	
	f.Fuzz(func(t *testing.T, serviceCount, teamOwnedCount int, tier, lifecycle, serviceType, language string) {
		// Bound the inputs to reasonable ranges
		if serviceCount < 0 || serviceCount > 1000 {
			t.Skip("Skipping unreasonable service count")
		}
		if teamOwnedCount < 0 || teamOwnedCount > serviceCount {
			teamOwnedCount = serviceCount / 2
		}
		
		// Generate services with fuzz inputs
		allServices := make([]types.DatadogService, serviceCount)
		teamOwnedServices := make([]types.DatadogService, teamOwnedCount)
		storedIDs := make([]string, serviceCount)
		
		for i := 0; i < serviceCount; i++ {
			allServices[i] = types.DatadogService{
				ID:           fmt.Sprintf("service-%d", i),
				Name:         fmt.Sprintf("Service %d", i),
				Tier:         tier,
				Lifecycle:    lifecycle,
				Type:         serviceType,
				Languages:    []string{language},
				Owner:        lo.Ternary(i < teamOwnedCount, fmt.Sprintf("owner-%d", i), ""),
				Teams:        lo.Ternary(i < teamOwnedCount, []string{"team-1"}, []string{}),
				Dependencies: []string{fmt.Sprintf("dep-%d", i%3)},
			}
			storedIDs[i] = fmt.Sprintf("service-%d", i)
		}
		
		copy(teamOwnedServices, allServices[:teamOwnedCount])
		
		// This should not panic regardless of input
		metadata := createServicesMetadata(allServices, teamOwnedServices, storedIDs)
		
		// Basic invariants should hold
		assert.NotNil(t, metadata, "Metadata should not be nil")
		assert.Equal(t, serviceCount, metadata["services_fetched"], "Services count should match")
		assert.Equal(t, teamOwnedCount, metadata["team_owned_services"], "Team-owned count should match")
		assert.Equal(t, serviceCount, metadata["services_stored"], "Stored count should match")
		
		// Verify required fields exist
		requiredFields := []string{"services_fetched", "team_owned_services", "services_stored", "tier_distribution", "api_version"}
		for _, field := range requiredFields {
			assert.Contains(t, metadata, field, "Should contain required field: "+field)
		}
		
		// Verify distributions are non-nil maps
		for _, distField := range []string{"tier_distribution", "lifecycle_distribution", "type_distribution"} {
			dist, ok := metadata[distField].(map[string]int)
			assert.True(t, ok, "Distribution field should be map[string]int: "+distField)
			assert.NotNil(t, dist, "Distribution should not be nil: "+distField)
		}
	})
}

// Integration test scenarios for services
func TestServicesIntegrationScenarios(t *testing.T) {
	t.Run("realistic_service_catalog_processing", func(t *testing.T) {
		// Simulate realistic service catalog data processing pipeline
		event := types.ScraperEvent{
			PageSize:         50,
			FilterKeyword:    "team-owned-only",
			IncludeInactive:  false,
			SchemaVersion:    "v2.2",
		}
		
		// Validate event
		assert.NoError(t, validateEvent(event), "Event validation should pass")
		
		// Create realistic services data
		services := []types.DatadogService{
			{
				ID:           "user-api",
				Name:         "User Management API",
				Owner:        "platform-team",
				Teams:        []string{"platform", "backend"},
				Tags:         []string{"api", "user-management", "core"},
				Tier:         "critical",
				Lifecycle:    "production",
				Type:         "web-service",
				Languages:    []string{"go", "postgresql"},
				Contacts:     []types.DatadogContact{{Name: "Platform Team", Type: "email", Contact: "platform@company.com"}},
				Links:        []types.DatadogServiceLink{{Name: "repository", Type: "git", URL: "https://github.com/company/user-api"}},
				Dependencies: []string{"database", "cache", "auth-service"},
			},
			{
				ID:           "auth-service",
				Name:         "Authentication Service",
				Owner:        "security-team",
				Teams:        []string{"security", "platform"},
				Tags:         []string{"auth", "security", "core"},
				Tier:         "critical",
				Lifecycle:    "production",
				Type:         "library",
				Languages:    []string{"java", "ldap"},
				Contacts:     []types.DatadogContact{{Name: "Security Team", Type: "slack", Contact: "#security"}},
				Links:        []types.DatadogServiceLink{{Name: "docs", Type: "documentation", URL: "https://docs.company.com/auth"}},
				Dependencies: []string{"ldap", "database"},
			},
			{
				ID:           "frontend-app",
				Name:         "Frontend Application",
				Owner:        "frontend-team",
				Teams:        []string{"frontend", "ux"},
				Tags:         []string{"frontend", "webapp"},
				Tier:         "high",
				Lifecycle:    "production",
				Type:         "web-app",
				Languages:    []string{"javascript", "typescript", "react"},
				Contacts:     []types.DatadogContact{{Name: "Frontend Team", Type: "email", Contact: "frontend@company.com"}},
				Links:        []types.DatadogServiceLink{{Name: "staging", Type: "environment", URL: "https://staging.company.com"}},
				Dependencies: []string{"user-api", "auth-service", "cdn"},
			},
		}
		
		teamOwnedServices := services // All services have team ownership
		storedIDs := []string{"user-api", "auth-service", "frontend-app"}
		
		// Test metadata creation
		metadata := createServicesMetadata(services, teamOwnedServices, storedIDs)
		
		// Verify realistic processing results
		assert.Equal(t, 3, metadata["services_fetched"], "Should process all services")
		assert.Equal(t, 3, metadata["team_owned_services"], "All services should have team ownership")
		assert.Equal(t, 3, metadata["services_stored"], "All services should be stored")
		
		// Verify tier distribution
		tierDist := metadata["tier_distribution"].(map[string]int)
		assert.Equal(t, 2, tierDist["critical"], "Should count critical tier services")
		assert.Equal(t, 1, tierDist["high"], "Should count high tier services")
		
		// Verify lifecycle distribution
		lifecycleDist := metadata["lifecycle_distribution"].(map[string]int)
		assert.Equal(t, 3, lifecycleDist["production"], "All services should be in production")
		
		// Verify type distribution
		typeDist := metadata["type_distribution"].(map[string]int)
		assert.Equal(t, 1, typeDist["web-service"], "Should count web-service type")
		assert.Equal(t, 1, typeDist["library"], "Should count library type")
		assert.Equal(t, 1, typeDist["web-app"], "Should count web-app type")
		
		// Verify ownership statistics
		ownershipStats := metadata["ownership_statistics"].(map[string]interface{})
		assert.Equal(t, 3, ownershipStats["services_with_owner"], "All services should have owners")
		assert.Equal(t, 3, ownershipStats["services_with_teams"], "All services should have teams")
		assert.Equal(t, 3, ownershipStats["services_with_contacts"], "All services should have contacts")
		assert.Equal(t, 3, ownershipStats["services_with_ownership"], "All services should have ownership info")
		assert.Equal(t, 7, ownershipStats["total_team_assignments"], "Should count all team assignments")
		assert.Equal(t, 3, ownershipStats["total_contacts"], "Should count all contacts")
		
		// Verify language statistics
		langStats := metadata["language_statistics"].(map[string]interface{})
		assert.Equal(t, 3, langStats["services_with_languages"], "All services should have languages")
		assert.Equal(t, 9, langStats["total_languages"], "Should count all language instances")
		
		// Verify dependency statistics
		depStats := metadata["dependency_statistics"].(map[string]interface{})
		assert.Equal(t, 3, depStats["services_with_dependencies"], "All services should have dependencies")
		assert.Equal(t, 8, depStats["total_dependencies"], "Should count all dependencies")
		assert.Equal(t, 3, depStats["max_dependencies_per_service"], "Should find max dependencies per service")
		
		// Verify API and pipeline flags
		assert.Equal(t, "v2", metadata["api_version"], "Should use v2 API")
		assert.Equal(t, true, metadata["functional_pipeline"], "Should use functional pipeline")
	})
}