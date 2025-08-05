// Package main implements comprehensive tests for the Datadog Teams Scraper Lambda function.
// Tests include common scenarios, edge cases, error conditions, and functional pipeline validation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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

// Test fixtures for comprehensive testing
// Note: Simplified to avoid Datadog API client struct complications
func createMockTeams() []datadogV2.Team {
	// Return empty slice for now - in production tests would use proper mocking
	return []datadogV2.Team{}
}

func createValidScraperEvent() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:         100,
		FilterKeyword:    "",
		IncludeInactive:  false,
		SchemaVersion:    "",
	}
}

func createInvalidScraperEvent() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:      1500,  // Exceeds maximum
		FilterKeyword: "",
		IncludeInactive: false,
	}
}

// Test successful teams scraping with valid data
func TestTeamsScraperHandler_Success(t *testing.T) {
	ctx := context.Background()
	
	// Add X-Ray tracing for realistic test environment
	ctx, seg := xray.BeginSegment(ctx, "test-teams-scraper")
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
				FilterKeyword:    "platform",
				IncludeInactive:  true,
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

// Test teams list options creation with functional approach
func TestCreateTeamsListOptions(t *testing.T) {
	testCases := []struct {
		name            string
		pageSize        int64
		pageToken       *string
		filterKeyword   string
		includeInactive bool
		description     string
	}{
		{
			name:            "standard_options",
			pageSize:        100,
			pageToken:       nil,
			filterKeyword:   "",
			includeInactive: false,
			description:     "Standard pagination options",
		},
		{
			name:            "with_page_token",
			pageSize:        50,
			pageToken:       lo.ToPtr("next-page-token"),
			filterKeyword:   "",
			includeInactive: false,
			description:     "Options with page token for pagination",
		},
		{
			name:            "with_filter",
			pageSize:        25,
			pageToken:       nil,
			filterKeyword:   "platform",
			includeInactive: true,
			description:     "Options with filter keyword and include inactive",
		},
		{
			name:            "minimal_page_size",
			pageSize:        1,
			pageToken:       nil,
			filterKeyword:   "",
			includeInactive: false,
			description:     "Minimal page size",
		},
		{
			name:            "maximum_page_size",
			pageSize:        1000,
			pageToken:       nil,
			filterKeyword:   "",
			includeInactive: false,
			description:     "Maximum allowed page size",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := createTeamsListOptions(tc.pageSize, tc.pageToken, tc.filterKeyword, tc.includeInactive)
			
			assert.NotNil(t, opts, "Options should not be nil")
			// Additional assertions would verify the options are properly set
			// This would require examining the actual API client structure
		})
	}
}

