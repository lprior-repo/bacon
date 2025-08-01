package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-xray-sdk-go/v2/xray"
    
    common "bacon/src/shared"
)

type GitHubEvent struct {
    Repository string `json:"repository"`
    Owner      string `json:"owner"`
}

type GitHubResponse struct {
    Status    string `json:"status"`
    Message   string `json:"message"`
    Timestamp string `json:"timestamp"`
}

type GitHubProcessingData struct {
    Event   GitHubEvent
    Context context.Context
    Repo    *GitHubRepo
    Segment *xray.Segment
}

type GitHubRepo struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Language    string `json:"language"`
    Stars       int    `json:"stargazers_count"`
    Forks       int    `json:"forks_count"`
}

func main() {
    lambda.Start(handleGitHubScrapeRequest)
}

func handleGitHubScrapeRequest(ctx context.Context, event GitHubEvent) (GitHubResponse, error) {
    return withTracedOperation(ctx, "github-scraper-handler", func(tracedCtx context.Context) (GitHubResponse, error) {
        pipeline := common.NewPipeline[GitHubProcessingData]()
        pipeline.AddStep(enrichWithTracing("repository", event.Repository, "owner", event.Owner))
        pipeline.AddStep(fetchRepositoryStep)
        pipeline.AddStep(storeRepositoryStep)
        pipeline.AddStep(addMetadataStep)
        
        input := GitHubProcessingData{
            Event: event,
            Context: tracedCtx,
        }
        
        result := common.WithTracedPipeline(tracedCtx, "github-processing-pipeline", pipeline, input)
        if result.IsFailure() {
            return createErrorResponse(result.Error.Error()), result.Error
        }
        
        return createSuccessResponse("GitHub repository data scraped successfully"), nil
    })
}

// Pure functions for functional composition
func buildGitHubURL(owner, repo string) string {
    return fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
}

func createAuthenticatedRequest(ctx context.Context, url string) (*http.Request, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
    }
    
    return req, nil
}

func executeHTTPRequest(req *http.Request) (*http.Response, error) {
    client := &http.Client{Timeout: 30 * time.Second}
    return client.Do(req)
}

func decodeGitHubResponse(resp *http.Response) (*GitHubRepo, error) {
    defer resp.Body.Close()
    
    var gitHubRepo GitHubRepo
    if err := json.NewDecoder(resp.Body).Decode(&gitHubRepo); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return &gitHubRepo, nil
}

// Composed functional pipeline steps
func fetchRepositoryStep(data GitHubProcessingData) (GitHubProcessingData, error) {
    return withTracedSubsegment(data.Context, "fetch-github-repository", func(ctx context.Context, seg *xray.Segment) (GitHubProcessingData, error) {
        url := buildGitHubURL(data.Event.Owner, data.Event.Repository)
        seg.AddAnnotation("github_url", url)
        
        req, err := createAuthenticatedRequest(ctx, url)
        if err != nil {
            return data, err
        }
        
        resp, err := executeHTTPRequest(req)
        if err != nil {
            return data, fmt.Errorf("failed to fetch repository: %w", err)
        }
        
        seg.AddAnnotation("response_status", resp.StatusCode)
        
        repo, err := decodeGitHubResponse(resp)
        if err != nil {
            return data, err
        }
        
        data.Repo = repo
        return data, nil
    })
}

// Pure functions for data transformation
func getTableName() string {
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        return "github-repositories"
    }
    return tableName
}

func createRepositoryItem(repo *GitHubRepo) map[string]types.AttributeValue {
    return map[string]types.AttributeValue{
        "id":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", repo.ID)},
        "name":        &types.AttributeValueMemberS{Value: repo.Name},
        "description": &types.AttributeValueMemberS{Value: repo.Description},
        "language":    &types.AttributeValueMemberS{Value: repo.Language},
        "stars":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", repo.Stars)},
        "forks":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", repo.Forks)},
        "timestamp":   &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
    }
}

func storeRepositoryStep(data GitHubProcessingData) (GitHubProcessingData, error) {
    return withTracedSubsegment(data.Context, "store-repository-data", func(ctx context.Context, seg *xray.Segment) (GitHubProcessingData, error) {
        cfg, err := config.LoadDefaultConfig(ctx)
        if err != nil {
            return data, fmt.Errorf("failed to load AWS config: %w", err)
        }
        
        client := dynamodb.NewFromConfig(cfg)
        tableName := getTableName()
        
        seg.AddAnnotation("table_name", tableName)
        seg.AddAnnotation("repository_id", data.Repo.ID)
        
        item := createRepositoryItem(data.Repo)
        
        _, err = client.PutItem(ctx, &dynamodb.PutItemInput{
            TableName: aws.String(tableName),
            Item:      item,
        })
        
        return data, err
    })
}

// Pure response constructors
func createSuccessResponse(message string) GitHubResponse {
    return GitHubResponse{
        Status:    "success",
        Message:   message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

func createErrorResponse(message string) GitHubResponse {
    log.Printf("Error: %s", message)
    return GitHubResponse{
        Status:    "error",
        Message:   message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

// Functional helper functions
func enrichWithTracing(annotations ...string) func(GitHubProcessingData) (GitHubProcessingData, error) {
    return func(data GitHubProcessingData) (GitHubProcessingData, error) {
        // Add annotations in pairs (key, value)
        for i := 0; i < len(annotations)-1; i += 2 {
            if data.Segment != nil {
                data.Segment.AddAnnotation(annotations[i], annotations[i+1])
            }
        }
        return data, nil
    }
}

func addMetadataStep(data GitHubProcessingData) (GitHubProcessingData, error) {
    if data.Segment != nil && data.Repo != nil {
        data.Segment.AddMetadata("repository_data", map[string]interface{}{
            "stars":    data.Repo.Stars,
            "forks":    data.Repo.Forks,
            "language": data.Repo.Language,
        })
    }
    return data, nil
}

// Higher-order function for tracing operations
func withTracedOperation[T any](ctx context.Context, operationName string, operation func(context.Context) (T, error)) (T, error) {
    ctx, seg := xray.BeginSubsegment(ctx, operationName)
    defer seg.Close(nil)
    
    result, err := operation(ctx)
    if err != nil {
        seg.AddError(err)
    }
    
    return result, err
}

func withTracedSubsegment(ctx context.Context, segmentName string, operation func(context.Context, *xray.Segment) (GitHubProcessingData, error)) (GitHubProcessingData, error) {
    ctx, seg := xray.BeginSubsegment(ctx, segmentName)
    defer seg.Close(nil)
    
    result, err := operation(ctx, seg)
    if err != nil {
        seg.AddError(err)
    }
    
    return result, err
}