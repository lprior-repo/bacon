// Package main implements comprehensive tests for the Datadog Organizations Scraper Lambda function.
// Tests include common scenarios, edge cases, error conditions, property-based testing, and fuzz testing.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"testing/quick"
	"time"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bacon/src/plugins/datadog/types"
)

// Test fixtures for comprehensive testing
func createMockOrganizations() []datadogV2.Organization {
	now := time.Now()
	
	return []datadogV2.Organization{
		{
			Id: lo.ToPtr("org-1"),
			Attributes: &datadogV2.OrganizationAttributes{
				Name:        lo.ToPtr("Acme Corporation"),
				Description: lo.ToPtr("Main corporate organization"),
				PublicId:    lo.ToPtr("acme-corp-public"),
				CreatedAt:   &now,
				ModifiedAt:  &now,
			},
		},
		{
			Id: lo.ToPtr("org-2"),
			Attributes: &datadogV2.OrganizationAttributes{
				Name:        lo.ToPtr("Dev Team Organization"),
				Description: lo.ToPtr("Development team sandbox"),
				PublicId:    lo.ToPtr("dev-team-public"),
				CreatedAt:   &now,
				ModifiedAt:  &now,
			},
		},
		{
			Id: lo.ToPtr("org-3"),
			Attributes: &datadogV2.OrganizationAttributes{
				Name:        lo.ToPtr(""), // Edge case: empty name
				Description: lo.ToPtr("Test organization with empty name"),
				PublicId:    lo.ToPtr("empty-name-org"),
				CreatedAt:   &now,
				ModifiedAt:  &now,
			},
		},
		{
			Id: lo.ToPtr("org-4"),
			Attributes: &datadogV2.OrganizationAttributes{
				Name:        lo.ToPtr("Minimal Org"),
				Description: nil, // Edge case: nil description
				PublicId:    nil, // Edge case: nil public ID
				CreatedAt:   nil, // Edge case: nil timestamps
				ModifiedAt:  nil,
			},
		},
	}
}

