// Package shared provides pure functional AWS utilities for Datadog data storage.
package shared

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/samber/lo"
	ddTypes "bacon/src/plugins/datadog/types"
)

// Higher-order function for tracing operations
// Pure function that wraps operations with X-Ray tracing
func WithTracedOperation[T any](ctx context.Context, operationName string, operation func(context.Context) (T, error)) (T, error) {
	ctx, seg := xray.BeginSubsegment(ctx, operationName)
	defer seg.Close(nil)

	result, err := operation(ctx)
	if err != nil {
		_ = seg.AddError(err)
	}

	return result, err
}

// Pure functional storage operations using samber/lo

// StoreTeamsData stores team data using functional transformations
// Pure functional pipeline for team storage
func StoreTeamsData(ctx context.Context, teams []ddTypes.DatadogTeam) ([]string, error) {
	return WithTracedOperation(ctx, "store-teams-data", func(tracedCtx context.Context) ([]string, error) {
		client, err := createDynamoDBClient(tracedCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamodb client: %w", err)
		}

		tableName := getTableName("DATADOG_TEAMS_TABLE", "datadog-teams")
		
		// Pure functional transformation pipeline
		storageItems := lo.Map(teams, createTeamStorageItem)
		batches := lo.Chunk(storageItems, 25) // DynamoDB batch limit
		
		var storedIDs []string
		for i, batch := range batches {
			batchIDs, err := executeBatchWrite(tracedCtx, client, tableName, batch)
			if err != nil {
				return nil, fmt.Errorf("failed to store teams batch %d: %w", i, err)
			}
			storedIDs = append(storedIDs, batchIDs...)
		}
		
		return storedIDs, nil
	})
}

// StoreUsersData stores user data using functional transformations
// Pure functional pipeline for user storage
func StoreUsersData(ctx context.Context, users []ddTypes.DatadogUser) ([]string, error) {
	return WithTracedOperation(ctx, "store-users-data", func(tracedCtx context.Context) ([]string, error) {
		client, err := createDynamoDBClient(tracedCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamodb client: %w", err)
		}

		tableName := getTableName("DATADOG_USERS_TABLE", "datadog-users")
		
		// Pure functional transformation pipeline
		storageItems := lo.Map(users, createUserStorageItem)
		batches := lo.Chunk(storageItems, 25)
		
		var storedIDs []string
		for i, batch := range batches {
			batchIDs, err := executeBatchWrite(tracedCtx, client, tableName, batch)
			if err != nil {
				return nil, fmt.Errorf("failed to store users batch %d: %w", i, err)
			}
			storedIDs = append(storedIDs, batchIDs...)
		}
		
		return storedIDs, nil
	})
}

// StoreServicesData stores service data using functional transformations
// Pure functional pipeline for service storage
func StoreServicesData(ctx context.Context, services []ddTypes.DatadogService) ([]string, error) {
	return WithTracedOperation(ctx, "store-services-data", func(tracedCtx context.Context) ([]string, error) {
		client, err := createDynamoDBClient(tracedCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamodb client: %w", err)
		}

		tableName := getTableName("DATADOG_SERVICES_TABLE", "datadog-services")
		
		// Pure functional transformation pipeline
		storageItems := lo.Map(services, createServiceStorageItem)
		batches := lo.Chunk(storageItems, 25)
		
		var storedIDs []string
		for i, batch := range batches {
			batchIDs, err := executeBatchWrite(tracedCtx, client, tableName, batch)
			if err != nil {
				return nil, fmt.Errorf("failed to store services batch %d: %w", i, err)
			}
			storedIDs = append(storedIDs, batchIDs...)
		}
		
		return storedIDs, nil
	})
}

// Pure transformation functions for DynamoDB storage items

