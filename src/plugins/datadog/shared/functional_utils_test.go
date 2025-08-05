package shared

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"bacon/src/plugins/datadog/types"
	"pgregory.net/rapid"
)

// Test TransformTeamResponse function
func TestTransformTeamResponse(t *testing.T) {
	testCases := []struct {
		name         string
		team         datadogV2.Team
		expectedTeam types.DatadogTeam
	}{
		{
			name: "complete team data",
			team: datadogV2.Team{
				Id: "team-123",
				Attributes: datadogV2.TeamAttributes{
					Name:        "Engineering Team",
					Handle:      "engineering",
					Description: createNullableString("Backend engineering team"),
					Summary:     createNullableString("Team summary"),
					Avatar:      createNullableString("avatar-url"),
					CreatedAt:   timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					ModifiedAt:  timePtr(time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			expectedTeam: types.DatadogTeam{
				ID:          "team-123",
				Name:        "Engineering Team",
				Handle:      "engineering",
				Description: "Backend engineering team",
				Links:       []types.DatadogTeamLink{},
				Metadata: map[string]interface{}{
					"summary": "Team summary",
					"avatar":  "avatar-url",
				},
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "minimal team data",
			team: datadogV2.Team{
				Id: "team-minimal",
				Attributes: datadogV2.TeamAttributes{
					Name:   "Minimal Team",
					Handle: "minimal",
				},
			},
			expectedTeam: types.DatadogTeam{
				ID:          "team-minimal",
				Name:        "Minimal Team",
				Handle:      "minimal",
				Description: "",
				Links:       []types.DatadogTeamLink{},
				Metadata:    map[string]interface{}{},
				CreatedAt:   time.Time{},
				UpdatedAt:   time.Time{},
			},
		},
		{
			name: "empty team data",
			team: datadogV2.Team{
				Id:         "",
				Attributes: datadogV2.TeamAttributes{},
			},
			expectedTeam: types.DatadogTeam{
				ID:          "",
				Name:        "",
				Handle:      "",
				Description: "",
				Links:       []types.DatadogTeamLink{},
				Metadata:    map[string]interface{}{},
				CreatedAt:   time.Time{},
				UpdatedAt:   time.Time{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TransformTeamResponse(tc.team, 0)

			if result.ID != tc.expectedTeam.ID {
				t.Errorf("Expected ID %s, got %s", tc.expectedTeam.ID, result.ID)
			}
			if result.Name != tc.expectedTeam.Name {
				t.Errorf("Expected Name %s, got %s", tc.expectedTeam.Name, result.Name)
			}
			if result.Handle != tc.expectedTeam.Handle {
				t.Errorf("Expected Handle %s, got %s", tc.expectedTeam.Handle, result.Handle)
			}
			if result.Description != tc.expectedTeam.Description {
				t.Errorf("Expected Description %s, got %s", tc.expectedTeam.Description, result.Description)
			}
			if !result.CreatedAt.Equal(tc.expectedTeam.CreatedAt) {
				t.Errorf("Expected CreatedAt %v, got %v", tc.expectedTeam.CreatedAt, result.CreatedAt)
			}
			if !result.UpdatedAt.Equal(tc.expectedTeam.UpdatedAt) {
				t.Errorf("Expected UpdatedAt %v, got %v", tc.expectedTeam.UpdatedAt, result.UpdatedAt)
			}

			// Check metadata equality
			if len(result.Metadata) != len(tc.expectedTeam.Metadata) {
				t.Errorf("Expected metadata length %d, got %d", len(tc.expectedTeam.Metadata), len(result.Metadata))
			}
			for key, expectedValue := range tc.expectedTeam.Metadata {
				if actualValue, exists := result.Metadata[key]; !exists || actualValue != expectedValue {
					t.Errorf("Expected metadata[%s] = %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// Test TransformUserResponse function
func TestTransformUserResponse(t *testing.T) {
	testCases := []struct {
		name         string
		user         datadogV2.User
		expectedUser types.DatadogUser
	}{
		{
			name: "complete user data",
			user: datadogV2.User{
				Id: stringPtr("user-123"),
				Attributes: &datadogV2.UserAttributes{
					Name:       createNullableString("John Doe"),
					Email:      stringPtr("john.doe@example.com"),
					Handle:     stringPtr("johndoe"),
					Status:     stringPtr("Active"),
					Verified:   boolPtr(true),
					Disabled:   boolPtr(false),
					Title:      createNullableString("Senior Engineer"),
					Icon:       stringPtr("user-icon"),
					CreatedAt:  timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					ModifiedAt: timePtr(time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			expectedUser: types.DatadogUser{
				ID:       "user-123",
				Name:     "John Doe",
				Email:    "john.doe@example.com",
				Handle:   "johndoe",
				Teams:    []string{},
				Roles:    []string{},
				Status:   "Active",
				Verified: true,
				Disabled: false,
				Title:    "Senior Engineer",
				Icon:     "user-icon",
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "minimal user data",
			user: datadogV2.User{
				Id: stringPtr("user-minimal"),
				Attributes: &datadogV2.UserAttributes{
					Email: stringPtr("minimal@example.com"),
				},
			},
			expectedUser: types.DatadogUser{
				ID:       "user-minimal",
				Name:     "",
				Email:    "minimal@example.com",
				Handle:   "",
				Teams:    []string{},
				Roles:    []string{},
				Status:   "",
				Verified: false,
				Disabled: false,
				Title:    "",
				Icon:     "",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TransformUserResponse(tc.user, 0)

			if result.ID != tc.expectedUser.ID {
				t.Errorf("Expected ID %s, got %s", tc.expectedUser.ID, result.ID)
			}
			if result.Name != tc.expectedUser.Name {
				t.Errorf("Expected Name %s, got %s", tc.expectedUser.Name, result.Name)
			}
			if result.Email != tc.expectedUser.Email {
				t.Errorf("Expected Email %s, got %s", tc.expectedUser.Email, result.Email)
			}
			if result.Handle != tc.expectedUser.Handle {
				t.Errorf("Expected Handle %s, got %s", tc.expectedUser.Handle, result.Handle)
			}
			if result.Status != tc.expectedUser.Status {
				t.Errorf("Expected Status %s, got %s", tc.expectedUser.Status, result.Status)
			}
			if result.Verified != tc.expectedUser.Verified {
				t.Errorf("Expected Verified %v, got %v", tc.expectedUser.Verified, result.Verified)
			}
			if result.Disabled != tc.expectedUser.Disabled {
				t.Errorf("Expected Disabled %v, got %v", tc.expectedUser.Disabled, result.Disabled)
			}
			if result.Title != tc.expectedUser.Title {
				t.Errorf("Expected Title %s, got %s", tc.expectedUser.Title, result.Title)
			}
			if result.Icon != tc.expectedUser.Icon {
				t.Errorf("Expected Icon %s, got %s", tc.expectedUser.Icon, result.Icon)
			}
		})
	}
}

// Test TransformServiceDefinition function
func TestTransformServiceDefinition(t *testing.T) {
	testCases := []struct {
		name            string
		service         datadogV2.ServiceDefinitionData
		expectedService types.DatadogService
	}{
		{
			name: "basic service data",
			service: datadogV2.ServiceDefinitionData{
				Id:   stringPtr("service-123"),
				Type: stringPtr("service"),
			},
			expectedService: types.DatadogService{
				ID:            "service-123",
				Name:          "service-123",
				Owner:         "",
				Teams:         []string{},
				Tags:          []string{},
				Schema:        "service",
				Description:   "",
				Tier:          "",
				Lifecycle:     "",
				Type:          "service",
				Languages:     []string{},
				Contacts:      []types.DatadogContact{},
				Links:         []types.DatadogServiceLink{},
				Integrations:  map[string]interface{}{},
				Dependencies:  []string{},
			},
		},
		{
			name: "empty service data",
			service: datadogV2.ServiceDefinitionData{
				Id:   stringPtr(""),
				Type: stringPtr(""),
			},
			expectedService: types.DatadogService{
				ID:            "",
				Name:          "",
				Owner:         "",
				Teams:         []string{},
				Tags:          []string{},
				Schema:        "",
				Description:   "",
				Tier:          "",
				Lifecycle:     "",
				Type:          "",
				Languages:     []string{},
				Contacts:      []types.DatadogContact{},
				Links:         []types.DatadogServiceLink{},
				Integrations:  map[string]interface{}{},
				Dependencies:  []string{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TransformServiceDefinition(tc.service, 0)

			if result.ID != tc.expectedService.ID {
				t.Errorf("Expected ID %s, got %s", tc.expectedService.ID, result.ID)
			}
			if result.Name != tc.expectedService.Name {
				t.Errorf("Expected Name %s, got %s", tc.expectedService.Name, result.Name)
			}
			if result.Schema != tc.expectedService.Schema {
				t.Errorf("Expected Schema %s, got %s", tc.expectedService.Schema, result.Schema)
			}
			if result.Type != tc.expectedService.Type {
				t.Errorf("Expected Type %s, got %s", tc.expectedService.Type, result.Type)
			}

			// Verify slice fields are initialized
			if result.Teams == nil {
				t.Errorf("Expected Teams to be initialized")
			}
			if result.Tags == nil {
				t.Errorf("Expected Tags to be initialized")
			}
			if result.Languages == nil {
				t.Errorf("Expected Languages to be initialized")
			}
			if result.Contacts == nil {
				t.Errorf("Expected Contacts to be initialized")
			}
			if result.Links == nil {
				t.Errorf("Expected Links to be initialized")
			}
			if result.Dependencies == nil {
				t.Errorf("Expected Dependencies to be initialized")
			}
			if result.Integrations == nil {
				t.Errorf("Expected Integrations to be initialized")
			}
		})
	}
}

// Test validation functions
func TestIsValidTeam(t *testing.T) {
	testCases := []struct {
		name     string
		team     types.DatadogTeam
		expected bool
	}{
		{
			name: "valid team",
			team: types.DatadogTeam{
				ID:     "team-123",
				Name:   "Engineering",
				Handle: "engineering",
			},
			expected: true,
		},
		{
			name: "missing ID",
			team: types.DatadogTeam{
				ID:     "",
				Name:   "Engineering",
				Handle: "engineering",
			},
			expected: false,
		},
		{
			name: "missing Name",
			team: types.DatadogTeam{
				ID:     "team-123",
				Name:   "",
				Handle: "engineering",
			},
			expected: false,
		},
		{
			name: "missing Handle",
			team: types.DatadogTeam{
				ID:     "team-123",
				Name:   "Engineering",
				Handle: "",
			},
			expected: false,
		},
		{
			name: "all fields empty",
			team: types.DatadogTeam{
				ID:     "",
				Name:   "",
				Handle: "",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidTeam(tc.team)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIsActiveUser(t *testing.T) {
	testCases := []struct {
		name     string
		user     types.DatadogUser
		expected bool
	}{
		{
			name: "active user",
			user: types.DatadogUser{
				Status:   "Active",
				Disabled: false,
			},
			expected: true,
		},
		{
			name: "pending user",
			user: types.DatadogUser{
				Status:   "Pending",
				Disabled: false,
			},
			expected: true,
		},
		{
			name: "disabled user",
			user: types.DatadogUser{
				Status:   "Active",
				Disabled: true,
			},
			expected: false,
		},
		{
			name: "inactive user",
			user: types.DatadogUser{
				Status:   "Inactive",
				Disabled: false,
			},
			expected: false,
		},
		{
			name: "disabled and inactive user",
			user: types.DatadogUser{
				Status:   "Inactive",
				Disabled: true,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsActiveUser(tc.user)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestHasTeamOwnership(t *testing.T) {
	testCases := []struct {
		name     string
		service  types.DatadogService
		expected bool
	}{
		{
			name: "service with teams",
			service: types.DatadogService{
				Teams: []string{"team-1", "team-2"},
			},
			expected: true,
		},
		{
			name: "service with owner",
			service: types.DatadogService{
				Owner: "john.doe",
			},
			expected: true,
		},
		{
			name: "service with contacts",
			service: types.DatadogService{
				Contacts: []types.DatadogContact{
					{Name: "John Doe", Contact: "john.doe@example.com"},
				},
			},
			expected: true,
		},
		{
			name: "service with no ownership",
			service: types.DatadogService{
				Teams:    []string{},
				Owner:    "",
				Contacts: []types.DatadogContact{},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := HasTeamOwnership(tc.service)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// Test enrichment functions
func TestEnrichTeamWithMembers(t *testing.T) {
	team := types.DatadogTeam{
		ID:     "team-123",
		Name:   "Engineering",
		Handle: "engineering",
	}

	users := []types.DatadogUser{
		{ID: "user-1", Name: "John Doe", Teams: []string{"team-123", "team-456"}},
		{ID: "user-2", Name: "Jane Smith", Teams: []string{"team-456"}},
		{ID: "user-3", Name: "Bob Johnson", Teams: []string{"team-123"}},
	}

	result := EnrichTeamWithMembers(team, users)

	expectedMembers := 2 // user-1 and user-3
	if len(result.Members) != expectedMembers {
		t.Errorf("Expected %d members, got %d", expectedMembers, len(result.Members))
	}

	// Verify correct members are included
	memberIDs := make(map[string]bool)
	for _, member := range result.Members {
		memberIDs[member.ID] = true
	}

	if !memberIDs["user-1"] {
		t.Errorf("Expected user-1 to be a member")
	}
	if !memberIDs["user-3"] {
		t.Errorf("Expected user-3 to be a member")
	}
	if memberIDs["user-2"] {
		t.Errorf("Did not expect user-2 to be a member")
	}

	// Verify team data is preserved
	if result.ID != team.ID {
		t.Errorf("Expected team ID to be preserved")
	}
	if result.Name != team.Name {
		t.Errorf("Expected team name to be preserved")
	}
	if result.Handle != team.Handle {
		t.Errorf("Expected team handle to be preserved")
	}
}

func TestEnrichTeamWithServices(t *testing.T) {
	team := types.DatadogTeam{
		ID:     "team-123",
		Name:   "Engineering",
		Handle: "engineering",
	}

	services := []types.DatadogService{
		{
			ID:    "service-1",
			Teams: []string{"team-123"},
		},
		{
			ID:    "service-2",
			Teams: []string{"team-456"},
			Contacts: []types.DatadogContact{
				{Contact: "engineering"},
			},
		},
		{
			ID:    "service-3",
			Teams: []string{"team-789"},
			Contacts: []types.DatadogContact{
				{Contact: "other-team"},
			},
		},
		{
			ID:    "service-4",
			Teams: []string{"team-456"},
			Contacts: []types.DatadogContact{
				{Contact: "Engineering"},
			},
		},
	}

	result := EnrichTeamWithServices(team, services)

	expectedServices := 3 // service-1 (by team), service-2 (by handle), service-4 (by name)
	if len(result.Services) != expectedServices {
		t.Errorf("Expected %d services, got %d", expectedServices, len(result.Services))
	}

	// Verify correct services are included
	serviceIDs := make(map[string]bool)
	for _, service := range result.Services {
		serviceIDs[service.ID] = true
	}

	if !serviceIDs["service-1"] {
		t.Errorf("Expected service-1 to be included (by team)")
	}
	if !serviceIDs["service-2"] {
		t.Errorf("Expected service-2 to be included (by handle)")
	}
	if !serviceIDs["service-4"] {
		t.Errorf("Expected service-4 to be included (by name)")
	}
	if serviceIDs["service-3"] {
		t.Errorf("Did not expect service-3 to be included")
	}
}

// Test helper functions for safe type conversion
func TestSafeStringFromPtr(t *testing.T) {
	testCases := []struct {
		name     string
		ptr      *string
		expected string
	}{
		{
			name:     "valid string pointer",
			ptr:      stringPtr("test-string"),
			expected: "test-string",
		},
		{
			name:     "nil pointer",
			ptr:      nil,
			expected: "",
		},
		{
			name:     "empty string pointer",
			ptr:      stringPtr(""),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := safeStringFromPtr(tc.ptr)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestSafeStringFromNullable(t *testing.T) {
	testCases := []struct {
		name     string
		nullable datadog.NullableString
		expected string
	}{
		{
			name:     "set nullable string",
			nullable: createNullableString("test-string"),
			expected: "test-string",
		},
		{
			name:     "unset nullable string",
			nullable: datadog.NullableString{},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := safeStringFromNullable(tc.nullable)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestSafeBoolFromPtr(t *testing.T) {
	testCases := []struct {
		name     string
		ptr      *bool
		expected bool
	}{
		{
			name:     "true boolean pointer",
			ptr:      boolPtr(true),
			expected: true,
		},
		{
			name:     "false boolean pointer",
			ptr:      boolPtr(false),
			expected: false,
		},
		{
			name:     "nil pointer",
			ptr:      nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := safeBoolFromPtr(tc.ptr)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// Property-based tests for transformations
func TestTransformTeamResponse_Property(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random team data
		teamID := rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(t, "teamID")
		teamName := rapid.StringMatching(`[a-zA-Z0-9 _-]+`).Draw(t, "teamName")
		teamHandle := rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(t, "teamHandle")
		description := rapid.String().Draw(t, "description")

		team := datadogV2.Team{
			Id: teamID,
			Attributes: datadogV2.TeamAttributes{
				Name:        teamName,
				Handle:      teamHandle,
				Description: createNullableString(description),
			},
		}

		result := TransformTeamResponse(team, 0)

		// Verify transformation properties
		if result.ID != teamID {
			t.Errorf("Expected ID to be preserved: %s != %s", result.ID, teamID)
		}
		if result.Name != teamName {
			t.Errorf("Expected Name to be preserved: %s != %s", result.Name, teamName)
		}
		if result.Handle != teamHandle {
			t.Errorf("Expected Handle to be preserved: %s != %s", result.Handle, teamHandle)
		}
		if result.Description != description {
			t.Errorf("Expected Description to be preserved: %s != %s", result.Description, description)
		}

		// Verify collections are always initialized
		if result.Links == nil {
			t.Errorf("Links should always be initialized")
		}
		if result.Metadata == nil {
			t.Errorf("Metadata should always be initialized")
		}
	})
}

func TestTransformUserResponse_Property(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user data
		userID := rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(t, "userID")
		userName := rapid.String().Draw(t, "userName")
		userEmail := rapid.StringMatching(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`).Draw(t, "userEmail")
		userHandle := rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(t, "userHandle")
		verified := rapid.Bool().Draw(t, "verified")
		disabled := rapid.Bool().Draw(t, "disabled")

		user := datadogV2.User{
			Id: &userID,
			Attributes: &datadogV2.UserAttributes{
				Name:     createNullableString(userName),
				Email:    &userEmail,
				Handle:   &userHandle,
				Verified: &verified,
				Disabled: &disabled,
			},
		}

		result := TransformUserResponse(user, 0)

		// Verify transformation properties
		if result.ID != userID {
			t.Errorf("Expected ID to be preserved: %s != %s", result.ID, userID)
		}
		if result.Name != userName {
			t.Errorf("Expected Name to be preserved: %s != %s", result.Name, userName)
		}
		if result.Email != userEmail {
			t.Errorf("Expected Email to be preserved: %s != %s", result.Email, userEmail)
		}
		if result.Handle != userHandle {
			t.Errorf("Expected Handle to be preserved: %s != %s", result.Handle, userHandle)
		}
		if result.Verified != verified {
			t.Errorf("Expected Verified to be preserved: %v != %v", result.Verified, verified)
		}
		if result.Disabled != disabled {
			t.Errorf("Expected Disabled to be preserved: %v != %v", result.Disabled, disabled)
		}

		// Verify collections are always initialized
		if result.Teams == nil {
			t.Errorf("Teams should always be initialized")
		}
		if result.Roles == nil {
			t.Errorf("Roles should always be initialized")
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkTransformTeamResponse(b *testing.B) {
	team := datadogV2.Team{
		Id: "team-123",
		Attributes: datadogV2.TeamAttributes{
			Name:        "Engineering Team",
			Handle:      "engineering",
			Description: createNullableString("Backend engineering team"),
			Summary:     createNullableString("Team summary"),
			Avatar:      createNullableString("avatar-url"),
			CreatedAt:   timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
			ModifiedAt:  timePtr(time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TransformTeamResponse(team, 0)
	}
}

func BenchmarkTransformUserResponse(b *testing.B) {
	user := datadogV2.User{
		Id: stringPtr("user-123"),
		Attributes: &datadogV2.UserAttributes{
			Name:       createNullableString("John Doe"),
			Email:      stringPtr("john.doe@example.com"),
			Handle:     stringPtr("johndoe"),
			Status:     stringPtr("Active"),
			Verified:   boolPtr(true),
			Disabled:   boolPtr(false),
			Title:      createNullableString("Senior Engineer"),
			Icon:       stringPtr("user-icon"),
			CreatedAt:  timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
			ModifiedAt: timePtr(time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TransformUserResponse(user, 0)
	}
}

func BenchmarkIsValidTeam(b *testing.B) {
	team := types.DatadogTeam{
		ID:     "team-123",
		Name:   "Engineering",
		Handle: "engineering",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsValidTeam(team)
	}
}

func BenchmarkEnrichTeamWithMembers(b *testing.B) {
	team := types.DatadogTeam{
		ID:     "team-123",
		Name:   "Engineering",
		Handle: "engineering",
	}

	users := make([]types.DatadogUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = types.DatadogUser{
			ID:    fmt.Sprintf("user-%d", i),
			Name:  fmt.Sprintf("User %d", i),
			Teams: []string{"team-123", "team-456"},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EnrichTeamWithMembers(team, users)
	}
}

// Edge case tests
func TestTransformations_EdgeCases(t *testing.T) {
	t.Run("nil attributes", func(t *testing.T) {
		team := datadogV2.Team{
			Id:         "team-123",
			Attributes: datadogV2.TeamAttributes{},
		}

		// Should not panic
		result := TransformTeamResponse(team, 0)
		if result.ID != "team-123" {
			t.Errorf("Expected ID to be preserved even with nil attributes")
		}
	})

	t.Run("unicode data", func(t *testing.T) {
		team := datadogV2.Team{
			Id: "team-🚀",
			Attributes: datadogV2.TeamAttributes{
				Name:        "Engineering 团队",
				Handle:      "engineering-🔧",
				Description: createNullableString("Backend engineering team with émojis 🎉"),
			},
		}

		result := TransformTeamResponse(team, 0)
		if !utf8.ValidString(result.Name) {
			t.Errorf("Name should be valid UTF-8")
		}
		if !utf8.ValidString(result.Handle) {
			t.Errorf("Handle should be valid UTF-8")
		}
		if !utf8.ValidString(result.Description) {
			t.Errorf("Description should be valid UTF-8")
		}
	})

	t.Run("very long strings", func(t *testing.T) {
		longString := strings.Repeat("a", 10000)
		team := datadogV2.Team{
			Id: "team-123",
			Attributes: datadogV2.TeamAttributes{
				Name:        longString,
				Handle:      longString,
				Description: createNullableString(longString),
			},
		}

		result := TransformTeamResponse(team, 0)
		if len(result.Name) != 10000 {
			t.Errorf("Expected long name to be preserved")
		}
	})
}

// Helper functions for test fixtures
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func createNullableString(s string) datadog.NullableString {
	ns := datadog.NewNullableString(&s)
	return *ns
}