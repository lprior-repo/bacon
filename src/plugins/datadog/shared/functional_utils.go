// Package shared provides pure functional utilities for Datadog API v2 data transformations.
package shared

import (
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/samber/lo"
	"bacon/src/plugins/datadog/types"
)

// Pure transformation functions using samber/lo for Teams API

// TransformTeamResponse converts a Datadog API team response to our internal type
// Pure function with no side effects
func TransformTeamResponse(team datadogV2.Team, _ int) types.DatadogTeam {
	return types.DatadogTeam{
		ID:          safeStringFromDirectValue(team.Id),
		Name:        safeStringFromDirectValue(team.Attributes.Name),
		Handle:      safeStringFromDirectValue(team.Attributes.Handle),
		Description: safeStringFromNullable(team.Attributes.Description),
		Links:       lo.Map(getTeamLinks(team), TransformTeamLink),
		Metadata:    extractTeamMetadata(team),
		CreatedAt:   parseDatadogTime(team.Attributes.CreatedAt),
		UpdatedAt:   parseDatadogTime(team.Attributes.ModifiedAt),
	}
}

// TransformTeamLink converts team link data
// Pure function for link transformation
func TransformTeamLink(link datadogV2.TeamLink, _ int) types.DatadogTeamLink {
	return types.DatadogTeamLink{
		Label: safeStringFromDirectValue(link.Attributes.Label),
		URL:   safeStringFromDirectValue(link.Attributes.Url),
		Type:  string(link.Type),
	}
}

// Pure transformation functions using samber/lo for Users API

// TransformUserResponse converts a Datadog API user response to our internal type
// Pure function with no side effects
func TransformUserResponse(user datadogV2.User, _ int) types.DatadogUser {
	return types.DatadogUser{
		ID:       safeStringFromPtr(user.Id),
		Name:     safeStringFromNullable(user.Attributes.Name),
		Email:    safeStringFromPtr(user.Attributes.Email),
		Handle:   safeStringFromPtr(user.Attributes.Handle),
		Teams:    lo.Map(getUserTeams(user), extractTeamID),
		Roles:    lo.Map(getUserRoles(user), extractRoleName),
		Status:   safeStringFromPtr(user.Attributes.Status),
		Verified: safeBoolFromPtr(user.Attributes.Verified),
		Disabled: safeBoolFromPtr(user.Attributes.Disabled),
		Title:    safeStringFromNullable(user.Attributes.Title),
		Icon:     safeStringFromPtr(user.Attributes.Icon),
		CreatedAt: parseDatadogTime(user.Attributes.CreatedAt),
		UpdatedAt: parseDatadogTime(user.Attributes.ModifiedAt),
	}
}

// Pure transformation functions using samber/lo for Services API

// TransformServiceDefinition converts a Datadog service definition to our internal type
// Pure function with no side effects
func TransformServiceDefinition(service datadogV2.ServiceDefinitionData, _ int) types.DatadogService {
	return types.DatadogService{
		ID:           safeStringFromPtr(service.Id),
		Name:         safeStringFromPtr(service.Id), // Simplified - using ID as name for now
		Owner:        "", // Will be filled by actual implementation
		Teams:        []string{}, // Will be filled by actual implementation
		Tags:         []string{}, // Will be filled by actual implementation
		Schema:       safeStringFromPtr(service.Type),
		Description:  "", // Will be filled by actual implementation
		Tier:         "", // Will be filled by actual implementation
		Lifecycle:    "", // Will be filled by actual implementation
		Type:         safeStringFromPtr(service.Type),
		Languages:    []string{}, // Will be filled by actual implementation
		Contacts:     []types.DatadogContact{}, // Will be filled by actual implementation
		Links:        []types.DatadogServiceLink{}, // Will be filled by actual implementation
		Integrations: make(map[string]interface{}), // Will be filled by actual implementation
		Dependencies: []string{}, // Will be filled by actual implementation
		CreatedAt:    time.Now(), // Services don't have created_at in API
		UpdatedAt:    time.Now(),
	}
}

