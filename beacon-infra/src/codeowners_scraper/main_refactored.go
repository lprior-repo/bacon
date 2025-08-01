package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"beacon-infra/src/codeowners_scraper/cache"
	"beacon-infra/src/codeowners_scraper/github"
	"beacon-infra/src/codeowners_scraper/parser"
	"beacon-infra/src/codeowners_scraper/types"
	"beacon-infra/src/common"
)

func HandleRequest(ctx context.Context, event types.Event) (string, error) {
	pipeline := createProcessingPipeline()
	
	result := common.WithTracedPipeline(ctx, "codeowners-scraper", pipeline, event)
	if result.IsFailure() {
		return "", result.Error
	}
	
	response, _ := json.Marshal(result.Value)
	return string(response), nil
}

func createProcessingPipeline() *common.Pipeline[types.Event] {
	return common.NewPipeline[types.Event]().
		AddStep(validateEvent).
		AddStep(initializeContext).
		AddStep(fetchRepositoriesStep).
		AddStep(processRepositoriesStep).
		AddStep(buildOwnershipDataStep)
}

func validateEvent(event types.Event) (types.Event, error) {
	if event.BatchSize == 0 {
		event.BatchSize = 100
	}
	if event.Organization == "" {
		event.Organization = "your-company"
	}
	return event, nil
}

func initializeContext(event types.Event) (types.Event, error) {
	common.WithAnnotation(context.Background(), "organization", event.Organization)
	common.WithAnnotation(context.Background(), "batch_size", event.BatchSize)
	return event, nil
}

func fetchRepositoriesStep(event types.Event) (types.Event, error) {
	cfg, err := common.LoadAWSConfig(context.Background())
	if err != nil {
		return event, fmt.Errorf("failed to load AWS config: %w", err)
	}

	token, err := getGitHubToken(context.Background(), cfg)
	if err != nil {
		return event, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	client := github.NewClient(token)
	repos, hasNext, nextCursor, err := client.FetchRepositories(
		context.Background(), 
		event.Organization, 
		event.BatchSize, 
		event.Cursor,
	)
	if err != nil {
		return event, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	// Store fetched data in event for next pipeline step
	// Note: In a real implementation, you'd use a context object instead
	return event, nil
}

func processRepositoriesStep(event types.Event) (types.Event, error) {
	cfg, err := common.LoadAWSConfig(context.Background())
	if err != nil {
		return event, err
	}

	cacheManager := cache.NewManager(cfg)
	
	// This would process the repositories stored from previous step
	// Implementation simplified for brevity
	return event, nil
}

func buildOwnershipDataStep(event types.Event) (types.Event, error) {
	// Build final ownership data structure
	// Implementation simplified for brevity
	return event, nil
}

func processRepository(ctx context.Context, repo types.Repository) *types.RepoOwnership {
	repoKey := fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
	
	ownership := &types.RepoOwnership{
		Repository:      repoKey,
		CodeownersFound: false,
		LastModified:    repo.PushedAt,
	}

	content := extractCodeownersContent(repo)
	if content != "" {
		entries := parser.ParseCodeowners(content, repoKey)
		ownership.Entries = entries
		ownership.CodeownersHash = parser.CalculateHash(content)
		ownership.CodeownersFound = true
	}

	return ownership
}

func extractCodeownersContent(repo types.Repository) string {
	if repo.Codeowners != nil && repo.Codeowners.Text != "" {
		return repo.Codeowners.Text
	}
	if repo.CodeownersInDocs != nil && repo.CodeownersInDocs.Text != "" {
		return repo.CodeownersInDocs.Text
	}
	if repo.CodeownersGithub != nil && repo.CodeownersGithub.Text != "" {
		return repo.CodeownersGithub.Text
	}
	return ""
}

func shouldSkipRepository(repo types.Repository, cached types.CachedRepo) bool {
	if cached.Repository == "" {
		return false
	}
	return repo.PushedAt.Before(cached.LastScraped) || repo.PushedAt.Equal(cached.LastScraped)
}

func getGitHubToken(ctx context.Context, cfg aws.Config) (string, error) {
	client := common.CreateSecretsClient(cfg)
	secretName := os.Getenv("GITHUB_SECRET_ARN")

	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", err
	}

	return *result.SecretString, nil
}

func main() {
	lambda.Start(HandleRequest)
}