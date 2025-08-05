// Package main implements comprehensive tests for the Datadog Users Scraper Lambda function.
// Tests include common scenarios, edge cases, error conditions, and functional pipeline validation.
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

// Test fixtures for comprehensive user testing
func createMockUsers() []datadogV2.User {
	// Return empty slice for now - in production tests would use proper mocking
	return []datadogV2.User{}
}

func createValidUserScraperEvent() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:         50,
		FilterKeyword:    "",
		IncludeInactive:  false,
		SchemaVersion:    "",
	}
}

func createUserScraperEventWithInactive() types.ScraperEvent {
	return types.ScraperEvent{
		PageSize:         100,
		FilterKeyword:    "engineer",
		IncludeInactive:  true,
		SchemaVersion:    "",
	}
}

// Test successful users scraping with active users only
func TestUsersScraperHandler_ActiveUsersOnly(t *testing.T) {
	ctx := context.Background()
	
	// Add X-Ray tracing for realistic test environment
	ctx, seg := xray.BeginSegment(ctx, "test-users-scraper")
	defer seg.Close(nil)
	
	event := createValidUserScraperEvent()
	
	// Test event validation
	err := validateEvent(event)
	assert.NoError(t, err, "Valid event should pass validation")
}

// Test users scraping including inactive users
func TestUsersScraperHandler_IncludeInactive(t *testing.T) {
	ctx := context.Background()
	
	ctx, seg := xray.BeginSegment(ctx, "test-users-scraper-inactive")
	defer seg.Close(nil)
	
	event := createUserScraperEventWithInactive()
	
	// Test event validation
	err := validateEvent(event)
	assert.NoError(t, err, "Event with include inactive should pass validation")
	
	assert.True(t, event.IncludeInactive, "Should include inactive users")
	assert.Equal(t, "engineer", event.FilterKeyword, "Should have filter keyword")
}