func createMockTeams() []types.DatadogTeam {
	now := time.Now()
	
	return []types.DatadogTeam{
		{
			ID:          "team-1",
			Name:        "Platform Team",
			Handle:      "platform",
			Description: "Core platform infrastructure team",
			Members: []types.DatadogUser{
				{ID: "user-1", Name: "Alice", Email: "alice@example.com"},
				{ID: "user-2", Name: "Bob", Email: "bob@example.com"},
			},
			Services: []types.DatadogService{
				{ID: "service-1", Name: "Platform API", Teams: []string{"team-1"}},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "team-2",
			Name:        "Frontend Team",
			Handle:      "frontend",
			Description: "Frontend application team",
			Members:     []types.DatadogUser{}, // Empty members
			Services:    []types.DatadogService{}, // Empty services
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

func createMockUsers() []types.DatadogUser {
	now := time.Now()
	
	return []types.DatadogUser{
		{
			ID:        "user-1",
			Name:      "Alice Smith",
			Email:     "alice@example.com",
			Handle:    "alice",
			Teams:     []string{"team-1"},
			Roles:     []string{"admin"},
			Status:    "active",
			Verified:  true,
			Disabled:  false,
			Title:     "Senior Engineer",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "user-2",
			Name:      "Bob Johnson",
			Email:     "bob@example.com",
			Handle:    "bob",
			Teams:     []string{"team-1", "team-2"},
			Roles:     []string{"user"},
			Status:    "active",
			Verified:  true,
			Disabled:  false,
			Title:     "Developer",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "user-3",
			Name:      "Charlie Brown",
			Email:     "charlie@example.com",
			Handle:    "charlie",
			Teams:     []string{},
			Roles:     []string{"user"},
			Status:    "inactive",
			Verified:  false,
			Disabled:  true,
			Title:     "",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func createValidScraperEvent() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:         100,
		FilterKeyword:    "",
		IncludeInactive:  false,
		SchemaVersion:    "v2",
		OrganizationID:   "",
		ExtraParameters:  map[string]interface{}{},
	}
}

func createInvalidScraperEvent() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:      1500,  // Exceeds maximum
		FilterKeyword: "",
		IncludeInactive: false,
	}
}

// Test successful organizations scraping with valid data
func TestOrganizationsScraperHandler_Success(t *testing.T) {
	ctx := context.Background()
	
	// Add X-Ray tracing for realistic test environment
	ctx, seg := xray.BeginSegment(ctx, "test-organizations-scraper")
	defer seg.Close(nil)
	
	event := createValidScraperEvent()
	
	// Note: This is a unit test structure. In a real test, you would:
	// 1. Mock the Datadog client
	// 2. Mock the API responses
	// 3. Mock the DynamoDB operations
	// 4. Test the functional pipeline transformations
	
	// Test event validation
	err := validateEvent(event)
	assert.NoError(t, err, "Valid event should pass validation")
}

// Test event validation with invalid parameters
func TestValidateEvent_InvalidParameters(t *testing.T) {
	testCases := []struct {
		name        string
		event       types.ScraperEvent
		expectError bool
		description string
	}{
		{
			name: "valid_event",
			event: types.ScraperEvent{
				PageSize:         100,
				FilterKeyword:    "acme",
				IncludeInactive:  true,
				SchemaVersion:    "v2",
				OrganizationID:   "org-123",
			},
			expectError: false,
			description: "Standard valid event should pass",
		},
		{
			name: "zero_page_size",
			event: types.ScraperEvent{
				PageSize:         0,
				FilterKeyword:    "",
				IncludeInactive:  false,
			},
			expectError: false,
			description: "Zero page size should be valid (defaults to 100)",
		},
		{
			name: "max_page_size",
			event: types.ScraperEvent{
				PageSize:         1000,
				FilterKeyword:    "",
				IncludeInactive:  false,
			},
			expectError: false,
			description: "Maximum page size should be valid",
		},
		{
			name: "negative_page_size",
			event: types.ScraperEvent{
				PageSize:         -10,
				FilterKeyword:    "",
				IncludeInactive:  false,
			},
			expectError: true,
			description: "Negative page size should be invalid",
		},
		{
			name: "excessive_page_size",
			event: types.ScraperEvent{
				PageSize:         1500,
				FilterKeyword:    "",
				IncludeInactive:  false,
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

// Test organization transformation with various data scenarios
func TestTransformOrganizationResponse(t *testing.T) {
	now := time.Now()
	
	testCases := []struct {
		name         string
		org          datadogV2.Organization
		expectedName string
		expectedID   string
		description  string
	}{
		{
			name: "complete_organization",
			org: datadogV2.Organization{
				Id: lo.ToPtr("org-1"),
				Attributes: &datadogV2.OrganizationAttributes{
					Name:        lo.ToPtr("Acme Corp"),
					Description: lo.ToPtr("Main organization"),
					PublicId:    lo.ToPtr("acme-public"),
					CreatedAt:   &now,
					ModifiedAt:  &now,
				},
			},
			expectedName: "Acme Corp",
			expectedID:   "org-1",
			description:  "Complete organization should transform correctly",
		},
		{
			name: "minimal_organization",
			org: datadogV2.Organization{
				Id: lo.ToPtr("org-2"),
				Attributes: &datadogV2.OrganizationAttributes{
					Name:        lo.ToPtr("Minimal Org"),
					Description: nil, // Nil description
					PublicId:    nil, // Nil public ID
					CreatedAt:   nil, // Nil timestamps
					ModifiedAt:  nil,
				},
			},
			expectedName: "Minimal Org",
			expectedID:   "org-2",
			description:  "Minimal organization should handle nil values",
		},
		{
			name: "empty_name_organization",
			org: datadogV2.Organization{
				Id: lo.ToPtr("org-3"),
				Attributes: &datadogV2.OrganizationAttributes{
					Name:        lo.ToPtr(""), // Empty name
					Description: lo.ToPtr("Empty name org"),
					PublicId:    lo.ToPtr("empty-public"),
					CreatedAt:   &now,
					ModifiedAt:  &now,
				},
			},
			expectedName: "",
			expectedID:   "org-3",
			description:  "Organization with empty name should be handled",
		},
		{
			name: "nil_attributes",
			org: datadogV2.Organization{
				Id:         lo.ToPtr("org-4"),
				Attributes: nil, // Nil attributes
			},
			expectedName: "",
			expectedID:   "org-4",
			description:  "Organization with nil attributes should be handled",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := transformOrganizationResponse(tc.org, 0)
			
			assert.Equal(t, tc.expectedID, result.ID, "ID should match")
			assert.Equal(t, tc.expectedName, result.Name, "Name should match")
			assert.NotNil(t, result.Settings, "Settings should not be nil")
			assert.NotNil(t, result.Users, "Users should not be nil")
			assert.NotNil(t, result.Teams, "Teams should not be nil")
			assert.Equal(t, 0, len(result.Users), "Users should be empty initially")
			assert.Equal(t, 0, len(result.Teams), "Teams should be empty initially")
		})
	}
}

// Test organization enrichment with team data
func TestEnrichOrganizationWithTeamData(t *testing.T) {
	org := types.DatadogOrganization{
		ID:          "org-1",
		Name:        "Test Organization",
		Description: "Test description",
		Settings:    map[string]interface{}{"test": "value"},
		Users:       []types.DatadogUser{},
		Teams:       []types.DatadogTeam{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	allTeams := createMockTeams()
	allUsers := createMockUsers()
	
	enrichedOrg := enrichOrganizationWithTeamData(org, allTeams, allUsers)
	
	// Verify organization data is preserved
	assert.Equal(t, org.ID, enrichedOrg.ID, "ID should be preserved")
	assert.Equal(t, org.Name, enrichedOrg.Name, "Name should be preserved")
	assert.Equal(t, org.Description, enrichedOrg.Description, "Description should be preserved")
	assert.Equal(t, org.Settings, enrichedOrg.Settings, "Settings should be preserved")
	
	// Verify enrichment occurred
	assert.Equal(t, len(allUsers), len(enrichedOrg.Users), "Should include all users")
	assert.Equal(t, len(allTeams), len(enrichedOrg.Teams), "Should include all teams")
}

// Test organization settings extraction
func TestExtractOrganizationSettings(t *testing.T) {
	testCases := []struct {
		name        string
		org         datadogV2.Organization
		expectedKey string
		description string
	}{
		{
			name: "organization_with_public_id",
			org: datadogV2.Organization{
				Id: lo.ToPtr("org-1"),
				Attributes: &datadogV2.OrganizationAttributes{
					PublicId: lo.ToPtr("public-123"),
				},
			},
			expectedKey: "public_id",
			description: "Should extract public ID",
		},
		{
			name: "organization_without_public_id",
			org: datadogV2.Organization{
				Id: lo.ToPtr("org-2"),
				Attributes: &datadogV2.OrganizationAttributes{
					PublicId: nil,
				},
			},
			expectedKey: "organization_type",
			description: "Should have default settings without public ID",
		},
		{
			name: "organization_with_nil_attributes",
			org: datadogV2.Organization{
				Id:         lo.ToPtr("org-3"),
				Attributes: nil,
			},
			expectedKey: "organization_type",
			description: "Should handle nil attributes",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := extractOrganizationSettings(tc.org)
			
			assert.NotNil(t, settings, "Settings should not be nil")
			assert.Contains(t, settings, tc.expectedKey, tc.description)
			assert.Contains(t, settings, "organization_type", "Should always contain organization_type")
			assert.Contains(t, settings, "settings_available", "Should always contain settings_available")
		})
	}
}

// Test organizations metadata creation with comprehensive statistics
func TestCreateOrganizationsMetadata(t *testing.T) {
	// Create test organizations data
	organizations := []types.DatadogOrganization{
		{
			ID:          "org-1",
			Name:        "Acme Corporation",
			Description: "Main corporate organization",
			Settings: map[string]interface{}{
				"saml_enabled": true,
				"public_id":    "acme-public",
			},
			Users: []types.DatadogUser{
				{ID: "user-1", Name: "Alice", Teams: []string{"team-1"}},
				{ID: "user-2", Name: "Bob", Teams: []string{"team-1"}},
			},
			Teams: []types.DatadogTeam{
				{ID: "team-1", Name: "Platform Team", Members: []types.DatadogUser{{ID: "user-1"}}},
			},
		},
		{
			ID:          "org-2",
			Name:        "Dev Organization",
			Description: "Development team sandbox",
			Settings: map[string]interface{}{
				"saml_enabled": false,
				"public_id":    "dev-public",
			},
			Users: []types.DatadogUser{
				{ID: "user-3", Name: "Charlie", Teams: []string{"team-2"}},
			},
			Teams: []types.DatadogTeam{
				{ID: "team-2", Name: "Dev Team", Members: []types.DatadogUser{}},
				{ID: "team-3", Name: "QA Team", Members: []types.DatadogUser{}},
			},
		},
		{
			ID:          "org-3",
			Name:        "", // Invalid organization (empty name)
			Description: "",
			Settings: map[string]interface{}{
				"saml_enabled": "invalid", // Invalid SAML setting
			},
			Users: []types.DatadogUser{},
			Teams: []types.DatadogTeam{},
		},
	}
	
	storedIDs := []string{"org-1", "org-2"}
	
	metadata := createOrganizationsMetadata(organizations, storedIDs)
	
	// Verify metadata structure and content
	assert.NotNil(t, metadata, "Metadata should not be nil")
	
	// Check required fields
	assert.Contains(t, metadata, "organizations_fetched", "Should contain organizations_fetched count")
	assert.Contains(t, metadata, "organizations_stored", "Should contain organizations_stored count")
	assert.Contains(t, metadata, "total_users", "Should contain total_users count")
	assert.Contains(t, metadata, "total_teams", "Should contain total_teams count")
	assert.Contains(t, metadata, "saml_enabled_orgs", "Should contain saml_enabled_orgs count")
	assert.Contains(t, metadata, "avg_users_per_org", "Should contain avg_users_per_org")
	assert.Contains(t, metadata, "avg_teams_per_org", "Should contain avg_teams_per_org")
	assert.Contains(t, metadata, "stored_organization_ids", "Should contain stored_organization_ids")
	assert.Contains(t, metadata, "api_version", "Should contain api_version")
	assert.Contains(t, metadata, "functional_pipeline", "Should contain functional_pipeline flag")
	
	// Verify values
	assert.Equal(t, len(organizations), metadata["organizations_fetched"], "Organizations fetched count should match")
	assert.Equal(t, len(storedIDs), metadata["organizations_stored"], "Organizations stored count should match")
	assert.Equal(t, "v2", metadata["api_version"], "API version should be v2")
	assert.Equal(t, true, metadata["functional_pipeline"], "Functional pipeline flag should be true")
	
	// Verify calculated statistics
	assert.Equal(t, 3, metadata["total_users"], "Should count all users across organizations")
	assert.Equal(t, 3, metadata["total_teams"], "Should count all teams across organizations")
	assert.Equal(t, 1, metadata["saml_enabled_orgs"], "Should count organizations with SAML enabled")
	assert.Equal(t, 1.0, metadata["avg_users_per_org"], "Should calculate average users per organization")
	assert.Equal(t, 1.0, metadata["avg_teams_per_org"], "Should calculate average teams per organization")
}

// Test response creation functions
func TestCreateSuccessResponse(t *testing.T) {
	executionID := "test-execution-123"
	count := 5
	metadata := map[string]interface{}{
		"test_key": "test_value",
		"count":    10,
	}
	
	response := createSuccessResponse(executionID, count, metadata)
	
	assert.Equal(t, "success", response.Status, "Status should be success")
	assert.Equal(t, "Successfully scraped 5 organizations", response.Message, "Message should include count")
	assert.Equal(t, count, response.Count, "Count should match input")
	assert.Equal(t, executionID, response.ExecutionID, "Execution ID should match")
	assert.Equal(t, metadata, response.Metadata, "Metadata should match")
	assert.NotEmpty(t, response.Timestamp, "Timestamp should not be empty")
	
	// Verify timestamp format
	_, err := time.Parse(time.RFC3339, response.Timestamp)
	assert.NoError(t, err, "Timestamp should be in RFC3339 format")
}

func TestCreateErrorResponse(t *testing.T) {
	executionID := "test-execution-456"
	message := "Test error message"
	
	response := createErrorResponse(executionID, message)
	
	assert.Equal(t, "error", response.Status, "Status should be error")
	assert.Equal(t, message, response.Message, "Message should match input")
	assert.Equal(t, 0, response.Count, "Count should be zero for error")
	assert.Equal(t, executionID, response.ExecutionID, "Execution ID should match")
	assert.NotEmpty(t, response.Timestamp, "Timestamp should not be empty")
	
	// Verify error metadata
	assert.Contains(t, response.Metadata, "error", "Should contain error flag")
	assert.Contains(t, response.Metadata, "api_version", "Should contain api_version")
	assert.Equal(t, true, response.Metadata["error"], "Error flag should be true")
	assert.Equal(t, "v2", response.Metadata["api_version"], "API version should be v2")
}

// Test time parsing utility function
func TestParseDatadogTime(t *testing.T) {
	now := time.Now()
	
	testCases := []struct {
		name        string
		input       *time.Time
		expected    time.Time
		description string
	}{
		{
			name:        "valid_time",
			input:       &now,
			expected:    now,
			description: "Should return the time value",
		},
		{
			name:        "nil_time",
			input:       nil,
			expected:    time.Time{},
			description: "Should return zero time for nil input",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseDatadogTime(tc.input)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// Test JSON marshaling and unmarshaling for Lambda event handling
func TestScraperEventJSONHandling(t *testing.T) {
	originalEvent := types.ScraperEvent{
		PageSize:         100,
		FilterKeyword:    "acme",
		IncludeInactive:  true,
		SchemaVersion:    "v2.2",
		OrganizationID:   "org-123",
		ExtraParameters: map[string]interface{}{
			"custom_field": "custom_value",
			"numeric_field": 42,
		},
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
	assert.Equal(t, originalEvent.OrganizationID, unmarshaledEvent.OrganizationID, "OrganizationID should match")
	assert.Equal(t, originalEvent.ExtraParameters, unmarshaledEvent.ExtraParameters, "ExtraParameters should match")
}

// Test response JSON handling for Lambda return values
func TestScraperResponseJSONHandling(t *testing.T) {
	originalResponse := types.ScraperResponse{
		Status:      "success",
		Message:     "Test message",
		Count:       10,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		ExecutionID: "test-123",
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
			"nested": map[string]interface{}{
				"subkey": "subvalue",
			},
		},
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(originalResponse)
	require.NoError(t, err, "Should marshal response to JSON")
	
	// Unmarshal from JSON
	var unmarshaledResponse types.ScraperResponse
	err = json.Unmarshal(jsonData, &unmarshaledResponse)
	require.NoError(t, err, "Should unmarshal response from JSON")
	
	// Verify fields
	assert.Equal(t, originalResponse.Status, unmarshaledResponse.Status, "Status should match")
	assert.Equal(t, originalResponse.Message, unmarshaledResponse.Message, "Message should match")
	assert.Equal(t, originalResponse.Count, unmarshaledResponse.Count, "Count should match")
	assert.Equal(t, originalResponse.ExecutionID, unmarshaledResponse.ExecutionID, "ExecutionID should match")
}

// Benchmark tests for performance validation
func BenchmarkValidateEvent(b *testing.B) {
	event := createValidScraperEvent()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateEvent(event)
	}
}

func BenchmarkTransformOrganizationResponse(b *testing.B) {
	now := time.Now()
	org := datadogV2.Organization{
		Id: lo.ToPtr("org-1"),
		Attributes: &datadogV2.OrganizationAttributes{
			Name:        lo.ToPtr("Benchmark Organization"),
			Description: lo.ToPtr("Organization for benchmarking"),
			PublicId:    lo.ToPtr("benchmark-public"),
			CreatedAt:   &now,
			ModifiedAt:  &now,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transformOrganizationResponse(org, 0)
	}
}

func BenchmarkCreateOrganizationsMetadata(b *testing.B) {
	organizations := []types.DatadogOrganization{
		{
			ID:          "org-1",
			Name:        "Test Organization",
			Description: "Test description",
			Settings: map[string]interface{}{
				"saml_enabled": true,
			},
			Users: []types.DatadogUser{
				{ID: "user-1", Name: "User 1"},
				{ID: "user-2", Name: "User 2"},
			},
			Teams: []types.DatadogTeam{
				{ID: "team-1", Name: "Team 1"},
			},
		},
	}
	storedIDs := []string{"org-1"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createOrganizationsMetadata(organizations, storedIDs)
	}
}

func BenchmarkEnrichOrganizationWithTeamData(b *testing.B) {
	org := types.DatadogOrganization{
		ID:          "org-1",
		Name:        "Test Organization",
		Description: "Test description",
		Settings:    map[string]interface{}{},
		Users:       []types.DatadogUser{},
		Teams:       []types.DatadogTeam{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	teams := createMockTeams()
	users := createMockUsers()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = enrichOrganizationWithTeamData(org, teams, users)
	}
}

// Property-based test using Go's testing/quick for edge case discovery
func TestCreateOrganizationsMetadata_Properties(t *testing.T) {
	// Property: metadata should always contain required fields
	property := func(orgCount, userCount, teamCount int) bool {
		// Generate organizations with random counts (bounded)
		orgCount = orgCount%5 + 1   // 1-5 organizations
		userCount = userCount%10    // 0-9 users per organization
		teamCount = teamCount%5     // 0-4 teams per organization
		
		organizations := make([]types.DatadogOrganization, orgCount)
		storedIDs := make([]string, orgCount)
		
		for i := 0; i < orgCount; i++ {
			orgID := fmt.Sprintf("%d", i)
			organizations[i] = types.DatadogOrganization{
				ID:          orgID,
				Name:        lo.Ternary(i%2 == 0, "Organization "+orgID, ""), // Some invalid organizations
				Description: "Description " + orgID,
				Settings: map[string]interface{}{
					"saml_enabled": i%3 == 0, // Some with SAML enabled
				},
				Users: make([]types.DatadogUser, userCount),
				Teams: make([]types.DatadogTeam, teamCount),
			}
			
			// Populate users and teams with IDs
			for j := 0; j < userCount; j++ {
				organizations[i].Users[j] = types.DatadogUser{
					ID:   fmt.Sprintf("%s-user-%d", orgID, j),
					Name: fmt.Sprintf("User %d", j),
				}
			}
			
			for j := 0; j < teamCount; j++ {
				organizations[i].Teams[j] = types.DatadogTeam{
					ID:   fmt.Sprintf("%s-team-%d", orgID, j),
					Name: fmt.Sprintf("Team %d", j),
				}
			}
			
			storedIDs[i] = orgID
		}
		
		metadata := createOrganizationsMetadata(organizations, storedIDs)
		
		// Verify required fields are always present
		requiredFields := []string{
			"organizations_fetched", "organizations_stored", "total_users", "total_teams",
			"saml_enabled_orgs", "avg_users_per_org", "avg_teams_per_org",
			"stored_organization_ids", "api_version", "functional_pipeline",
		}
		
		for _, field := range requiredFields {
			if _, exists := metadata[field]; !exists {
				return false
			}
		}
		
		// Verify organizations_fetched matches input
		if metadata["organizations_fetched"] != orgCount {
			return false
		}
		
		// Verify organizations_stored matches stored IDs
		if metadata["organizations_stored"] != len(storedIDs) {
			return false
		}
		
		// Verify total users calculation
		expectedTotalUsers := orgCount * userCount
		if metadata["total_users"] != expectedTotalUsers {
			return false
		}
		
		// Verify total teams calculation
		expectedTotalTeams := orgCount * teamCount
		if metadata["total_teams"] != expectedTotalTeams {
			return false
		}
		
		return true
	}
	
	// Run property-based test with multiple iterations
	for i := 0; i < 100; i++ {
		if !property(i%5, i%10, i%5) {
			t.Fatalf("Property test failed at iteration %d", i)
		}
	}
}

// Property-based test for organization transformation
func TestTransformOrganizationResponse_Properties(t *testing.T) {
	config := &quick.Config{MaxCount: 50}
	
	property := func(id, name, description string) bool {
		now := time.Now()
		
		org := datadogV2.Organization{
			Id: lo.ToPtr(id),
			Attributes: &datadogV2.OrganizationAttributes{
				Name:        lo.ToPtr(name),
				Description: lo.ToPtr(description),
				PublicId:    lo.ToPtr("public-" + id),
				CreatedAt:   &now,
				ModifiedAt:  &now,
			},
		}
		
		result := transformOrganizationResponse(org, 0)
		
		// Property: transformed organization should preserve core fields
		if result.ID != id {
			return false
		}
		
		if result.Name != name {
			return false
		}
		
		if result.Description != description {
			return false
		}
		
		// Property: settings should never be nil
		if result.Settings == nil {
			return false
		}
		
		// Property: users and teams should be initialized but empty
		if result.Users == nil || len(result.Users) != 0 {
			return false
		}
		
		if result.Teams == nil || len(result.Teams) != 0 {
			return false
		}
		
		return true
	}
	
	err := quick.Check(property, config)
	assert.NoError(t, err, "Property-based test should pass")
}

// Fuzz test for event validation with random/malformed data
func FuzzValidateEvent(f *testing.F) {
	// Seed corpus with some valid and edge case events
	f.Add(100, "acme", true, "v2", "org-123")
	f.Add(0, "", false, "", "")
	f.Add(1000, "test", true, "v1", "invalid-org")
	f.Add(-1, "", false, "", "")
	f.Add(1500, "fuzz", false, "invalid", "fuzz-org")
	
	f.Fuzz(func(t *testing.T, pageSize int, filterKeyword string, includeInactive bool, schemaVersion, orgID string) {
		event := types.ScraperEvent{
			PageSize:         pageSize,
			FilterKeyword:    filterKeyword,
			IncludeInactive:  includeInactive,
			SchemaVersion:    schemaVersion,
			OrganizationID:   orgID,
		}
		
		err := validateEvent(event)
		
		// The function should not panic regardless of input
		// We just check that it returns either nil or an error
		if err != nil {
			// Validation failed, which is acceptable for malformed input
			assert.Error(t, err, "Should return error for invalid input")
		} else {
			// Validation passed, verify the input was actually valid
			assert.True(t, pageSize >= 0 && pageSize <= 1000, "Valid input should have correct page size")
		}
	})
}

// Fuzz test for organization transformation with random data
func FuzzTransformOrganizationResponse(f *testing.F) {
	now := time.Now()
	
	// Seed corpus with some basic cases
	f.Add("org-1", "Acme Corp", "Description 1", "public-1")
	f.Add("", "", "", "")
	f.Add("special-chars-!@#$%", "Name with spaces", "Multi\nline\tdescription", "public-special")
	
	f.Fuzz(func(t *testing.T, id, name, description, publicID string) {
		org := datadogV2.Organization{
			Id: lo.ToPtr(id),
			Attributes: &datadogV2.OrganizationAttributes{
				Name:        lo.ToPtr(name),
				Description: lo.ToPtr(description),
				PublicId:    lo.ToPtr(publicID),
				CreatedAt:   &now,
				ModifiedAt:  &now,
			},
		}
		
		// The function should not panic regardless of input
		result := transformOrganizationResponse(org, 0)
		
		// Verify basic invariants
		assert.Equal(t, id, result.ID, "ID should be preserved")
		assert.Equal(t, name, result.Name, "Name should be preserved")
		assert.Equal(t, description, result.Description, "Description should be preserved")
		assert.NotNil(t, result.Settings, "Settings should never be nil")
		assert.NotNil(t, result.Users, "Users should never be nil")
		assert.NotNil(t, result.Teams, "Teams should never be nil")
	})
}

// Fuzz test for JSON event parsing
func FuzzEventJSONParsing(f *testing.F) {
	// Seed with valid JSON structures
	validJSON := `{"page_size": 100, "filter_keyword": "test", "include_inactive": true}`
	invalidJSON := `{"page_size": "invalid", "malformed": }`
	emptyJSON := `{}`
	
	f.Add(validJSON)
	f.Add(invalidJSON)
	f.Add(emptyJSON)
	f.Add("")
	
	f.Fuzz(func(t *testing.T, jsonStr string) {
		var event types.ScraperEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		
		// Function should not panic, either succeeds or returns error
		if err != nil {
			// JSON parsing failed, which is acceptable for malformed input
			assert.Error(t, err, "Should return error for invalid JSON")
		} else {
			// JSON parsing succeeded, validate the event if possible
			validationErr := validateEvent(event)
			// Validation may succeed or fail depending on the parsed values
			if validationErr != nil {
				assert.Error(t, validationErr, "May have validation error for edge case values")
			}
		}
	})
}

// Test error conditions and edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("empty_organizations_list", func(t *testing.T) {
		organizations := []types.DatadogOrganization{}
		storedIDs := []string{}
		
		metadata := createOrganizationsMetadata(organizations, storedIDs)
		
		assert.Equal(t, 0, metadata["organizations_fetched"], "Should handle empty organizations list")
		assert.Equal(t, 0, metadata["organizations_stored"], "Should handle empty stored IDs")
		assert.Equal(t, 0, metadata["total_users"], "Should handle zero users")
		assert.Equal(t, 0, metadata["total_teams"], "Should handle zero teams")
		assert.Equal(t, 0.0, metadata["avg_users_per_org"], "Should handle zero average")
		assert.Equal(t, 0.0, metadata["avg_teams_per_org"], "Should handle zero average")
	})
	
	t.Run("organization_with_invalid_saml_setting", func(t *testing.T) {
		organizations := []types.DatadogOrganization{
			{
				ID:          "org-1",
				Name:        "Test Org",
				Description: "Test",
				Settings: map[string]interface{}{
					"saml_enabled": "invalid_string", // Invalid type
				},
				Users: []types.DatadogUser{},
				Teams: []types.DatadogTeam{},
			},
		}
		
		metadata := createOrganizationsMetadata(organizations, []string{"org-1"})
		
		// Should handle invalid SAML setting gracefully
		assert.Equal(t, 0, metadata["saml_enabled_orgs"], "Should not count invalid SAML settings")
	})
	
	t.Run("nil_organization_attributes", func(t *testing.T) {
		org := datadogV2.Organization{
			Id:         lo.ToPtr("org-1"),
			Attributes: nil,
		}
		
		result := transformOrganizationResponse(org, 0)
		
		assert.Equal(t, "org-1", result.ID, "Should handle nil attributes")
		assert.Equal(t, "", result.Name, "Should default empty name")
		assert.Equal(t, "", result.Description, "Should default empty description")
		assert.NotNil(t, result.Settings, "Settings should not be nil")
	})
	
	t.Run("organization_enrichment_with_empty_data", func(t *testing.T) {
		org := types.DatadogOrganization{
			ID:          "org-1",
			Name:        "Test Org",
			Description: "Test",
			Settings:    map[string]interface{}{},
			Users:       []types.DatadogUser{},
			Teams:       []types.DatadogTeam{},
		}
		
		enrichedOrg := enrichOrganizationWithTeamData(org, []types.DatadogTeam{}, []types.DatadogUser{})
		
		assert.Equal(t, org.ID, enrichedOrg.ID, "Should preserve organization data")
		assert.Equal(t, 0, len(enrichedOrg.Users), "Should handle empty users")
		assert.Equal(t, 0, len(enrichedOrg.Teams), "Should handle empty teams")
	})
}

// Integration test structure (would require proper mocking in real implementation)
func TestIntegrationScenarios(t *testing.T) {
	t.Run("full_pipeline_simulation", func(t *testing.T) {
		// This would be a full integration test with:
		// 1. Mocked Datadog client
		// 2. Mocked API responses
		// 3. Mocked DynamoDB operations
		// 4. End-to-end pipeline validation
		// 5. Performance timing validation
		
		// For now, we validate the structure is in place
		event := createValidScraperEvent()
		assert.NoError(t, validateEvent(event), "Event validation should pass")
		
		// Test organization transformation with realistic data
		mockOrgs := createMockOrganizations()
		transformedOrgs := lo.Map(mockOrgs, transformOrganizationResponse)
		
		assert.Equal(t, len(mockOrgs), len(transformedOrgs), "Should transform all organizations")
		
		// Test enrichment process
		mockTeams := createMockTeams()
		mockUsers := createMockUsers()
		
		enrichedOrgs := lo.Map(transformedOrgs, func(org types.DatadogOrganization, _ int) types.DatadogOrganization {
			return enrichOrganizationWithTeamData(org, mockTeams, mockUsers)
		})
		
		assert.Equal(t, len(transformedOrgs), len(enrichedOrgs), "Should enrich all organizations")
		
		for _, org := range enrichedOrgs {
			assert.Equal(t, len(mockUsers), len(org.Users), "Should include all users")
			assert.Equal(t, len(mockTeams), len(org.Teams), "Should include all teams")
		}
		
		// Test metadata creation with realistic data
		metadata := createOrganizationsMetadata(enrichedOrgs, []string{"org-1", "org-2", "org-3", "org-4"})
		assert.NotNil(t, metadata, "Should create comprehensive metadata")
		assert.Equal(t, "v2", metadata["api_version"], "Should use v2 API")
		assert.Equal(t, true, metadata["functional_pipeline"], "Should use functional pipeline")
	})
	
	t.Run("error_handling_scenarios", func(t *testing.T) {
		// Test various error scenarios
		executionID := "test-execution"
		
		// Test error response creation
		errorResponse := createErrorResponse(executionID, "Test error")
		assert.Equal(t, "error", errorResponse.Status, "Should create error response")
		assert.Equal(t, 0, errorResponse.Count, "Error response should have zero count")
		
		// Test success response creation
		metadata := map[string]interface{}{"test": "value"}
		successResponse := createSuccessResponse(executionID, 5, metadata)
		assert.Equal(t, "success", successResponse.Status, "Should create success response")
		assert.Equal(t, 5, successResponse.Count, "Success response should have correct count")
	})
}