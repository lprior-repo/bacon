// Package types provides immutable data structures for Datadog API v2 integration.
package types

import (
	"time"
)

// DatadogTeam represents a team from the Datadog Teams API v2
type DatadogTeam struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Handle      string                 `json:"handle"`
	Description string                 `json:"description"`
	Members     []DatadogUser          `json:"members"`
	Services    []DatadogService       `json:"services"`
	Links       []DatadogTeamLink      `json:"links"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DatadogUser represents a user from the Datadog Users API v2
type DatadogUser struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Handle   string    `json:"handle"`
	Teams    []string  `json:"teams"`
	Roles    []string  `json:"roles"`
	Status   string    `json:"status"`
	Verified bool      `json:"verified"`
	Disabled bool      `json:"disabled"`
	Title    string    `json:"title"`
	Icon     string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DatadogService represents a service from the Datadog Service Catalog API v2
type DatadogService struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Owner         string                 `json:"owner"`
	Teams         []string               `json:"teams"`
	Tags          []string               `json:"tags"`
	Schema        string                 `json:"schema_version"`
	Description   string                 `json:"description"`
	Tier          string                 `json:"tier"`
	Lifecycle     string                 `json:"lifecycle"`
	Type          string                 `json:"type"`
	Languages     []string               `json:"languages"`
	Contacts      []DatadogContact       `json:"contacts"`
	Links         []DatadogServiceLink   `json:"links"`
	Integrations  map[string]interface{} `json:"integrations"`
	Dependencies  []string               `json:"dependencies"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// DatadogOrganization represents organization data from Datadog Organizations API v2
type DatadogOrganization struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
	Users       []DatadogUser          `json:"users"`
	Teams       []DatadogTeam          `json:"teams"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DatadogTeamLink represents team-related links
type DatadogTeamLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

// DatadogContact represents service contact information
type DatadogContact struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Contact string `json:"contact"`
}

// DatadogServiceLink represents service-related links
type DatadogServiceLink struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// DatadogTeamSnapshot represents a complete snapshot of all team-related data
type DatadogTeamSnapshot struct {
	Teams         []DatadogTeam         `json:"teams"`
	Users         []DatadogUser         `json:"users"`
	Services      []DatadogService      `json:"services"`
	Organizations []DatadogOrganization `json:"organizations"`
	Timestamp     time.Time             `json:"timestamp"`
	TotalTeams    int                   `json:"total_teams"`
	TotalUsers    int                   `json:"total_users"`
	TotalServices int                   `json:"total_services"`
}

// ScraperEvent represents the input event for individual scraper Lambda functions
type ScraperEvent struct {
	FilterKeyword    string                 `json:"filter_keyword,omitempty"`
	PageSize         int                    `json:"page_size,omitempty"`
	SchemaVersion    string                 `json:"schema_version,omitempty"`
	IncludeInactive  bool                   `json:"include_inactive,omitempty"`
	TeamID           string                 `json:"team_id,omitempty"`
	OrganizationID   string                 `json:"organization_id,omitempty"`
	ExtraParameters  map[string]interface{} `json:"extra_parameters,omitempty"`
}

// ScraperResponse represents the output response from individual scraper Lambda functions
type ScraperResponse struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	Count       int                    `json:"count"`
	Timestamp   string                 `json:"timestamp"`
	ExecutionID string                 `json:"execution_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// OrchestrationEvent represents the input event for the orchestrator Lambda
type OrchestrationEvent struct {
	TriggerType    string                 `json:"trigger_type"` // "scheduled", "manual", "event"
	TargetEndpoints []string               `json:"target_endpoints,omitempty"` // specific endpoints to scrape
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
}

// OrchestrationResponse represents the output from the orchestrator Lambda
type OrchestrationResponse struct {
	ExecutionARN  string                 `json:"execution_arn"`
	Status        string                 `json:"status"`
	RequestID     string                 `json:"request_id"`
	TriggeredJobs []string               `json:"triggered_jobs"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}