// createTeamStorageItem converts a team to DynamoDB storage format
// Pure function with no side effects
func createTeamStorageItem(team ddTypes.DatadogTeam, _ int) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"team_id":     &types.AttributeValueMemberS{Value: team.ID},
		"name":        &types.AttributeValueMemberS{Value: team.Name},
		"handle":      &types.AttributeValueMemberS{Value: team.Handle},
		"description": &types.AttributeValueMemberS{Value: team.Description},
		"members":     createStringListAttribute(lo.Map(team.Members, func(user ddTypes.DatadogUser, _ int) string { return user.ID })),
		"services":    createStringListAttribute(lo.Map(team.Services, func(service ddTypes.DatadogService, _ int) string { return service.ID })),
		"links":       createLinksAttribute(team.Links),
		"metadata":    createMapAttribute(team.Metadata),
		"created_at":  &types.AttributeValueMemberS{Value: team.CreatedAt.Format(time.RFC3339)},
		"updated_at":  &types.AttributeValueMemberS{Value: team.UpdatedAt.Format(time.RFC3339)},
		"scraped_at":  &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}
}

// createUserStorageItem converts a user to DynamoDB storage format
// Pure function with no side effects
func createUserStorageItem(user ddTypes.DatadogUser, _ int) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"user_id":    &types.AttributeValueMemberS{Value: user.ID},
		"name":       &types.AttributeValueMemberS{Value: user.Name},
		"email":      &types.AttributeValueMemberS{Value: user.Email},
		"handle":     &types.AttributeValueMemberS{Value: user.Handle},
		"teams":      createStringListAttribute(user.Teams),
		"roles":      createStringListAttribute(user.Roles),
		"status":     &types.AttributeValueMemberS{Value: user.Status},
		"verified":   &types.AttributeValueMemberBOOL{Value: user.Verified},
		"disabled":   &types.AttributeValueMemberBOOL{Value: user.Disabled},
		"title":      &types.AttributeValueMemberS{Value: user.Title},
		"icon":       &types.AttributeValueMemberS{Value: user.Icon},
		"created_at": &types.AttributeValueMemberS{Value: user.CreatedAt.Format(time.RFC3339)},
		"updated_at": &types.AttributeValueMemberS{Value: user.UpdatedAt.Format(time.RFC3339)},
		"scraped_at": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}
}

// createServiceStorageItem converts a service to DynamoDB storage format
// Pure function with no side effects
func createServiceStorageItem(service ddTypes.DatadogService, _ int) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"service_id":    &types.AttributeValueMemberS{Value: service.ID},
		"name":          &types.AttributeValueMemberS{Value: service.Name},
		"owner":         &types.AttributeValueMemberS{Value: service.Owner},
		"teams":         createStringListAttribute(service.Teams),
		"tags":          createStringListAttribute(service.Tags),
		"schema":        &types.AttributeValueMemberS{Value: service.Schema},
		"description":   &types.AttributeValueMemberS{Value: service.Description},
		"tier":          &types.AttributeValueMemberS{Value: service.Tier},
		"lifecycle":     &types.AttributeValueMemberS{Value: service.Lifecycle},
		"type":          &types.AttributeValueMemberS{Value: service.Type},
		"languages":     createStringListAttribute(service.Languages),
		"contacts":      createContactsAttribute(service.Contacts),
		"links":         createServiceLinksAttribute(service.Links),
		"integrations":  createMapAttribute(service.Integrations),
		"dependencies":  createStringListAttribute(service.Dependencies),
		"created_at":    &types.AttributeValueMemberS{Value: service.CreatedAt.Format(time.RFC3339)},
		"updated_at":    &types.AttributeValueMemberS{Value: service.UpdatedAt.Format(time.RFC3339)},
		"scraped_at":    &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}
}

// Pure helper functions for DynamoDB attribute creation

// createStringListAttribute creates a DynamoDB string list attribute
// Pure function for attribute creation
func createStringListAttribute(items []string) *types.AttributeValueMemberSS {
	if len(items) == 0 {
		return &types.AttributeValueMemberSS{Value: []string{}}
	}
	return &types.AttributeValueMemberSS{Value: items}
}

