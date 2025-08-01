package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	codeownersTypes "bacon/src/code-analysis/types"
)

type Manager struct {
	client    *dynamodb.Client
	tableName string
}

func NewManager(cfg aws.Config) *Manager {
	return &Manager{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: os.Getenv("DYNAMODB_TABLE"),
	}
}

func (m *Manager) GetCachedRepositories(ctx context.Context, org string, repos []codeownersTypes.Repository) (map[string]codeownersTypes.CachedRepo, error) {
	if m.tableName == "" {
		return make(map[string]codeownersTypes.CachedRepo), nil
	}

	cached := make(map[string]codeownersTypes.CachedRepo)
	requestItems := m.buildBatchGetItems(org, repos)
	
	return m.processBatchGetItems(ctx, requestItems, cached)
}

func (m *Manager) buildBatchGetItems(org string, repos []codeownersTypes.Repository) []types.TransactGetItem {
	var requestItems []types.TransactGetItem
	for _, repo := range repos {
		repoKey := fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
		requestItems = append(requestItems, types.TransactGetItem{
			Get: &types.Get{
				TableName: aws.String(m.tableName),
				Key: map[string]types.AttributeValue{
					"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("REPO_CACHE#%s", org)},
					"sk": &types.AttributeValueMemberS{Value: repoKey},
				},
			},
		})
	}
	return requestItems
}

func (m *Manager) processBatchGetItems(ctx context.Context, requestItems []types.TransactGetItem, cached map[string]codeownersTypes.CachedRepo) (map[string]codeownersTypes.CachedRepo, error) {
	for i := 0; i < len(requestItems); i += 100 {
		end := i + 100
		if end > len(requestItems) {
			end = len(requestItems)
		}
		
		batch := requestItems[i:end]
		err := m.processBatch(ctx, batch, cached)
		if err != nil {
			log.Printf("Warning: failed to get cached items: %v", err)
			continue
		}
	}
	return cached, nil
}

func (m *Manager) processBatch(ctx context.Context, batch []types.TransactGetItem, cached map[string]codeownersTypes.CachedRepo) error {
	input := &dynamodb.TransactGetItemsInput{
		TransactItems: batch,
	}
	
	result, err := m.client.TransactGetItems(ctx, input)
	if err != nil {
		return err
	}
	
	for j, response := range result.Responses {
		if len(response.Item) > 0 {
			m.parseCachedItem(response.Item, batch[j], cached)
		}
	}
	
	return nil
}

func (m *Manager) parseCachedItem(item map[string]types.AttributeValue, batchItem types.TransactGetItem, cached map[string]codeownersTypes.CachedRepo) {
	repoKey := batchItem.Get.Key["sk"].(*types.AttributeValueMemberS).Value
	if lastScrapedAttr, ok := item["last_scraped"]; ok {
		if lastScrapedStr := lastScrapedAttr.(*types.AttributeValueMemberS).Value; lastScrapedStr != "" {
			if lastScraped, err := time.Parse(time.RFC3339, lastScrapedStr); err == nil {
				cached[repoKey] = codeownersTypes.CachedRepo{
					Repository:  repoKey,
					LastScraped: lastScraped,
				}
			}
		}
	}
}

func (m *Manager) UpdateRepositoryCache(ctx context.Context, org string, repoOwnerships []codeownersTypes.RepoOwnership) error {
	if m.tableName == "" {
		return nil
	}

	now := time.Now().Format(time.RFC3339)
	writeRequests := m.buildWriteRequests(org, repoOwnerships, now)
	
	return m.processBatchWrites(ctx, writeRequests)
}

func (m *Manager) buildWriteRequests(org string, repoOwnerships []codeownersTypes.RepoOwnership, now string) []types.WriteRequest {
	var writeRequests []types.WriteRequest
	
	for _, ownership := range repoOwnerships {
		item := map[string]types.AttributeValue{
			"pk":              &types.AttributeValueMemberS{Value: fmt.Sprintf("REPO_CACHE#%s", org)},
			"sk":              &types.AttributeValueMemberS{Value: ownership.Repository},
			"last_scraped":    &types.AttributeValueMemberS{Value: now},
			"last_modified":   &types.AttributeValueMemberS{Value: ownership.LastModified.Format(time.RFC3339)},
			"codeowners_hash": &types.AttributeValueMemberS{Value: ownership.CodeownersHash},
			"codeowners_found": &types.AttributeValueMemberBOOL{Value: ownership.CodeownersFound},
			"ttl":             &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().AddDate(0, 1, 0).Unix())},
		}
		
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})
	}
	
	return writeRequests
}

func (m *Manager) processBatchWrites(ctx context.Context, writeRequests []types.WriteRequest) error {
	for i := 0; i < len(writeRequests); i += 25 {
		end := i + 25
		if end > len(writeRequests) {
			end = len(writeRequests)
		}
		
		batch := writeRequests[i:end]
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				m.tableName: batch,
			},
		}
		
		_, err := m.client.BatchWriteItem(ctx, input)
		if err != nil {
			log.Printf("Warning: failed to update cache batch: %v", err)
		}
	}
	
	return nil
}