// TransformContact converts service contact information
// Pure function for contact transformation
func TransformContact(contact datadogV2.ServiceDefinitionV2Dot2Contact, _ int) types.DatadogContact {
	return types.DatadogContact{
		Name:    safeStringFromPtr(contact.Name),
		Type:    safeStringFromDirectValue(contact.Type),
		Contact: safeStringFromDirectValue(contact.Contact),
	}
}

// TransformServiceLink converts service link information
// Pure function for service link transformation
func TransformServiceLink(link datadogV2.ServiceDefinitionV2Dot2Link, _ int) types.DatadogServiceLink {
	return types.DatadogServiceLink{
		Name: safeStringFromDirectValue(link.Name),
		Type: safeStringFromDirectValue(link.Type),
		URL:  safeStringFromDirectValue(link.Url),
	}
}

// Pure validation functions using samber/lo

// IsValidTeam validates team data using functional composition
// Pure function with no side effects
func IsValidTeam(team types.DatadogTeam) bool {
	return lo.EveryBy([]string{team.ID, team.Name, team.Handle}, func(field string) bool {
		return field != ""
	})
}

// IsActiveUser validates if a user is active and verified
// Pure function for user validation
func IsActiveUser(user types.DatadogUser) bool {
	conditions := []bool{
		user.Status == "Active" || user.Status == "Pending",
		!user.Disabled,
	}
	return lo.EveryBy(conditions, func(condition bool) bool {
		return condition
	})
}

// HasTeamOwnership checks if a service has team ownership information
// Pure function for service validation
func HasTeamOwnership(service types.DatadogService) bool {
	return lo.SomeBy([]bool{
		len(service.Teams) > 0,
		service.Owner != "",
		len(service.Contacts) > 0,
	}, func(condition bool) bool {
		return condition
	})
}

// Pure enrichment functions using samber/lo

// EnrichTeamWithMembers adds user members to a team using functional composition
// Pure function that creates a new team with enriched data
func EnrichTeamWithMembers(team types.DatadogTeam, users []types.DatadogUser) types.DatadogTeam {
	teamMembers := lo.Filter(users, func(user types.DatadogUser, _ int) bool {
		return lo.Contains(user.Teams, team.ID)
	})

	return types.DatadogTeam{
		ID:          team.ID,
		Name:        team.Name,
		Handle:      team.Handle,
		Description: team.Description,
		Members:     teamMembers,
		Services:    team.Services,
		Links:       team.Links,
		Metadata:    team.Metadata,
		CreatedAt:   team.CreatedAt,
		UpdatedAt:   team.UpdatedAt,
	}
}

// EnrichTeamWithServices adds services to a team using functional composition
// Pure function that creates a new team with service information
func EnrichTeamWithServices(team types.DatadogTeam, services []types.DatadogService) types.DatadogTeam {
	teamServices := lo.Filter(services, func(service types.DatadogService, _ int) bool {
		return lo.Contains(service.Teams, team.ID) || 
			   lo.SomeBy(service.Contacts, func(contact types.DatadogContact) bool {
				   return contact.Contact == team.Handle || contact.Contact == team.Name
			   })
	})

	return types.DatadogTeam{
		ID:          team.ID,
		Name:        team.Name,
		Handle:      team.Handle,
		Description: team.Description,
		Members:     team.Members,
		Services:    teamServices,
		Links:       team.Links,
		Metadata:    team.Metadata,
		CreatedAt:   team.CreatedAt,
		UpdatedAt:   team.UpdatedAt,
	}
}

// Pure helper functions for data extraction

// extractTeamID extracts team ID from team relationship data
func extractTeamID(teamData datadogV2.RelationshipToTeamData, _ int) string {
	return safeStringFromPtr(teamData.Id)
}

// extractRoleName extracts role name from role relationship data
func extractRoleName(roleData datadogV2.RelationshipToRoleData, _ int) string {
	return safeStringFromPtr(roleData.Id)
}

