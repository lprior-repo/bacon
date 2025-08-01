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
    "github.com/aws/aws-xray-sdk-go/xray"
    "github.com/aws/aws-xray-sdk-go/xrayhttp"
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
    ctx, seg := xray.BeginSubsegment(ctx, "github-scraper-handler")
    defer seg.Close(nil)

    seg.AddAnnotation("repository", event.Repository)
    seg.AddAnnotation("owner", event.Owner)

    repo, err := fetchGitHubRepository(ctx, event.Owner, event.Repository)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(err.Error()), err
    }

    err = storeRepositoryData(ctx, repo)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(fmt.Sprintf("failed to store data: %v", err)), err
    }

    seg.AddMetadata("repository_data", map[string]interface{}{
        "stars": repo.Stars,
        "forks": repo.Forks,
        "language": repo.Language,
    })

    return createSuccessResponse("GitHub repository data scraped successfully"), nil
}

func fetchGitHubRepository(ctx context.Context, owner, repo string) (*GitHubRepo, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "fetch-github-repository")
    defer seg.Close(nil)

    url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
    seg.AddAnnotation("github_url", url)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
    }

    client := xrayhttp.Client(&http.Client{Timeout: 30 * time.Second})
    resp, err := client.Do(req)
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to fetch repository: %w", err)
    }
    defer resp.Body.Close()

    seg.AddAnnotation("response_status", resp.StatusCode)

    var gitHubRepo GitHubRepo
    if err := json.NewDecoder(resp.Body).Decode(&gitHubRepo); err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &gitHubRepo, nil
}

func storeRepositoryData(ctx context.Context, repo *GitHubRepo) error {
    ctx, seg := xray.BeginSubsegment(ctx, "store-repository-data")
    defer seg.Close(nil)

    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        seg.AddError(err)
        return fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := dynamodb.NewFromConfig(cfg)
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        tableName = "github-repositories"
    }

    seg.AddAnnotation("table_name", tableName)
    seg.AddAnnotation("repository_id", repo.ID)

    item := map[string]types.AttributeValue{
        "id":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", repo.ID)},
        "name":        &types.AttributeValueMemberS{Value: repo.Name},
        "description": &types.AttributeValueMemberS{Value: repo.Description},
        "language":    &types.AttributeValueMemberS{Value: repo.Language},
        "stars":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", repo.Stars)},
        "forks":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", repo.Forks)},
        "timestamp":   &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
    }

    _, err = client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(tableName),
        Item:      item,
    })

    if err != nil {
        seg.AddError(err)
    }

    return err
}

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