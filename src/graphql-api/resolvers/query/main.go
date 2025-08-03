// Package main implements GraphQL query resolvers for the resource ownership system.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
)

type AppSyncEvent struct {
	Info      RequestInfo                `json:"info"`
	Arguments map[string]interface{}     `json:"arguments"`
	Source    map[string]interface{}     `json:"source"`
	Request   map[string]interface{}     `json:"request"`
}

type RequestInfo struct {
	FieldName      string `json:"fieldName"`
	ParentTypeName string `json:"parentTypeName"`
}

type Resource struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Description  string         `json:"description"`
	Relationships []Relationship `json:"relationships"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

type Relationship struct {
	ID              string  `json:"id"`
	From            string  `json:"from"`
	To              string  `json:"to"`
	Type            string  `json:"type"`
	Confidence      float64 `json:"confidence"`
	ConfidenceLevel string  `json:"confidenceLevel"`
	Source          string  `json:"source"`
	HasConflict     bool    `json:"hasConflict"`
	LastValidated   string  `json:"lastValidated"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type OwnershipStats struct {
	TotalResources      int                   `json:"totalResources"`
	OwnedResources      int                   `json:"ownedResources"`
	UnownedResources    int                   `json:"unownedResources"`
	CoveragePercentage  float64               `json:"coveragePercentage"`
	ByResourceType      []ResourceTypeStats   `json:"byResourceType"`
	ByTeam              []TeamStats           `json:"byTeam"`
}

type ResourceTypeStats struct {
	Type     string  `json:"type"`
	Total    int     `json:"total"`
	Owned    int     `json:"owned"`
	Coverage float64 `json:"coverage"`
}

type TeamStats struct {
	Team              string  `json:"team"`
	ResourceCount     int     `json:"resourceCount"`
	AverageConfidence float64 `json:"averageConfidence"`
}

type ConfidenceStats struct {
	High                   int                      `json:"high"`
	Medium                 int                      `json:"medium"`
	Low                    int                      `json:"low"`
	VeryLow                int                      `json:"veryLow"`
	AverageConfidence      float64                  `json:"averageConfidence"`
	DistributionBySource   []SourceConfidenceStats  `json:"distributionBySource"`
}

type SourceConfidenceStats struct {
	Source            string  `json:"source"`
	Count             int     `json:"count"`
	AverageConfidence float64 `json:"averageConfidence"`
}