// extractTeamFromTag extracts team name from service tags
func extractTeamFromTag(tag string, _ int) string {
	// Remove "team:" prefix if present using strings package
	if len(tag) > 5 && tag[:5] == "team:" {
		return tag[5:]
	}
	return tag
}

// extractOwnerFromContacts finds the owner from service contacts
func extractOwnerFromContacts(contacts []datadogV2.ServiceDefinitionV2Dot2Contact) string {
	// Simplified implementation - will be enhanced later
	return ""
}

// Helper functions for safe data access

// getTeamLinks safely extracts team links from team data
func getTeamLinks(team datadogV2.Team) []datadogV2.TeamLink {
	if team.Relationships == nil || team.Relationships.TeamLinks == nil {
		return []datadogV2.TeamLink{}
	}
	// Simplified implementation - return empty for now
	return []datadogV2.TeamLink{}
}

// getUserTeams safely extracts user teams from user data
func getUserTeams(user datadogV2.User) []datadogV2.RelationshipToTeamData {
	// Simplified implementation - return empty for now
	return []datadogV2.RelationshipToTeamData{}
}

// getUserRoles safely extracts user roles from user data
func getUserRoles(user datadogV2.User) []datadogV2.RelationshipToRoleData {
	// Simplified implementation - return empty for now
	return []datadogV2.RelationshipToRoleData{}
}

// getServiceTeamTags safely extracts team tags from service
func getServiceTeamTags(service datadogV2.ServiceDefinitionData) []string {
	// Simplified implementation - will be enhanced later
	return []string{}
}

// getServiceLinks safely extracts service links
func getServiceLinks(service datadogV2.ServiceDefinitionData) []datadogV2.ServiceDefinitionV2Dot2Link {
	// Simplified implementation - will be enhanced later
	return []datadogV2.ServiceDefinitionV2Dot2Link{}
}

// extractDependencies extracts service dependencies
func extractDependencies(service datadogV2.ServiceDefinitionData) []string {
	// Simplified implementation - will be enhanced later
	return []string{}
}

// extractTeamMetadata extracts metadata from team attributes
func extractTeamMetadata(team datadogV2.Team) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	if team.Attributes.Summary.IsSet() {
		metadata["summary"] = safeStringFromNullable(team.Attributes.Summary)
	}
	
	if team.Attributes.Avatar.IsSet() {
		metadata["avatar"] = safeStringFromNullable(team.Attributes.Avatar)
	}
	
	if team.Attributes.Banner.IsSet() {
		if bannerValue := team.Attributes.Banner.Get(); bannerValue != nil {
			metadata["banner"] = *bannerValue
		}
	}
	
	return metadata
}

// parseDatadogTime safely parses Datadog timestamp
func parseDatadogTime(timePtr *time.Time) time.Time {
	return lo.FromPtrOr(timePtr, time.Time{})
}

// Helper functions for safe type conversion from Datadog API v2 nullable types

// safeStringFromPtr safely extracts string from pointer
func safeStringFromPtr(ptr *string) string {
	return lo.FromPtrOr(ptr, "")
}

// safeStringFromDirectValue safely extracts string from direct string value or empty if empty
func safeStringFromDirectValue(value string) string {
	return value
}

// safeStringFromNullable safely extracts string from Datadog NullableString
func safeStringFromNullable(nullable datadog.NullableString) string {
	if nullable.IsSet() {
		return lo.FromPtrOr(nullable.Get(), "")
	}
	return ""
}

// safeBoolFromPtr safely extracts bool from pointer
func safeBoolFromPtr(ptr *bool) bool {
	return lo.FromPtrOr(ptr, false)
}

// safeBoolFromNullable safely extracts bool from Datadog NullableBool
func safeBoolFromNullable(nullable datadog.NullableBool) bool {
	if nullable.IsSet() {
		return lo.FromPtrOr(nullable.Get(), false)
	}
	return false
}