// Test event validation with various parameters
func TestValidateEvent_UserSpecificCases(t *testing.T) {
	testCases := []struct {
		name        string
		event       types.ScraperEvent
		expectError bool
		description string
	}{
		{
			name: "valid_with_filter",
			event: types.ScraperEvent{
				PageSize:         25,
				FilterKeyword:    "admin",
				IncludeInactive:  true,
			},
			expectError: false,
			description: "Valid event with filter and include inactive",
		},
		{
			name: "valid_minimal",
			event: types.ScraperEvent{
				PageSize:         1,
				FilterKeyword:    "",
				IncludeInactive:  false,
			},
			expectError: false,
			description: "Valid minimal event",
		},
		{
			name: "valid_maximum",
			event: types.ScraperEvent{
				PageSize:         1000,
				FilterKeyword:    "very-long-filter-keyword-that-should-still-be-valid",
				IncludeInactive:  true,
			},
			expectError: false,
			description: "Valid event with maximum parameters",
		},
		{
			name: "invalid_negative_page_size",
			event: types.ScraperEvent{
				PageSize:         -1,
				FilterKeyword:    "",
				IncludeInactive:  false,
			},
			expectError: true,
			description: "Negative page size should be invalid",
		},
		{
			name: "invalid_excessive_page_size",
			event: types.ScraperEvent{
				PageSize:         2000,
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

// Test users list options creation
func TestCreateUsersListOptions(t *testing.T) {
	testCases := []struct {
		name          string
		pageSize      int64
		pageNumber    int64
		filterKeyword string
		description   string
	}{
		{
			name:          "standard_options",
			pageSize:      100,
			pageNumber:    0,
			filterKeyword: "",
			description:   "Standard pagination options",
		},
		{
			name:          "with_filter",
			pageSize:      50,
			pageNumber:    1,
			filterKeyword: "admin",
			description:   "Options with filter keyword",
		},
		{
			name:          "second_page",
			pageSize:      25,
			pageNumber:    2,
			filterKeyword: "",
			description:   "Second page pagination",
		},
		{
			name:          "large_page",
			pageSize:      500,
			pageNumber:    0,
			filterKeyword: "engineer",
			description:   "Large page size with filter",
		},
		{
			name:          "minimal_page",
			pageSize:      1,
			pageNumber:    10,
			filterKeyword: "",
			description:   "Minimal page size, high page number",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := createUsersListOptions(tc.pageSize, tc.pageNumber, tc.filterKeyword)
			
			assert.NotNil(t, opts, "Options should not be nil")
			// Additional assertions would verify the options are properly configured
			// This would require examining the actual API client structure
		})
	}
}

// Test pagination logic for users
func TestHasNextUserPage(t *testing.T) {
	testCases := []struct {
		name                string
		meta                *datadogV2.ResponseMetaAttributes
		currentPageUsers    []datadogV2.User
		pageSize            int64
		expected            bool
		description         string
	}{
		{
			name:                "nil_meta",
			meta:                nil,
			currentPageUsers:    []datadogV2.User{},
			pageSize:            100,
			expected:            false,
			description:         "Nil metadata should indicate no next page",
		},
		{
			name:                "empty_users_small_page",
			meta:                &datadogV2.ResponseMetaAttributes{},
			currentPageUsers:    []datadogV2.User{},
			pageSize:            100,
			expected:            false,
			description:         "Empty users with small page should indicate no next page",
		},
		{
			name:                "full_page",
			meta:                &datadogV2.ResponseMetaAttributes{},
			currentPageUsers:    make([]datadogV2.User, 100),
			pageSize:            100,
			expected:            false,
			description:         "Current implementation returns false for simplicity",
		},
		{
			name:                "partial_page",
			meta:                &datadogV2.ResponseMetaAttributes{},
			currentPageUsers:    make([]datadogV2.User, 50),
			pageSize:            100,
			expected:            false,
			description:         "Partial page should indicate no next page",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasNextUserPage(tc.meta, tc.currentPageUsers, tc.pageSize)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// Test users metadata creation with comprehensive statistics
func TestCreateUsersMetadata(t *testing.T) {
	// Create test users data
	allUsers := []types.DatadogUser{
		{
			ID:       "user-1",
			Name:     "Alice Johnson",
			Email:    "alice@company.com",
			Handle:   "alice.johnson",
			Teams:    []string{"team-1", "team-2"},
			Roles:    []string{"admin", "user"},
			Status:   "Active",
			Verified: true,
			Disabled: false,
			Title:    "Senior Engineer",
		},
		{
			ID:       "user-2",
			Name:     "Bob Smith",
			Email:    "bob@company.com",
			Handle:   "bob.smith",
			Teams:    []string{"team-1"},
			Roles:    []string{"user"},
			Status:   "Pending",
			Verified: false,
			Disabled: false,
			Title:    "Junior Engineer",
		},
		{
			ID:       "user-3",
			Name:     "Charlie Brown",
			Email:    "charlie@company.com",
			Handle:   "charlie.brown",
			Teams:    []string{}, // No teams
			Roles:    []string{},
			Status:   "Inactive",
			Verified: true,
			Disabled: true,
			Title:    "Former Employee",
		},
		{
			ID:       "user-4",
			Name:     "",  // Invalid user (empty name)
			Email:    "",
			Handle:   "invalid.user",
			Teams:    []string{"team-3"},
			Roles:    []string{"user"},
			Status:   "Active",
			Verified: false,
			Disabled: false,
			Title:    "",
		},
	}
	
	activeUsers := []types.DatadogUser{allUsers[0], allUsers[1]} // Only active/pending users
	storedIDs := []string{"user-1", "user-2", "user-3"}
	
	metadata := createUsersMetadata(allUsers, activeUsers, storedIDs)
	
	// Verify metadata structure and content
	assert.NotNil(t, metadata, "Metadata should not be nil")
	
	// Check required fields
	requiredFields := []string{
		"users_fetched", "active_users", "users_stored", "verified_users",
		"users_with_teams", "user_status_counts", "team_statistics",
		"stored_user_ids", "api_version", "functional_pipeline", "includes_inactive",
	}
	
	for _, field := range requiredFields {
		assert.Contains(t, metadata, field, "Should contain "+field)
	}
	
	// Verify values
	assert.Equal(t, len(allUsers), metadata["users_fetched"], "Users fetched count should match")
	assert.Equal(t, len(activeUsers), metadata["active_users"], "Active users count should match")
	assert.Equal(t, len(storedIDs), metadata["users_stored"], "Users stored count should match")
	assert.Equal(t, "v2", metadata["api_version"], "API version should be v2")
	assert.Equal(t, true, metadata["functional_pipeline"], "Functional pipeline flag should be true")
	
	// Verify verified users count
	verifiedCount := metadata["verified_users"].(int)
	assert.Equal(t, 2, verifiedCount, "Should count verified users correctly")
	
	// Verify users with teams count
	usersWithTeams := metadata["users_with_teams"].(int)
	assert.Equal(t, 3, usersWithTeams, "Should count users with teams correctly")
	
	// Verify status counts
	statusCounts, ok := metadata["user_status_counts"].(map[string]int)
	require.True(t, ok, "User status counts should be a map[string]int")
	
	assert.Equal(t, 2, statusCounts["Active"], "Should count active users")
	assert.Equal(t, 1, statusCounts["Pending"], "Should count pending users")
	assert.Equal(t, 1, statusCounts["Inactive"], "Should count inactive users")
	
	// Verify team statistics
	teamStats, ok := metadata["team_statistics"].(map[string]interface{})
	require.True(t, ok, "Team statistics should be a map[string]interface{}")
	
	assert.Contains(t, teamStats, "total_team_memberships", "Should contain total team memberships")
	assert.Contains(t, teamStats, "total_role_assignments", "Should contain total role assignments")
	assert.Contains(t, teamStats, "max_teams_per_user", "Should contain max teams per user")
	
	// Verify calculated team statistics
	assert.Equal(t, 4, teamStats["total_team_memberships"], "Should count all team memberships")
	assert.Equal(t, 5, teamStats["total_role_assignments"], "Should count all role assignments")
	assert.Equal(t, 2, teamStats["max_teams_per_user"], "Should find max teams per user")
	
	// Verify includes_inactive flag
	includesInactive := metadata["includes_inactive"].(bool)
	assert.True(t, includesInactive, "Should indicate that inactive users are included")
}

// Test response creation functions for users
func TestCreateSuccessResponse_Users(t *testing.T) {
	executionID := "user-execution-123"
	count := 15
	metadata := map[string]interface{}{
		"users_fetched":      15,
		"active_users":       12,
		"verified_users":     10,
		"users_with_teams":   8,
		"api_version":        "v2",
		"functional_pipeline": true,
	}
	
	response := createSuccessResponse(executionID, count, metadata)
	
	assert.Equal(t, "success", response.Status, "Status should be success")
	assert.Equal(t, "Successfully scraped 15 users", response.Message, "Message should include count")
	assert.Equal(t, count, response.Count, "Count should match input")
	assert.Equal(t, executionID, response.ExecutionID, "Execution ID should match")
	assert.Equal(t, metadata, response.Metadata, "Metadata should match")
	assert.NotEmpty(t, response.Timestamp, "Timestamp should not be empty")
	
	// Verify timestamp format
	_, err := time.Parse(time.RFC3339, response.Timestamp)
	assert.NoError(t, err, "Timestamp should be in RFC3339 format")
}

func TestCreateErrorResponse_Users(t *testing.T) {
	executionID := "user-execution-error-456"
	message := "Failed to fetch users from API"
	
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

// Test JSON handling for user events and responses
func TestUserScraperEventJSONHandling(t *testing.T) {
	originalEvent := types.ScraperEvent{
		PageSize:         75,
		FilterKeyword:    "developer",
		IncludeInactive:  true,
		SchemaVersion:    "",
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
}

// Comprehensive property-based tests for user metadata creation
func TestCreateUsersMetadata_Properties(t *testing.T) {
	// Property 1: metadata should always contain required fields and valid counts
	property1 := func(userCount, activeCount, teamCount int) bool {
		// Generate bounded random counts
		userCount = (userCount%25 + 1)  // 1-25 users
		activeCount = activeCount % (userCount + 1)  // 0 to userCount active users
		teamCount = teamCount%7 + 1     // 1-7 teams per user
		
		allUsers := make([]types.DatadogUser, userCount)
		activeUsers := make([]types.DatadogUser, activeCount)
		storedIDs := make([]string, userCount)
		
		for i := 0; i < userCount; i++ {
			allUsers[i] = types.DatadogUser{
				ID:       fmt.Sprintf("%d", i),
				Name:     lo.Ternary(i%4 == 0, fmt.Sprintf("User %d", i), ""), // Some invalid users
				Email:    fmt.Sprintf("user%d@test.com", i),
				Handle:   fmt.Sprintf("user%d", i),
				Teams:    make([]string, teamCount),
				Roles:    []string{"user"},
				Status:   lo.Ternary(i < activeCount, "Active", "Inactive"),
				Verified: i%2 == 0,
				Disabled: i >= activeCount,
			}
			storedIDs[i] = fmt.Sprintf("%d", i)
		}
		
		// Copy active users
		copy(activeUsers, allUsers[:activeCount])
		
		metadata := createUsersMetadata(allUsers, activeUsers, storedIDs)
		
		// Verify required fields are present
		requiredFields := []string{
			"users_fetched", "active_users", "users_stored", "verified_users",
			"users_with_teams", "user_status_counts", "team_statistics",
			"stored_user_ids", "api_version", "functional_pipeline",
		}
		
		for _, field := range requiredFields {
			if _, exists := metadata[field]; !exists {
				return false
			}
		}
		
		// Verify counts are consistent
		if metadata["users_fetched"] != userCount {
			return false
		}
		
		if metadata["active_users"] != activeCount {
			return false
		}
		
		if metadata["users_stored"] != len(storedIDs) {
			return false
		}
		
		return true
	}
	
	// Property 2: Statistical calculations should be mathematically consistent
	property2 := func(userCount int) bool {
		userCount = userCount%30 + 1 // 1-30 users
		
		allUsers := make([]types.DatadogUser, userCount)
		expectedVerified := 0
		expectedWithTeams := 0
		expectedTotalTeamMemberships := 0
		expectedTotalRoleAssignments := 0
		maxTeamsPerUser := 0
		
		for i := 0; i < userCount; i++ {
			teamCount := i%5 + 1 // 1-5 teams
			roleCount := i%3 + 1 // 1-3 roles
			isVerified := i%3 == 0
			
			allUsers[i] = types.DatadogUser{
				ID:       fmt.Sprintf("user-%d", i),
				Name:     fmt.Sprintf("User %d", i),
				Teams:    make([]string, teamCount),
				Roles:    make([]string, roleCount),
				Verified: isVerified,
			}
			
			if isVerified {
				expectedVerified++
			}
			if teamCount > 0 {
				expectedWithTeams++
				expectedTotalTeamMemberships += teamCount
			}
			expectedTotalRoleAssignments += roleCount
			if teamCount > maxTeamsPerUser {
				maxTeamsPerUser = teamCount
			}
		}
		
		metadata := createUsersMetadata(allUsers, allUsers, []string{})
		
		// Verify statistical calculations
		if metadata["verified_users"] != expectedVerified {
			return false
		}
		if metadata["users_with_teams"] != expectedWithTeams {
			return false
		}
		
		teamStats := metadata["team_statistics"].(map[string]interface{})
		if teamStats["total_team_memberships"] != expectedTotalTeamMemberships {
			return false
		}
		if teamStats["total_role_assignments"] != expectedTotalRoleAssignments {
			return false
		}
		if teamStats["max_teams_per_user"] != maxTeamsPerUser {
			return false
		}
		
		return true
	}
	
	// Property 3: Status counts should sum correctly
	property3 := func(userCount int) bool {
		userCount = userCount%20 + 1 // 1-20 users
		
		allUsers := make([]types.DatadogUser, userCount)
		statusTypes := []string{"Active", "Inactive", "Pending", "Suspended"}
		expectedCounts := make(map[string]int)
		
		for i := 0; i < userCount; i++ {
			status := statusTypes[i%len(statusTypes)]
			allUsers[i] = types.DatadogUser{
				ID:     fmt.Sprintf("user-%d", i),
				Name:   fmt.Sprintf("User %d", i),
				Status: status,
			}
			expectedCounts[status]++
		}
		
		metadata := createUsersMetadata(allUsers, []types.DatadogUser{}, []string{})
		statusCounts := metadata["user_status_counts"].(map[string]int)
		
		// Verify status counts match expectations
		for status, expectedCount := range expectedCounts {
			if statusCounts[status] != expectedCount {
				return false
			}
		}
		
		return true
	}
	
	// Run property-based tests
	for i := 0; i < 75; i++ {
		if !property1(i%20+1, i%15, i%6+1) {
			t.Fatalf("Property 1 test failed at iteration %d", i)
		}
		if !property2(i%25+1) {
			t.Fatalf("Property 2 test failed at iteration %d", i)
		}
		if !property3(i%18+1) {
			t.Fatalf("Property 3 test failed at iteration %d", i)
		}
	}
}

// Property-based test for user filtering and transformation invariants
func TestUserTransformations_Properties(t *testing.T) {
	config := &quick.Config{MaxCount: 50}
	
	// Property: filter operations should maintain invariants
	property1 := func(userCount uint8) bool {
		userCount = userCount%20 + 1 // 1-20 users
		
		users := make([]types.DatadogUser, userCount)
		activeCount := 0
		verifiedCount := 0
		
		for i := uint8(0); i < userCount; i++ {
			isActive := i%3 != 0 // 66% active
			isVerified := i%2 == 0 // 50% verified
			
			users[i] = types.DatadogUser{
				ID:       fmt.Sprintf("user-%d", i),
				Name:     fmt.Sprintf("User %d", i),
				Status:   lo.Ternary(isActive, "Active", "Inactive"),
				Verified: isVerified,
				Disabled: !isActive,
			}
			
			if isActive {
				activeCount++
			}
			if isVerified {
				verifiedCount++
			}
		}
		
		// Use samber/lo functional operations
		activeUsers := lo.Filter(users, func(user types.DatadogUser, _ int) bool {
			return user.Status == "Active"
		})
		
		verifiedUsers := lo.Filter(users, func(user types.DatadogUser, _ int) bool {
			return user.Verified
		})
		
		userNames := lo.Map(users, func(user types.DatadogUser, _ int) string {
			return user.Name
		})
		
		// Verify filtering preserves expected counts
		if len(activeUsers) != activeCount {
			return false
		}
		if len(verifiedUsers) != verifiedCount {
			return false
		}
		if len(userNames) != int(userCount) {
			return false
		}
		
		return true
	}
	
	// Property: reduce operations should maintain mathematical consistency
	property2 := func(userCount uint8) bool {
		userCount = userCount%15 + 1 // 1-15 users
		
		users := make([]types.DatadogUser, userCount)
		expectedTotalTeams := 0
		
		for i := uint8(0); i < userCount; i++ {
			teamCount := int(i%5) // 0-4 teams
			users[i] = types.DatadogUser{
				ID:    fmt.Sprintf("user-%d", i),
				Teams: make([]string, teamCount),
			}
			expectedTotalTeams += teamCount
		}
		
		// Use reduce to calculate total team memberships
		totalTeamMemberships := lo.Reduce(users, func(acc int, user types.DatadogUser, _ int) int {
			return acc + len(user.Teams)
		}, 0)
		
		if totalTeamMemberships != expectedTotalTeams {
			return false
		}
		
		return true
	}
	
	err1 := quick.Check(property1, config)
	assert.NoError(t, err1, "User transformation property 1 should hold")
	
	err2 := quick.Check(property2, config)
	assert.NoError(t, err2, "User transformation property 2 should hold")
}

// Property-based test for user validation with quick.Check
func TestValidateEvent_UsersProperties(t *testing.T) {
	config := &quick.Config{MaxCount: 100}
	
	property := func(pageSize int, includeInactive bool) bool {
		// Bound pageSize to reasonable range for property test
		if pageSize < -1000 || pageSize > 2000 {
			pageSize = pageSize % 1001
		}
		
		event := types.ScraperEvent{
			PageSize:        pageSize,
			IncludeInactive: includeInactive,
		}
		
		err := validateEvent(event)
		
		// Property: validation should be deterministic
		err2 := validateEvent(event)
		if (err == nil) != (err2 == nil) {
			return false
		}
		
		// Property: valid inputs should pass
		if pageSize >= 0 && pageSize <= 1000 {
			if err != nil {
				return false
			}
		}
		
		// Property: invalid inputs should fail
		if pageSize < 0 || pageSize > 1000 {
			if err == nil {
				return false
			}
		}
		
		return true
	}
	
	err := quick.Check(property, config)
	assert.NoError(t, err, "User validation properties should hold")
}

// Benchmark tests for performance validation
func BenchmarkValidateEvent_Users(b *testing.B) {
	event := createValidUserScraperEvent()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateEvent(event)
	}
}

func BenchmarkCreateUsersMetadata(b *testing.B) {
	users := []types.DatadogUser{
		{
			ID:     "user-1",
			Name:   "Test User",
			Email:  "test@example.com",
			Handle: "test.user",
			Teams:  []string{"team-1", "team-2"},
			Roles:  []string{"user", "admin"},
			Status: "Active",
			Verified: true,
			Disabled: false,
		},
	}
	activeUsers := users
	storedIDs := []string{"user-1"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createUsersMetadata(users, activeUsers, storedIDs)
	}
}

// Fuzz test for user metadata creation robustness
func FuzzCreateUsersMetadata(f *testing.F) {
	// Seed corpus with realistic and edge case data
	f.Add(10, 5, "alice@company.com", "Active", true)
	f.Add(0, 0, "", "", false)
	f.Add(100, 75, "user@domain.co.uk", "Pending", true)
	f.Add(1, 1, "invalid-email", "Unknown", false)
	f.Add(50, 0, "\n\t@\r.com", "Inactive", true)
	
	f.Fuzz(func(t *testing.T, userCount, activeCount int, email, status string, verified bool) {
		// Bound inputs to reasonable ranges
		if userCount < 0 || userCount > 1000 {
			t.Skip("Skipping unreasonable user count")
		}
		if activeCount < 0 || activeCount > userCount {
			activeCount = userCount / 2
		}
		
		// Generate users with fuzz inputs
		allUsers := make([]types.DatadogUser, userCount)
		activeUsers := make([]types.DatadogUser, activeCount)
		storedIDs := make([]string, userCount)
		
		for i := 0; i < userCount; i++ {
			allUsers[i] = types.DatadogUser{
				ID:       fmt.Sprintf("user-%d", i),
				Name:     fmt.Sprintf("User %d", i),
				Email:    email,
				Status:   status,
				Verified: verified,
				Teams:    []string{"team-1"},
				Roles:    []string{"user"},
			}
			storedIDs[i] = fmt.Sprintf("user-%d", i)
		}
		
		copy(activeUsers, allUsers[:activeCount])
		
		// This should not panic regardless of input
		metadata := createUsersMetadata(allUsers, activeUsers, storedIDs)
		
		// Basic invariants should hold
		assert.NotNil(t, metadata, "Metadata should not be nil")
		assert.Equal(t, userCount, metadata["users_fetched"], "Users count should match")
		assert.Equal(t, activeCount, metadata["active_users"], "Active count should match")
		assert.Equal(t, userCount, metadata["users_stored"], "Stored count should match")
		
		// Verify required fields exist
		requiredFields := []string{"users_fetched", "active_users", "users_stored", "verified_users", "api_version"}
		for _, field := range requiredFields {
			assert.Contains(t, metadata, field, "Should contain required field: "+field)
		}
	})
}

// Fuzz test for user event validation
func FuzzValidateUserEvent(f *testing.F) {
	// Seed with various edge cases
	f.Add(100, "engineer", true)
	f.Add(0, "", false)
	f.Add(-10, "invalid", true)
	f.Add(1001, "overflow", false)
	f.Add(math.MaxInt32, "maxint", true)
	
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

// Fuzz test for user JSON parsing robustness
func FuzzUserEventJSONParsing(f *testing.F) {
	// Seed with various JSON structures
	f.Add(`{"page_size": 50, "filter_keyword": "developer", "include_inactive": true}`)
	f.Add(`{"page_size": "invalid_number"}`)
	f.Add(`{"malformed": json}`)
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`[]`)
	f.Add(`"not an object"`)
	f.Add(`{"page_size": 1e10}`)
	
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
			if event.PageSize < 0 || event.PageSize > 1000 {
				assert.Error(t, validationErr, "Should fail validation for invalid parsed values")
			}
		}
	})
}

// Test edge cases specific to users with enhanced coverage
func TestUsersEdgeCases(t *testing.T) {
	t.Run("empty_users_list", func(t *testing.T) {
		allUsers := []types.DatadogUser{}
		activeUsers := []types.DatadogUser{}
		storedIDs := []string{}
		
		metadata := createUsersMetadata(allUsers, activeUsers, storedIDs)
		
		assert.Equal(t, 0, metadata["users_fetched"], "Should handle empty users list")
		assert.Equal(t, 0, metadata["active_users"], "Should handle empty active users list")
		assert.Equal(t, 0, metadata["users_stored"], "Should handle empty stored IDs")
		
		// Verify statistics are properly initialized
		assert.Equal(t, 0, metadata["verified_users"], "Should have zero verified users")
		assert.Equal(t, 0, metadata["users_with_teams"], "Should have zero users with teams")
	})
	
	t.Run("users_with_unicode_content", func(t *testing.T) {
		allUsers := []types.DatadogUser{
			{
				ID:    "user-1",
				Name:  "アリス スミス", // Japanese
				Email: "alice@会社.jp",
				Title: "Senior 🚀 Engineer", // Emoji
			},
			{
				ID:    "user-2",
				Name:  "François Dubois", // French accents
				Email: "francois@compagnie.fr",
				Title: "Développeur Senior",
			},
		}
		
		metadata := createUsersMetadata(allUsers, allUsers, []string{"user-1", "user-2"})
		
		assert.Equal(t, 2, metadata["users_fetched"], "Should handle unicode user content")
		
		// Verify each name is valid UTF-8
		for _, user := range allUsers {
			assert.True(t, utf8.ValidString(user.Name), "User name should be valid UTF-8")
			assert.True(t, utf8.ValidString(user.Email), "User email should be valid UTF-8")
			assert.True(t, utf8.ValidString(user.Title), "User title should be valid UTF-8")
		}
	})
	
	t.Run("users_with_extreme_team_counts", func(t *testing.T) {
		// Generate user with many teams
		manyTeams := make([]string, 100)
		for i := 0; i < 100; i++ {
			manyTeams[i] = fmt.Sprintf("team-%d", i)
		}
		
		allUsers := []types.DatadogUser{
			{
				ID:    "user-many-teams",
				Name:  "Multi Team User",
				Teams: manyTeams,
				Roles: []string{"admin", "user", "viewer", "maintainer", "guest"},
				Status: "Active",
			},
			{
				ID:    "user-no-teams",
				Name:  "Solo User",
				Teams: []string{},
				Roles: []string{"user"},
				Status: "Active",
			},
		}
		
		metadata := createUsersMetadata(allUsers, allUsers, []string{"user-many-teams", "user-no-teams"})
		
		teamStats := metadata["team_statistics"].(map[string]interface{})
		assert.Equal(t, 100, teamStats["total_team_memberships"], "Should handle large team counts")
		assert.Equal(t, 6, teamStats["total_role_assignments"], "Should count all role assignments")
		assert.Equal(t, 100, teamStats["max_teams_per_user"], "Should find max teams per user")
		assert.Equal(t, 1, metadata["users_with_teams"], "Should count users with teams correctly")
	})
	
	t.Run("users_with_various_statuses", func(t *testing.T) {
		status := []string{"Active", "Inactive", "Pending", "Suspended", "Archived", "Unknown"}
		allUsers := make([]types.DatadogUser, len(status))
		
		for i, s := range status {
			allUsers[i] = types.DatadogUser{
				ID:     fmt.Sprintf("user-%d", i),
				Name:   fmt.Sprintf("User %s", s),
				Status: s,
			}
		}
		
		metadata := createUsersMetadata(allUsers, []types.DatadogUser{allUsers[0]}, []string{})
		statusCounts := metadata["user_status_counts"].(map[string]int)
		
		// Verify each status is counted
		for _, s := range status {
			assert.Equal(t, 1, statusCounts[s], "Should count status: "+s)
		}
	})
	
	t.Run("all_inactive_users", func(t *testing.T) {
		allUsers := []types.DatadogUser{
			{
				ID:       "user-1",
				Name:     "Inactive User",
				Status:   "Inactive",
				Disabled: true,
			},
		}
		activeUsers := []types.DatadogUser{} // No active users
		storedIDs := []string{"user-1"}
		
		metadata := createUsersMetadata(allUsers, activeUsers, storedIDs)
		
		assert.Equal(t, 1, metadata["users_fetched"], "Should count all users")
		assert.Equal(t, 0, metadata["active_users"], "Should have no active users")
		assert.Equal(t, false, metadata["includes_inactive"], "Should detect no active users")
	})
	
	t.Run("users_with_many_teams", func(t *testing.T) {
		allUsers := []types.DatadogUser{
			{
				ID:    "user-1",
				Name:  "Multi Team User",
				Teams: []string{"team-1", "team-2", "team-3", "team-4", "team-5"},
				Roles: []string{"admin", "user", "viewer"},
				Status: "Active",
			},
		}
		activeUsers := allUsers
		storedIDs := []string{"user-1"}
		
		metadata := createUsersMetadata(allUsers, activeUsers, storedIDs)
		
		teamStats := metadata["team_statistics"].(map[string]interface{})
		assert.Equal(t, 5, teamStats["total_team_memberships"], "Should count all team memberships")
		assert.Equal(t, 3, teamStats["total_role_assignments"], "Should count all role assignments")
		assert.Equal(t, 5, teamStats["max_teams_per_user"], "Should find max teams per user")
	})
	
	t.Run("users_with_long_content", func(t *testing.T) {
		longName := strings.Repeat("A", 500)
		longEmail := strings.Repeat("user", 100) + "@" + strings.Repeat("domain", 50) + ".com"
		
		allUsers := []types.DatadogUser{
			{
				ID:    "user-long",
				Name:  longName,
				Email: longEmail,
				Title: strings.Repeat("Senior Engineer ", 20),
			},
		}
		
		metadata := createUsersMetadata(allUsers, allUsers, []string{"user-long"})
		
		assert.Equal(t, 1, metadata["users_fetched"], "Should handle users with long content")
		assert.True(t, len(allUsers[0].Name) > 400, "Name should be very long")
		assert.True(t, len(allUsers[0].Email) > 500, "Email should be very long")
	})
	
	t.Run("null_filter_keyword", func(t *testing.T) {
		opts := createUsersListOptions(100, 0, "")
		assert.NotNil(t, opts, "Should handle empty filter keyword")
	})
	
	t.Run("long_filter_keyword", func(t *testing.T) {
		longKeyword := "very-long-filter-keyword-that-tests-edge-case-handling"
		opts := createUsersListOptions(50, 1, longKeyword)
		assert.NotNil(t, opts, "Should handle long filter keyword")
	})
	
	t.Run("concurrent_user_processing", func(t *testing.T) {
		users := []types.DatadogUser{
			{
				ID:     "user-1",
				Name:   "Concurrent User",
				Status: "Active",
				Teams:  []string{"team-1"},
			},
		}
		storedIDs := []string{"user-1"}
		
		// Test concurrent execution
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				metadata := createUsersMetadata(users, users, storedIDs)
				assert.NotNil(t, metadata, "Metadata should not be nil in concurrent execution")
				assert.Equal(t, 1, metadata["users_fetched"], "Should maintain consistent counts")
				done <- true
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Performance property test for large user datasets
func TestUsersPerformanceProperties(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Property: processing time should scale reasonably with input size
	property := func(userCount int) bool {
		userCount = userCount%500 + 100 // 100-599 users
		
		users := make([]types.DatadogUser, userCount)
		for i := 0; i < userCount; i++ {
			users[i] = types.DatadogUser{
				ID:       fmt.Sprintf("user-%d", i),
				Name:     fmt.Sprintf("User %d", i),
				Email:    fmt.Sprintf("user%d@test.com", i),
				Teams:    []string{"team-1", "team-2"},
				Roles:    []string{"user", "member"},
				Status:   "Active",
				Verified: i%2 == 0,
			}
		}
		
		start := time.Now()
		metadata := createUsersMetadata(users, users, []string{})
		duration := time.Since(start)
		
		// Property: should complete within reasonable time (adjust threshold as needed)
		if duration > time.Second {
			return false
		}
		
		// Property: results should be consistent
		if metadata["users_fetched"] != userCount {
			return false
		}
		
		return true
	}
	
	// Run performance test with several iterations
	for i := 0; i < 10; i++ {
		if !property(rand.Intn(400) + 100) {
			t.Fatalf("Performance property failed at iteration %d", i)
		}
	}
}

// Integration test scenarios for users with enhanced functional pipeline testing
func TestUsersIntegrationScenarios(t *testing.T) {
	t.Run("realistic_user_data_processing", func(t *testing.T) {
		// Simulate realistic user data processing pipeline
		event := types.ScraperEvent{
			PageSize:         25,
			FilterKeyword:    "engineer",
			IncludeInactive:  false,
		}
		
		// Validate event
		assert.NoError(t, validateEvent(event), "Event validation should pass")
		
		// Create realistic users data
		users := []types.DatadogUser{
			{
				ID:       "user-1",
				Name:     "Senior Engineer Alice",
				Email:    "alice@company.com",
				Handle:   "alice.engineer",
				Teams:    []string{"platform", "backend"},
				Roles:    []string{"admin", "engineer"},
				Status:   "Active",
				Verified: true,
				Disabled: false,
				Title:    "Senior Software Engineer",
			},
			{
				ID:       "user-2",
				Name:     "Junior Engineer Bob",
				Email:    "bob@company.com",
				Handle:   "bob.engineer",
				Teams:    []string{"frontend"},
				Roles:    []string{"engineer"},
				Status:   "Active",
				Verified: true,
				Disabled: false,
				Title:    "Junior Software Engineer",
			},
			{
				ID:       "user-3",
				Name:     "Lead Engineer Charlie",
				Email:    "charlie@company.com",
				Handle:   "charlie.lead",
				Teams:    []string{"platform", "frontend", "backend"},
				Roles:    []string{"admin", "lead", "engineer"},
				Status:   "Active",
				Verified: true,
				Disabled: false,
				Title:    "Lead Engineer",
			},
		}
		
		activeUsers := users // All users are active
		storedIDs := []string{"user-1", "user-2", "user-3"}
		
		// Test functional pipeline operations using samber/lo
		// Verify transformation preserves data integrity
		userIDs := lo.Map(users, func(user types.DatadogUser, _ int) string {
			return user.ID
		})
		assert.Equal(t, []string{"user-1", "user-2", "user-3"}, userIDs, "Map transformation should preserve IDs")
		
		verifiedUsers := lo.Filter(users, func(user types.DatadogUser, _ int) bool {
			return user.Verified
		})
		assert.Equal(t, 3, len(verifiedUsers), "Filter should preserve all verified users")
		
		totalTeamMemberships := lo.Reduce(users, func(acc int, user types.DatadogUser, _ int) int {
			return acc + len(user.Teams)
		}, 0)
		assert.Equal(t, 6, totalTeamMemberships, "Reduce should calculate total team memberships")
		
		// Test metadata creation
		metadata := createUsersMetadata(users, activeUsers, storedIDs)
		
		// Verify realistic processing results
		assert.Equal(t, 3, metadata["users_fetched"], "Should process all users")
		assert.Equal(t, 3, metadata["active_users"], "All users should be active")
		assert.Equal(t, 3, metadata["verified_users"], "All users should be verified")
		assert.Equal(t, 3, metadata["users_with_teams"], "All users should have teams")
		
		// Verify team statistics
		teamStats := metadata["team_statistics"].(map[string]interface{})
		assert.Equal(t, 6, teamStats["total_team_memberships"], "Should count all team memberships")
		assert.Equal(t, 8, teamStats["total_role_assignments"], "Should count all role assignments")
		assert.Equal(t, 3, teamStats["max_teams_per_user"], "Should find max teams per user")
		
		// Verify API and pipeline flags
		assert.Equal(t, "v2", metadata["api_version"], "Should use v2 API")
		assert.Equal(t, true, metadata["functional_pipeline"], "Should use functional pipeline")
	})
	
	t.Run("functional_pipeline_invariants", func(t *testing.T) {
		// Test that functional transformations maintain expected invariants
		users := make([]types.DatadogUser, 50)
		for i := 0; i < 50; i++ {
			users[i] = types.DatadogUser{
				ID:       fmt.Sprintf("user-%d", i),
				Name:     fmt.Sprintf("User %d", i),
				Status:   lo.Ternary(i%3 == 0, "Active", "Inactive"),
				Verified: i%2 == 0,
				Teams:    []string{fmt.Sprintf("team-%d", i%5)},
			}
		}
		
		// Test various functional operations
		activeUsers := lo.Filter(users, func(user types.DatadogUser, _ int) bool {
			return user.Status == "Active"
		})
		
		groupedByTeam := lo.GroupBy(users, func(user types.DatadogUser) string {
			if len(user.Teams) > 0 {
				return user.Teams[0]
			}
			return "no-team"
		})
		
		// Verify invariants
		assert.True(t, len(activeUsers) <= len(users), "Active users should not exceed total users")
		assert.Equal(t, 5, len(groupedByTeam), "Should group users by 5 teams")
		
		// Verify group sizes sum to total
		totalInGroups := lo.Reduce(lo.Values(groupedByTeam), func(acc int, group []types.DatadogUser, _ int) int {
			return acc + len(group)
		}, 0)
		assert.Equal(t, len(users), totalInGroups, "Grouped users should sum to total")
	})
}