func HandleRequest(ctx context.Context, event AppSyncEvent) (interface{}, error) {
	ctx, seg := xray.BeginSegment(ctx, "appsync-query-resolver")
	defer seg.Close(nil)

	log.Printf("Handling AppSync query: %s", event.Info.FieldName)
	_ = seg.AddAnnotation("field_name", event.Info.FieldName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	_ = cfg // Use for Neptune client in production

	switch event.Info.FieldName {
	case "getResource":
		return handleGetResource(ctx, event.Arguments)
	case "getResourcesByConfidence":
		return handleGetResourcesByConfidence(ctx, event.Arguments)
	case "getConflictedRelationships":
		return handleGetConflictedRelationships(ctx, event.Arguments)
	case "getRelationshipsBySource":
		return handleGetRelationshipsBySource(ctx, event.Arguments)
	case "searchResources":
		return handleSearchResources(ctx, event.Arguments)
	case "searchResourcesByOwner":
		return handleSearchResourcesByOwner(ctx, event.Arguments)
	case "getOwnershipCoverage":
		return handleGetOwnershipCoverage(ctx, event.Arguments)
	case "getConfidenceDistribution":
		return handleGetConfidenceDistribution(ctx, event.Arguments)
	default:
		return nil, fmt.Errorf("unknown field: %s", event.Info.FieldName)
	}
}

func handleGetResource(ctx context.Context, args map[string]interface{}) (*Resource, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-resource")
	defer seg.Close(nil)

	resourceID := args["id"].(string)
	_ = seg.AddAnnotation("resource_id", resourceID)

	// Mock resource for POC - in production would query Neptune
	resource := &Resource{
		ID:          resourceID,
		Name:        "example-api-service",
		Type:        "KUBERNETES_SERVICE",
		Description: "Main API service for the application",
		Relationships: []Relationship{
			{
				ID:              "rel-1",
				From:            "backend-team",
				To:              resourceID,
				Type:            "OWNS",
				Confidence:      0.85,
				ConfidenceLevel: "HIGH",
				Source:          "openshift-metadata",
				HasConflict:     false,
				LastValidated:   "2024-01-15T10:00:00Z",
				CreatedAt:       "2024-01-01T00:00:00Z",
				UpdatedAt:       "2024-01-15T10:00:00Z",
			},
		},
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-15T10:00:00Z",
	}

	return resource, nil
}

func handleGetResourcesByConfidence(ctx context.Context, args map[string]interface{}) ([]Resource, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-resources-by-confidence")
	defer seg.Close(nil)

	minConfidence := args["minConfidence"].(float64)
	_ = seg.AddAnnotation("min_confidence", minConfidence)

	// Mock filtered resources
	resources := []Resource{
		{
			ID:          "res-1",
			Name:        "high-confidence-service",
			Type:        "KUBERNETES_SERVICE",
			Description: "Service with high confidence ownership",
			Relationships: []Relationship{
				{
					ID:              "rel-1",
					Confidence:      0.92,
					ConfidenceLevel: "VERY_HIGH",
					Source:          "aws-tags",
					HasConflict:     false,
				},
			},
		},
	}

	return resources, nil
}

func handleGetConflictedRelationships(ctx context.Context, _ map[string]interface{}) ([]Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-conflicted-relationships")
	defer seg.Close(nil)

	// Mock conflicted relationships
	relationships := []Relationship{
		{
			ID:              "rel-conflict-1",
			From:            "team-a",
			To:              "disputed-service",
			Type:            "OWNS",
			Confidence:      0.75,
			ConfidenceLevel: "DISPUTED",
			Source:          "github-codeowners",
			HasConflict:     true,
			LastValidated:   "2024-01-15T10:00:00Z",
		},
	}

	return relationships, nil
}

func handleGetRelationshipsBySource(ctx context.Context, args map[string]interface{}) ([]Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-relationships-by-source")
	defer seg.Close(nil)

	source := args["source"].(string)
	_ = seg.AddAnnotation("source", source)

	// Mock source-filtered relationships
	relationships := []Relationship{
		{
			ID:              "rel-1",
			From:            "platform-team",
			To:              "infrastructure-service",
			Type:            "OWNS",
			Confidence:      0.88,
			ConfidenceLevel: "HIGH",
			Source:          source,
			HasConflict:     false,
		},
	}

	return relationships, nil
}

func handleSearchResources(ctx context.Context, args map[string]interface{}) ([]Resource, error) {
	_, seg := xray.BeginSubsegment(ctx, "search-resources")
	defer seg.Close(nil)

	searchText := args["text"].(string)
	_ = seg.AddAnnotation("search_text", searchText)

	// Mock search results
	resources := []Resource{
		{
			ID:          "search-result-1",
			Name:        fmt.Sprintf("service-matching-%s", strings.ToLower(searchText)),
			Type:        "APPLICATION",
			Description: fmt.Sprintf("Service that matches search term: %s", searchText),
		},
	}

	return resources, nil
}

func handleSearchResourcesByOwner(ctx context.Context, args map[string]interface{}) ([]Resource, error) {
	_, seg := xray.BeginSubsegment(ctx, "search-resources-by-owner")
	defer seg.Close(nil)

	owner := args["owner"].(string)
	_ = seg.AddAnnotation("owner", owner)

	// Mock owner-based search
	resources := []Resource{
		{
			ID:          "owner-resource-1",
			Name:        fmt.Sprintf("%s-owned-service", owner),
			Type:        "KUBERNETES_SERVICE",
			Description: fmt.Sprintf("Service owned by %s", owner),
		},
	}

	return resources, nil
}

func handleGetOwnershipCoverage(ctx context.Context, _ map[string]interface{}) (*OwnershipStats, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-ownership-coverage")
	defer seg.Close(nil)

	// Mock ownership statistics
	stats := &OwnershipStats{
		TotalResources:     150,
		OwnedResources:     127,
		UnownedResources:   23,
		CoveragePercentage: 84.7,
		ByResourceType: []ResourceTypeStats{
			{Type: "KUBERNETES_SERVICE", Total: 45, Owned: 42, Coverage: 93.3},
			{Type: "AWS_RESOURCE", Total: 65, Owned: 58, Coverage: 89.2},
			{Type: "GITHUB_REPOSITORY", Total: 40, Owned: 27, Coverage: 67.5},
		},
		ByTeam: []TeamStats{
			{Team: "backend-team", ResourceCount: 35, AverageConfidence: 0.87},
			{Team: "platform-team", ResourceCount: 42, AverageConfidence: 0.91},
			{Team: "frontend-team", ResourceCount: 28, AverageConfidence: 0.79},
		},
	}

	return stats, nil
}

func handleGetConfidenceDistribution(ctx context.Context, _ map[string]interface{}) (*ConfidenceStats, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-confidence-distribution")
	defer seg.Close(nil)

	// Mock confidence statistics
	stats := &ConfidenceStats{
		High:              85,
		Medium:            42,
		Low:               18,
		VeryLow:           5,
		AverageConfidence: 0.82,
		DistributionBySource: []SourceConfidenceStats{
			{Source: "aws-tags", Count: 45, AverageConfidence: 0.91},
			{Source: "openshift-metadata", Count: 38, AverageConfidence: 0.89},
			{Source: "github-codeowners", Count: 52, AverageConfidence: 0.78},
			{Source: "github-activity", Count: 15, AverageConfidence: 0.64},
		},
	}

	return stats, nil
}

func main() {
	lambda.Start(HandleRequest)
}