// createLinksAttribute creates a DynamoDB attribute for team links
// Pure function using lo.Map for transformation
func createLinksAttribute(links []ddTypes.DatadogTeamLink) *types.AttributeValueMemberL {
	linkItems := lo.Map(links, func(link ddTypes.DatadogTeamLink, _ int) types.AttributeValue {
		return &types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"label": &types.AttributeValueMemberS{Value: link.Label},
				"url":   &types.AttributeValueMemberS{Value: link.URL},
				"type":  &types.AttributeValueMemberS{Value: link.Type},
			},
		}
	})
	
	return &types.AttributeValueMemberL{Value: linkItems}
}

// createContactsAttribute creates a DynamoDB attribute for service contacts
// Pure function using lo.Map for transformation
func createContactsAttribute(contacts []ddTypes.DatadogContact) *types.AttributeValueMemberL {
	contactItems := lo.Map(contacts, func(contact ddTypes.DatadogContact, _ int) types.AttributeValue {
		return &types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"name":    &types.AttributeValueMemberS{Value: contact.Name},
				"type":    &types.AttributeValueMemberS{Value: contact.Type},
				"contact": &types.AttributeValueMemberS{Value: contact.Contact},
			},
		}
	})
	
	return &types.AttributeValueMemberL{Value: contactItems}
}

// createServiceLinksAttribute creates a DynamoDB attribute for service links
// Pure function using lo.Map for transformation
func createServiceLinksAttribute(links []ddTypes.DatadogServiceLink) *types.AttributeValueMemberL {
	linkItems := lo.Map(links, func(link ddTypes.DatadogServiceLink, _ int) types.AttributeValue {
		return &types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"name": &types.AttributeValueMemberS{Value: link.Name},
				"type": &types.AttributeValueMemberS{Value: link.Type},
				"url":  &types.AttributeValueMemberS{Value: link.URL},
			},
		}
	})
	
	return &types.AttributeValueMemberL{Value: linkItems}
}

// createMapAttribute creates a DynamoDB map attribute from interface{}
// Pure function for map attribute creation
func createMapAttribute(data map[string]interface{}) *types.AttributeValueMemberM {
	result := make(map[string]types.AttributeValue)
	
	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = &types.AttributeValueMemberS{Value: v}
		case bool:
			result[key] = &types.AttributeValueMemberBOOL{Value: v}
		case float64:
			result[key] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", v)}
		default:
			result[key] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%v", v)}
		}
	}
	
	return &types.AttributeValueMemberM{Value: result}
}

// Infrastructure helper functions

// createDynamoDBClient creates a new DynamoDB client with proper configuration
// Pure function for client creation
func createDynamoDBClient(ctx context.Context) (*dynamodb.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return dynamodb.NewFromConfig(cfg), nil
}

// getTableName gets table name from environment or uses default
// Pure function for table name resolution
func getTableName(envVar, defaultName string) string {
	tableName := os.Getenv(envVar)
	return lo.Ternary(tableName != "", tableName, defaultName)
}

// executeBatchWrite executes a batch write operation to DynamoDB
// Pure function for batch write operations
func executeBatchWrite(ctx context.Context, client *dynamodb.Client, tableName string, items []map[string]types.AttributeValue) ([]string, error) {
	if len(items) == 0 {
		return []string{}, nil
	}

	// Create batch write items using lo.Map
	writeRequests := lo.Map(items, func(item map[string]types.AttributeValue, _ int) types.WriteRequest {
		return types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		}
	})

	_, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			tableName: writeRequests,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to execute batch write: %w", err)
	}

	// Extract IDs from stored items using lo.Map
	storedIDs := lo.Map(items, func(item map[string]types.AttributeValue, _ int) string {
		// Try different ID field names depending on the data type
		for _, idField := range []string{"team_id", "user_id", "service_id"} {
			if idAttr, exists := item[idField]; exists {
				if s, ok := idAttr.(*types.AttributeValueMemberS); ok {
					return s.Value
				}
			}
		}
		return ""
	})

	return lo.Compact(storedIDs), nil
}