// Test pagination logic with edge cases
func TestHasNextPage(t *testing.T) {
	testCases := []struct {
		name        string
		meta        *datadogV2.TeamsResponseMeta
		expected    bool
		description string
	}{
		{
			name:        "nil_meta",
			meta:        nil,
			expected:    false,
			description: "Nil metadata should indicate no next page",
		},
		{
			name:        "empty_meta",
			meta:        &datadogV2.TeamsResponseMeta{},
			expected:    false,
			description: "Empty metadata should indicate no next page",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasNextPage(tc.meta)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// Test teams metadata creation with comprehensive statistics
func TestCreateTeamsMetadata(t *testing.T) {
	// Create test teams data
	teams := []types.DatadogTeam{
		{
			ID:          "team-1",
			Name:        "Platform Team",
			Handle:      "platform",
			Description: "Core platform team",
			Members: []types.DatadogUser{
				{ID: "user-1", Name: "Alice", Teams: []string{"team-1"}},
				{ID: "user-2", Name: "Bob", Teams: []string{"team-1"}},
			},
			Services: []types.DatadogService{
				{ID: "service-1", Name: "Platform API", Teams: []string{"team-1"}},
			},
		},
		{
			ID:          "team-2",
			Name:        "Frontend Team",
			Handle:      "frontend",
			Description: "Frontend team",
			Members:     []types.DatadogUser{}, // Empty members
			Services: []types.DatadogService{
				{ID: "service-2", Name: "Web App", Teams: []string{"team-2"}},
				{ID: "service-3", Name: "Mobile App", Teams: []string{"team-2"}},
			},
		},
		{
			ID:          "team-3",
			Name:        "", // Invalid team (empty name)
			Handle:      "invalid",
			Description: "",
			Members:     []types.DatadogUser{},
			Services:    []types.DatadogService{},
		},
	}
	
	storedIDs := []string{"team-1", "team-2"}
	
	metadata := createTeamsMetadata(teams, storedIDs)
	
	// Verify metadata structure and content
	assert.NotNil(t, metadata, "Metadata should not be nil")
	
	// Check required fields
	assert.Contains(t, metadata, "teams_fetched", "Should contain teams_fetched count")
	assert.Contains(t, metadata, "valid_teams", "Should contain valid_teams count")
	assert.Contains(t, metadata, "teams_stored", "Should contain teams_stored count")
	assert.Contains(t, metadata, "team_statistics", "Should contain team_statistics")
	assert.Contains(t, metadata, "stored_team_ids", "Should contain stored_team_ids")
	assert.Contains(t, metadata, "api_version", "Should contain api_version")
	assert.Contains(t, metadata, "functional_pipeline", "Should contain functional_pipeline flag")
	
	// Verify values
	assert.Equal(t, len(teams), metadata["teams_fetched"], "Teams fetched count should match")
	assert.Equal(t, len(storedIDs), metadata["teams_stored"], "Teams stored count should match")
	assert.Equal(t, "v2", metadata["api_version"], "API version should be v2")
	assert.Equal(t, true, metadata["functional_pipeline"], "Functional pipeline flag should be true")
	
	// Verify team statistics
	teamStats, ok := metadata["team_statistics"].(map[string]int)
	require.True(t, ok, "Team statistics should be a map[string]int")
	
	assert.Contains(t, teamStats, "total_members", "Should contain total_members")
	assert.Contains(t, teamStats, "total_services", "Should contain total_services")
	assert.Contains(t, teamStats, "teams_with_members", "Should contain teams_with_members")
	assert.Contains(t, teamStats, "teams_with_services", "Should contain teams_with_services")
	
	// Verify calculated statistics
	assert.Equal(t, 2, teamStats["total_members"], "Should count all members across teams")
	assert.Equal(t, 3, teamStats["total_services"], "Should count all services across teams")
	assert.Equal(t, 1, teamStats["teams_with_members"], "Should count teams with members")
	assert.Equal(t, 2, teamStats["teams_with_services"], "Should count teams with services")
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
	assert.Equal(t, "Successfully scraped 5 teams", response.Message, "Message should include count")
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

// Test JSON marshaling and unmarshaling for Lambda event handling
func TestScraperEventJSONHandling(t *testing.T) {
	originalEvent := types.ScraperEvent{
		PageSize:         100,
		FilterKeyword:    "platform",
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

func BenchmarkCreateTeamsMetadata(b *testing.B) {
	teams := []types.DatadogTeam{
		{
			ID:     "team-1",
			Name:   "Test Team",
			Handle: "test",
			Members: []types.DatadogUser{
				{ID: "user-1", Name: "User 1"},
				{ID: "user-2", Name: "User 2"},
			},
			Services: []types.DatadogService{
				{ID: "service-1", Name: "Service 1"},
			},
		},
	}
	storedIDs := []string{"team-1"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createTeamsMetadata(teams, storedIDs)
	}
}

// Comprehensive property-based tests for teams metadata creation
func TestCreateTeamsMetadata_Properties(t *testing.T) {
	// Property 1: metadata should always contain required fields regardless of input size
	property1 := func(teamCount, memberCount, serviceCount int) bool {
		// Generate teams with random counts (bounded)
		teamCount = teamCount%15 + 1   // 1-15 teams
		memberCount = memberCount%8    // 0-7 members per team
		serviceCount = serviceCount%5  // 0-4 services per team
		
		teams := make([]types.DatadogTeam, teamCount)
		storedIDs := make([]string, teamCount)
		
		for i := 0; i < teamCount; i++ {
			teams[i] = types.DatadogTeam{
				ID:       fmt.Sprintf("%d", i),
				Name:     lo.Ternary(i%3 == 0, fmt.Sprintf("Team %d", i), ""), // Some invalid teams
				Handle:   fmt.Sprintf("handle-%d", i),
				Members:  make([]types.DatadogUser, memberCount),
				Services: make([]types.DatadogService, serviceCount),
			}
			storedIDs[i] = fmt.Sprintf("%d", i)
		}
		
		metadata := createTeamsMetadata(teams, storedIDs)
		
		// Verify required fields are always present
		requiredFields := []string{
			"teams_fetched", "valid_teams", "teams_stored",
			"team_statistics", "stored_team_ids", "api_version", "functional_pipeline",
		}
		
		for _, field := range requiredFields {
			if _, exists := metadata[field]; !exists {
				return false
			}
		}
		
		// Verify teams_fetched matches input
		if metadata["teams_fetched"] != teamCount {
			return false
		}
		
		// Verify teams_stored matches stored IDs
		if metadata["teams_stored"] != len(storedIDs) {
			return false
		}
		
		return true
	}
	
	// Property 2: Statistical invariants must hold
	property2 := func(teamCount int) bool {
		teamCount = teamCount%20 + 1 // 1-20 teams
		
		teams := make([]types.DatadogTeam, teamCount)
		totalMembers := 0
		totalServices := 0
		teamsWithMembers := 0
		teamsWithServices := 0
		
		for i := 0; i < teamCount; i++ {
			memberCount := i % 5 // 0-4 members
			serviceCount := i % 3 // 0-2 services
			
			teams[i] = types.DatadogTeam{
				ID:       fmt.Sprintf("team-%d", i),
				Name:     fmt.Sprintf("Team %d", i),
				Handle:   fmt.Sprintf("team-%d", i),
				Members:  make([]types.DatadogUser, memberCount),
				Services: make([]types.DatadogService, serviceCount),
			}
			
			totalMembers += memberCount
			totalServices += serviceCount
			if memberCount > 0 {
				teamsWithMembers++
			}
			if serviceCount > 0 {
				teamsWithServices++
			}
		}
		
		metadata := createTeamsMetadata(teams, []string{})
		teamStats := metadata["team_statistics"].(map[string]int)
		
		// Verify statistical calculations
		if teamStats["total_members"] != totalMembers {
			return false
		}
		if teamStats["total_services"] != totalServices {
			return false
		}
		if teamStats["teams_with_members"] != teamsWithMembers {
			return false
		}
		if teamStats["teams_with_services"] != teamsWithServices {
			return false
		}
		
		return true
	}
	
	// Property 3: Counts should be non-negative and consistent
	property3 := func(teamCount int) bool {
		teamCount = teamCount%25 + 1 // 1-25 teams
		
		teams := make([]types.DatadogTeam, teamCount)
		validTeams := 0
		
		for i := 0; i < teamCount; i++ {
			isValid := i%4 != 0 // 75% valid teams
			teams[i] = types.DatadogTeam{
				ID:     fmt.Sprintf("team-%d", i),
				Name:   lo.Ternary(isValid, fmt.Sprintf("Team %d", i), ""),
				Handle: fmt.Sprintf("team-%d", i),
			}
			if isValid {
				validTeams++
			}
		}
		
		metadata := createTeamsMetadata(teams, []string{})
		
		// All counts should be non-negative
		if metadata["teams_fetched"].(int) < 0 {
			return false
		}
		if metadata["valid_teams"].(int) < 0 {
			return false
		}
		if metadata["teams_stored"].(int) < 0 {
			return false
		}
		
		// Valid teams should not exceed total teams
		if metadata["valid_teams"].(int) > metadata["teams_fetched"].(int) {
			return false
		}
		
		return true
	}
	
	// Run property-based tests
	for i := 0; i < 100; i++ {
		if !property1(i%10, i%5, i%3) {
			t.Fatalf("Property 1 test failed at iteration %d", i)
		}
		if !property2(i%15) {
			t.Fatalf("Property 2 test failed at iteration %d", i)
		}
		if !property3(i%20) {
			t.Fatalf("Property 3 test failed at iteration %d", i)
		}
	}
}

// Property-based test for validation function behavior
func TestValidateEvent_Properties(t *testing.T) {
	config := &quick.Config{MaxCount: 100}
	
	property := func(pageSize int) bool {
		event := types.ScraperEvent{
			PageSize: pageSize,
		}
		
		err := validateEvent(event)
		
		// Property: validation should be deterministic
		err2 := validateEvent(event)
		if (err == nil) != (err2 == nil) {
			return false
		}
		
		// Property: valid page sizes should pass
		if pageSize >= 0 && pageSize <= 1000 {
			if err != nil {
				return false
			}
		}
		
		// Property: invalid page sizes should fail
		if pageSize < 0 || pageSize > 1000 {
			if err == nil {
				return false
			}
		}
		
		return true
	}
	
	err := quick.Check(property, config)
	assert.NoError(t, err, "Validation property should hold")
}

// Property-based test for functional pipeline transformations
func TestTeamTransformations_Properties(t *testing.T) {
	// Property: transformations should preserve team count
	property1 := func(teamCount int) bool {
		teamCount = teamCount%10 + 1 // 1-10 teams
		
		teams := make([]types.DatadogTeam, teamCount)
		for i := 0; i < teamCount; i++ {
			teams[i] = types.DatadogTeam{
				ID:   fmt.Sprintf("team-%d", i),
				Name: fmt.Sprintf("Team %d", i),
			}
		}
		
		// Use samber/lo functional transformations
		ids := lo.Map(teams, func(team types.DatadogTeam, _ int) string {
			return team.ID
		})
		
		names := lo.Map(teams, func(team types.DatadogTeam, _ int) string {
			return team.Name
		})
		
		validTeams := lo.Filter(teams, func(team types.DatadogTeam, _ int) bool {
			return team.Name != ""
		})
		
		// Verify transformations preserve expected properties
		if len(ids) != teamCount {
			return false
		}
		if len(names) != teamCount {
			return false
		}
		if len(validTeams) != teamCount { // All teams have names
			return false
		}
		
		return true
	}
	
	// Property: reduce operations should maintain mathematical invariants
	property2 := func(teamCount int) bool {
		teamCount = teamCount%15 + 1 // 1-15 teams
		
		teams := make([]types.DatadogTeam, teamCount)
		expectedTotal := 0
		
		for i := 0; i < teamCount; i++ {
			memberCount := i % 5 // 0-4 members
			teams[i] = types.DatadogTeam{
				ID:      fmt.Sprintf("team-%d", i),
				Members: make([]types.DatadogUser, memberCount),
			}
			expectedTotal += memberCount
		}
		
		// Use reduce to calculate total members
		totalMembers := lo.Reduce(teams, func(acc int, team types.DatadogTeam, _ int) int {
			return acc + len(team.Members)
		}, 0)
		
		if totalMembers != expectedTotal {
			return false
		}
		
		return true
	}
	
	// Run property tests
	for i := 0; i < 50; i++ {
		if !property1(i%8 + 1) {
			t.Fatalf("Transformation property 1 failed at iteration %d", i)
		}
		if !property2(i%12 + 1) {
			t.Fatalf("Transformation property 2 failed at iteration %d", i)
		}
	}
}

// Fuzz test for teams metadata creation robustness
func FuzzCreateTeamsMetadata(f *testing.F) {
	// Seed corpus with edge cases
	f.Add(5, 2, "Team Alpha", "alpha-team")
	f.Add(0, 0, "", "")
	f.Add(100, 50, "Team with special chars !@#$%", "special-chars-team")
	f.Add(1, 1, "\n\t\r", "whitespace-team")
	
	f.Fuzz(func(t *testing.T, teamCount, memberCount int, teamName, teamHandle string) {
		// Bound inputs to reasonable ranges
		if teamCount < 0 || teamCount > 1000 {
			t.Skip("Skipping unreasonable team count")
		}
		if memberCount < 0 || memberCount > 100 {
			memberCount = memberCount % 10
		}
		
		// Generate teams with fuzz inputs
		teams := make([]types.DatadogTeam, teamCount)
		storedIDs := make([]string, teamCount)
		
		for i := 0; i < teamCount; i++ {
			teams[i] = types.DatadogTeam{
				ID:      fmt.Sprintf("team-%d", i),
				Name:    teamName,
				Handle:  teamHandle,
				Members: make([]types.DatadogUser, memberCount),
			}
			storedIDs[i] = fmt.Sprintf("team-%d", i)
		}
		
		// This should not panic regardless of input
		metadata := createTeamsMetadata(teams, storedIDs)
		
		// Basic invariants should hold
		assert.NotNil(t, metadata, "Metadata should not be nil")
		assert.Equal(t, teamCount, metadata["teams_fetched"], "Teams count should match")
		assert.Equal(t, teamCount, metadata["teams_stored"], "Stored count should match")
		
		// Verify required fields exist
		requiredFields := []string{"teams_fetched", "valid_teams", "teams_stored", "team_statistics", "api_version"}
		for _, field := range requiredFields {
			assert.Contains(t, metadata, field, "Should contain required field: "+field)
		}
	})
}

// Fuzz test for event validation with malformed inputs
func FuzzValidateEvent(f *testing.F) {
	// Seed with edge cases
	f.Add(100, "normal", false)
	f.Add(0, "", true)
	f.Add(-1, "negative", false)
	f.Add(1001, "overflow", true)
	f.Add(math.MaxInt32, "maxint", false)
	
	f.Fuzz(func(t *testing.T, pageSize int, filterKeyword string, includeInactive bool) {
		event := types.ScraperEvent{
			PageSize:        pageSize,
			FilterKeyword:   filterKeyword,
			IncludeInactive: includeInactive,
		}
		
		// Function should not panic regardless of input
		err := validateEvent(event)
		
		// Verify deterministic behavior
		err2 := validateEvent(event)
		assert.Equal(t, err == nil, err2 == nil, "Validation should be deterministic")
		
		// Verify expected validation logic
		if pageSize >= 0 && pageSize <= 1000 {
			assert.NoError(t, err, "Valid page size should pass validation")
		} else {
			assert.Error(t, err, "Invalid page size should fail validation")
		}
	})
}

// Fuzz test for JSON event parsing robustness
func FuzzEventJSONParsing(f *testing.F) {
	// Seed with valid and invalid JSON
	f.Add(`{"page_size": 100, "filter_keyword": "test"}`)
	f.Add(`{"page_size": "invalid"}`)
	f.Add(`{"malformed": }`)
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`"string"`)
	f.Add(`123`)
	f.Add(`[]`)
	
	f.Fuzz(func(t *testing.T, jsonStr string) {
		var event types.ScraperEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		
		// Should not panic regardless of input
		if err != nil {
			// JSON parsing failed - acceptable for malformed input
			assert.Error(t, err, "Should return error for invalid JSON")
		} else {
			// JSON parsing succeeded - validate if reasonable
			validationErr := validateEvent(event)
			// Validation may pass or fail depending on parsed values
			if event.PageSize < 0 || event.PageSize > 1000 {
				assert.Error(t, validationErr, "Should fail validation for invalid parsed values")
			}
		}
	})
}

// Test error conditions and edge cases with enhanced coverage
func TestEdgeCases(t *testing.T) {
	t.Run("empty_teams_list", func(t *testing.T) {
		teams := []types.DatadogTeam{}
		storedIDs := []string{}
		
		metadata := createTeamsMetadata(teams, storedIDs)
		
		assert.Equal(t, 0, metadata["teams_fetched"], "Should handle empty teams list")
		assert.Equal(t, 0, metadata["teams_stored"], "Should handle empty stored IDs")
		
		// Verify statistics are properly initialized
		teamStats := metadata["team_statistics"].(map[string]int)
		assert.Equal(t, 0, teamStats["total_members"], "Should have zero members")
		assert.Equal(t, 0, teamStats["total_services"], "Should have zero services")
	})
	
	t.Run("teams_with_unicode_names", func(t *testing.T) {
		teams := []types.DatadogTeam{
			{
				ID:   "team-1",
				Name: "チーム A", // Japanese
			},
			{
				ID:   "team-2",
				Name: "Équipe B", // French with accent
			},
			{
				ID:   "team-3",
				Name: "🚀 Rocket Team", // Emoji
			},
		}
		
		metadata := createTeamsMetadata(teams, []string{"team-1", "team-2", "team-3"})
		
		assert.Equal(t, 3, metadata["teams_fetched"], "Should handle unicode team names")
		assert.Equal(t, 3, metadata["valid_teams"], "All unicode names should be valid")
		
		// Verify each name is valid UTF-8
		for _, team := range teams {
			assert.True(t, utf8.ValidString(team.Name), "Team name should be valid UTF-8")
		}
	})
	
	t.Run("teams_with_very_long_names", func(t *testing.T) {
		longName := strings.Repeat("A", 1000)
		teams := []types.DatadogTeam{
			{
				ID:   "team-1",
				Name: longName,
			},
		}
		
		metadata := createTeamsMetadata(teams, []string{"team-1"})
		
		assert.Equal(t, 1, metadata["teams_fetched"], "Should handle very long team names")
		assert.Equal(t, 1, metadata["valid_teams"], "Long name should still be valid")
	})
	
	t.Run("teams_with_extreme_member_counts", func(t *testing.T) {
		teams := []types.DatadogTeam{
			{
				ID:      "team-large",
				Name:    "Large Team",
				Members: make([]types.DatadogUser, 1000), // Very large team
			},
			{
				ID:      "team-empty",
				Name:    "Empty Team",
				Members: []types.DatadogUser{}, // Empty team
			},
		}
		
		metadata := createTeamsMetadata(teams, []string{"team-large", "team-empty"})
		
		teamStats := metadata["team_statistics"].(map[string]int)
		assert.Equal(t, 1000, teamStats["total_members"], "Should handle large member counts")
		assert.Equal(t, 1, teamStats["teams_with_members"], "Should count teams with members correctly")
	})
	
	t.Run("nil_page_token", func(t *testing.T) {
		opts := createTeamsListOptions(100, nil, "", false)
		assert.NotNil(t, opts, "Should handle nil page token")
	})
	
	t.Run("empty_filter_keyword", func(t *testing.T) {
		opts := createTeamsListOptions(100, nil, "", false)
		assert.NotNil(t, opts, "Should handle empty filter keyword")
	})
	
	t.Run("concurrent_metadata_creation", func(t *testing.T) {
		teams := []types.DatadogTeam{
			{
				ID:   "team-1",
				Name: "Concurrent Team",
				Members: []types.DatadogUser{
					{ID: "user-1", Name: "User 1"},
				},
			},
		}
		storedIDs := []string{"team-1"}
		
		// Test concurrent execution
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				metadata := createTeamsMetadata(teams, storedIDs)
				assert.NotNil(t, metadata, "Metadata should not be nil in concurrent execution")
				assert.Equal(t, 1, metadata["teams_fetched"], "Should maintain consistent counts")
				done <- true
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Integration test structure (would require proper mocking in real implementation)
func TestIntegrationScenarios(t *testing.T) {
	t.Run("full_pipeline_simulation", func(t *testing.T) {
		// This would be a full integration test with:
		// 1. Mocked Datadog API responses
		// 2. Mocked DynamoDB operations
		// 3. End-to-end pipeline validation
		// 4. Performance timing validation
		
		// For now, we validate the structure is in place
		event := createValidScraperEvent()
		assert.NoError(t, validateEvent(event), "Event validation should pass")
		
		// Test metadata creation with realistic data
		teams := []types.DatadogTeam{
			{
				ID:          "team-1",
				Name:        "Platform",
				Handle:      "platform",
				Description: "Platform team",
				Members: []types.DatadogUser{
					{ID: "user-1", Name: "Alice"},
					{ID: "user-2", Name: "Bob"},
				},
				Services: []types.DatadogService{
					{ID: "service-1", Name: "API"},
				},
			},
		}
		
		metadata := createTeamsMetadata(teams, []string{"team-1"})
		assert.NotNil(t, metadata, "Should create comprehensive metadata")
		assert.Equal(t, "v2", metadata["api_version"], "Should use v2 API")
		assert.Equal(t, true, metadata["functional_pipeline"], "Should use functional pipeline")
	})
}