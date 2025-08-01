package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/xray"
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

type CreateRelationshipInput struct {
	FromUserId   string                 `json:"fromUserId"`
	ToResourceId string                 `json:"toResourceId"`
	Type         string                 `json:"type"`
	Confidence   *float64               `json:"confidence"`
	Source       *string                `json:"source"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func HandleRequest(ctx context.Context, event AppSyncEvent) (interface{}, error) {
	ctx, seg := xray.BeginSegment(ctx, "appsync-mutation-resolver")
	defer seg.Close(nil)

	log.Printf("Handling AppSync mutation: %s", event.Info.FieldName)
	seg.AddAnnotation("field_name", event.Info.FieldName)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	_ = cfg // Use for Neptune client in production

	switch event.Info.FieldName {
	case "createRelationship":
		return handleCreateRelationship(ctx, event.Arguments)
	case "updateRelationshipConfidence":
		return handleUpdateRelationshipConfidence(ctx, event.Arguments)
	case "resolveConflict":
		return handleResolveConflict(ctx, event.Arguments)
	case "approveRelationships":
		return handleApproveRelationships(ctx, event.Arguments)
	case "rejectRelationships":
		return handleRejectRelationships(ctx, event.Arguments)
	default:
		return nil, fmt.Errorf("unknown field: %s", event.Info.FieldName)
	}
}

func handleCreateRelationship(ctx context.Context, args map[string]interface{}) (*Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "create-relationship")
	defer seg.Close(nil)

	inputRaw := args["input"].(map[string]interface{})
	
	// Parse input
	fromUserId := inputRaw["fromUserId"].(string)
	toResourceId := inputRaw["toResourceId"].(string)
	relType := inputRaw["type"].(string)
	
	confidence := 0.5 // Default confidence
	if inputRaw["confidence"] != nil {
		confidence = inputRaw["confidence"].(float64)
	}
	
	source := "manual"
	if inputRaw["source"] != nil {
		source = inputRaw["source"].(string)
	}

	seg.AddAnnotation("from_user_id", fromUserId)
	seg.AddAnnotation("to_resource_id", toResourceId)
	seg.AddAnnotation("relationship_type", relType)

	// Mock relationship creation - in production would create in Neptune
	now := time.Now().Format(time.RFC3339)
	relationship := &Relationship{
		ID:              fmt.Sprintf("rel-%d", time.Now().Unix()),
		From:            fromUserId,
		To:              toResourceId,
		Type:            relType,
		Confidence:      confidence,
		ConfidenceLevel: calculateConfidenceLevel(confidence),
		Source:          source,
		HasConflict:     false,
		LastValidated:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	log.Printf("Created relationship: %s -> %s (%s) with confidence %f", 
		fromUserId, toResourceId, relType, confidence)

	return relationship, nil
}

func handleUpdateRelationshipConfidence(ctx context.Context, args map[string]interface{}) (*Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "update-relationship-confidence")
	defer seg.Close(nil)

	relationshipId := args["id"].(string)
	newConfidence := args["confidence"].(float64)

	seg.AddAnnotation("relationship_id", relationshipId)
	seg.AddAnnotation("new_confidence", newConfidence)

	// Mock confidence update - in production would update Neptune
	now := time.Now().Format(time.RFC3339)
	relationship := &Relationship{
		ID:              relationshipId,
		From:            "updated-user",
		To:              "updated-resource",
		Type:            "OWNS",
		Confidence:      newConfidence,
		ConfidenceLevel: calculateConfidenceLevel(newConfidence),
		Source:          "manual-update",
		HasConflict:     false,
		LastValidated:   now,
		CreatedAt:       "2024-01-01T00:00:00Z",
		UpdatedAt:       now,
	}

	log.Printf("Updated relationship %s confidence to %f", relationshipId, newConfidence)

	return relationship, nil
}

func handleResolveConflict(ctx context.Context, args map[string]interface{}) (*Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "resolve-conflict")
	defer seg.Close(nil)

	conflictId := args["id"].(string)
	winnerId := args["winnerId"].(string)

	seg.AddAnnotation("conflict_id", conflictId)
	seg.AddAnnotation("winner_id", winnerId)

	// Mock conflict resolution - in production would update Neptune
	now := time.Now().Format(time.RFC3339)
	relationship := &Relationship{
		ID:              winnerId,
		From:            "resolved-user",
		To:              "disputed-resource",
		Type:            "OWNS",
		Confidence:      0.95, // High confidence after manual resolution
		ConfidenceLevel: "VERY_HIGH",
		Source:          "manual-resolution",
		HasConflict:     false, // Conflict resolved
		LastValidated:   now,
		CreatedAt:       "2024-01-01T00:00:00Z",
		UpdatedAt:       now,
	}

	log.Printf("Resolved conflict %s, winner: %s", conflictId, winnerId)

	return relationship, nil
}

func handleApproveRelationships(ctx context.Context, args map[string]interface{}) ([]Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "approve-relationships")
	defer seg.Close(nil)

	idsRaw := args["ids"].([]interface{})
	var ids []string
	for _, id := range idsRaw {
		ids = append(ids, id.(string))
	}

	seg.AddAnnotation("relationship_count", len(ids))

	// Mock bulk approval - in production would update Neptune
	var relationships []Relationship
	now := time.Now().Format(time.RFC3339)

	for _, id := range ids {
		rel := Relationship{
			ID:              id,
			From:            "approved-user",
			To:              "approved-resource",
			Type:            "OWNS",
			Confidence:      0.90, // High confidence after approval
			ConfidenceLevel: "VERY_HIGH",
			Source:          "manual-approval",
			HasConflict:     false,
			LastValidated:   now,
			CreatedAt:       "2024-01-01T00:00:00Z",
			UpdatedAt:       now,
		}
		relationships = append(relationships, rel)
	}

	log.Printf("Approved %d relationships", len(relationships))

	return relationships, nil
}

func handleRejectRelationships(ctx context.Context, args map[string]interface{}) ([]Relationship, error) {
	_, seg := xray.BeginSubsegment(ctx, "reject-relationships")
	defer seg.Close(nil)

	idsRaw := args["ids"].([]interface{})
	var ids []string
	for _, id := range idsRaw {
		ids = append(ids, id.(string))
	}

	seg.AddAnnotation("relationship_count", len(ids))

	// Mock bulk rejection - in production would delete from Neptune
	log.Printf("Rejected %d relationships: %v", len(ids), ids)

	// Return empty array as relationships were deleted
	return []Relationship{}, nil
}

func calculateConfidenceLevel(confidence float64) string {
	switch {
	case confidence >= 0.9:
		return "VERY_HIGH"
	case confidence >= 0.8:
		return "HIGH"
	case confidence >= 0.6:
		return "MEDIUM"
	case confidence >= 0.4:
		return "LOW"
	default:
		return "VERY_LOW"
	}
}

func main() {
	lambda.Start(HandleRequest)
}