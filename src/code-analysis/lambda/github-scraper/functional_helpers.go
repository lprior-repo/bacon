package main

import (
    "context"
    "fmt"

    "github.com/aws/aws-xray-sdk-go/v2/xray"
    
    common "bacon/src/shared"
)

// GitHubProcessingResult represents the result of GitHub processing operations
type GitHubProcessingResult struct {
    Data  GitHubProcessingData
    Error error
}

func (r GitHubProcessingResult) IsSuccess() bool {
    return r.Error == nil
}

func (r GitHubProcessingResult) IsFailure() bool {
    return r.Error != nil
}

// Functional composition for GitHub event processing
func processGitHubEvent(ctx context.Context, event GitHubEvent) GitHubProcessingResult {
    pipeline := common.NewPipeline[GitHubProcessingData]()
    pipeline.AddStep(enrichWithTracing("repository", event.Repository, "owner", event.Owner))
    pipeline.AddStep(fetchRepositoryStep)
    pipeline.AddStep(storeRepositoryStep)
    pipeline.AddStep(addMetadataStep)
    
    input := GitHubProcessingData{
        Event:   event,
        Context: ctx,
    }
    
    result := common.WithTracedPipeline(ctx, "github-processing-pipeline", pipeline, input)
    if result.IsFailure() {
        return GitHubProcessingResult{Error: result.Error}
    }
    
    return GitHubProcessingResult{Data: result.Value}
}

// Functional validation helpers
func validateGitHubEvent(event GitHubEvent) error {
    if event.Owner == "" {
        return fmt.Errorf("owner is required")
    }
    if event.Repository == "" {
        return fmt.Errorf("repository is required")
    }
    return nil
}

func validateGitHubRepo(repo *GitHubRepo) error {
    if repo == nil {
        return fmt.Errorf("repository data is nil")
    }
    if repo.Name == "" {
        return fmt.Errorf("repository name is required")
    }
    return nil
}