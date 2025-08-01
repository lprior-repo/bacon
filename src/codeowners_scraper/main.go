package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type Event struct {
	Organization string `json:"organization"`
	BatchSize    int    `json:"batch_size,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
}

type GitHubGraphQLResponse struct {
	Data struct {
		Organization struct {
			Repositories struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []Repository `json:"nodes"`
			} `json:"repositories"`
		} `json:"organization"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type Repository struct {
	Name             string    `json:"name"`
	Owner            RepoOwner `json:"owner"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
	PushedAt         time.Time `json:"pushedAt"`
	Description      string    `json:"description"`
	IsPrivate        bool      `json:"isPrivate"`
	IsArchived       bool      `json:"isArchived"`
	Languages        struct {
		Nodes []Language `json:"nodes"`
	} `json:"languages"`
	Topics           struct {
		Nodes []Topic `json:"nodes"`
	} `json:"repositoryTopics"`
	Collaborators    struct {
		Nodes []Collaborator `json:"nodes"`
	} `json:"collaborators"`
	Teams            struct {
		Nodes []Team `json:"nodes"`
	} `json:"assignableUsers"` // This gets teams with repo access
	Codeowners       *Blob     `json:"codeowners"`
	CodeownersInDocs *Blob     `json:"codeownersInDocs"`
	CodeownersGithub *Blob     `json:"codeownersInGithub"`
}

type Language struct {
	Name string `json:"name"`
}

type Topic struct {
	Topic struct {
		Name string `json:"name"`
	} `json:"topic"`
}

type Collaborator struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Permission  string `json:"permission"`
	AvatarUrl   string `json:"avatarUrl"`
}

type Team struct {
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	Permission  string       `json:"permission"`
	Members     struct {
		Nodes []TeamMember `json:"nodes"`
	} `json:"members"`
}

type TeamMember struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	AvatarUrl string `json:"avatarUrl"`
}

type RepoOwner struct {
	Login string `json:"login"`
}

type Blob struct {
	Text string `json:"text"`
}

type CodeownersEntry struct {
	Path       string   `json:"path"`
	Owners     []string `json:"owners"`
	Teams      []string `json:"teams"`
	Users      []string `json:"users"`
	Repository string   `json:"repository"`
}

type OwnershipData struct {
	Organization     string            `json:"organization"`
	Repositories     []RepoOwnership   `json:"repositories"`
	OrganizationTeams []OrgTeam        `json:"organization_teams"`
	Timestamp        string            `json:"timestamp"`
	Source           string            `json:"source"`
	Confidence       float64           `json:"confidence"`
	ProcessedCount   int               `json:"processed_count"`
	SkippedCount     int               `json:"skipped_count"`
	HasMore          bool              `json:"has_more"`
	NextCursor       string            `json:"next_cursor,omitempty"`
}

type OrgTeam struct {
	Name         string       `json:"name"`
	Slug         string       `json:"slug"`
	Description  string       `json:"description"`
	Privacy      string       `json:"privacy"`
	Members      []TeamMember `json:"members"`
	Repositories []string     `json:"repositories"` // Repository names this team has access to
}

type RepoOwnership struct {
	Repository      string            `json:"repository"`
	Entries         []CodeownersEntry `json:"entries"`
	CodeownersHash  string            `json:"codeowners_hash"`
	LastModified    time.Time         `json:"last_modified"`
	CodeownersFound bool              `json:"codeowners_found"`
}

type CachedRepo struct {
	Repository     string    `json:"repository"`
	LastScraped    time.Time `json:"last_scraped"`
	LastPushed     time.Time `json:"last_pushed"`
	CodeownersHash string    `json:"codeowners_hash"`
}

func HandleRequest(ctx context.Context, event Event) (string, error) {
	ctx, seg := xray.BeginSegment(ctx, "codeowners-scraper")
	defer seg.Close(nil)

	// Set defaults
	if event.BatchSize == 0 {
		event.BatchSize = 100
	}
	if event.Organization == "" {
		event.Organization = "your-company" // Set your default org
	}

	log.Printf("Processing CODEOWNERS for org: %s, batch size: %d", event.Organization, event.BatchSize)
	seg.AddAnnotation("organization", event.Organization)
	seg.AddAnnotation("batch_size", event.BatchSize)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	githubToken, err := getGitHubToken(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token: %w", err)
	}

	ownershipData, err := scrapeCodeownersBatch(ctx, event.Organization, event.BatchSize, event.Cursor, githubToken, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to scrape CODEOWNERS: %w", err)
	}

	result, _ := json.Marshal(ownershipData)
	return string(result), nil
}

func getGitHubToken(ctx context.Context, cfg aws.Config) (string, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-github-token")
	defer seg.Close(nil)

	client := secretsmanager.NewFromConfig(cfg)
	secretName := os.Getenv("GITHUB_SECRET_ARN")

	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		seg.AddError(err)
		return "", err
	}

	return *result.SecretString, nil
}

func scrapeCodeownersBatch(ctx context.Context, org string, batchSize int, cursor, token string, cfg aws.Config) (*OwnershipData, error) {
	_, seg := xray.BeginSubsegment(ctx, "scrape-codeowners-batch")
	defer seg.Close(nil)

	// Fetch repositories from GitHub GraphQL API
	repos, hasNext, nextCursor, err := fetchRepositoriesBatch(ctx, org, batchSize, cursor, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	// Get cache information from DynamoDB
	cachedRepos, err := getCachedRepositories(ctx, cfg, org, repos)
	if err != nil {
		log.Printf("Warning: failed to get cached repos: %v", err)
		cachedRepos = make(map[string]CachedRepo) // Continue without cache
	}

	var repoOwnerships []RepoOwnership
	processedCount := 0
	skippedCount := 0

	for _, repo := range repos {
		repoKey := fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
		
		// Check if repository needs processing
		if shouldSkipRepository(repo, cachedRepos[repoKey]) {
			skippedCount++
			seg.AddAnnotation("skipped_repos", skippedCount)
			continue
		}

		ownership := processRepository(ctx, repo)
		if ownership != nil {
			repoOwnerships = append(repoOwnerships, *ownership)
			processedCount++
		}
	}

	// Update cache for processed repositories
	err = updateRepositoryCache(ctx, cfg, org, repoOwnerships)
	if err != nil {
		log.Printf("Warning: failed to update cache: %v", err)
	}

	// Fetch organization teams if this is the first batch (no cursor)
	var orgTeams []OrgTeam
	if cursor == "" {
		teams, err := fetchOrganizationTeams(ctx, org, token)
		if err != nil {
			log.Printf("Warning: failed to fetch organization teams: %v", err)
		} else {
			orgTeams = teams
		}
	}

	ownershipData := &OwnershipData{
		Organization:     org,
		Repositories:     repoOwnerships,
		OrganizationTeams: orgTeams,
		Timestamp:        time.Now().Format(time.RFC3339),
		Source:           "github-codeowners",
		Confidence:       0.8,
		ProcessedCount:   processedCount,
		SkippedCount:     skippedCount,
		HasMore:          hasNext,
		NextCursor:       nextCursor,
	}

	seg.AddAnnotation("processed_count", processedCount)
	seg.AddAnnotation("skipped_count", skippedCount)
	seg.AddAnnotation("has_more", hasNext)

	return ownershipData, nil
}

func fetchRepositoriesBatch(ctx context.Context, org string, batchSize int, cursor, token string) ([]Repository, bool, string, error) {
	_, seg := xray.BeginSubsegment(ctx, "fetch-repositories-batch")
	defer seg.Close(nil)

	query := buildGraphQLQuery(batchSize)
	variables := map[string]interface{}{
		"org":   org,
		"first": batchSize,
	}
	if cursor != "" {
		variables["after"] = cursor
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, false, "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, "", err
	}

	var response GitHubGraphQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false, "", err
	}

	if len(response.Errors) > 0 {
		return nil, false, "", fmt.Errorf("GitHub API error: %s", response.Errors[0].Message)
	}

	repos := response.Data.Organization.Repositories.Nodes
	hasNext := response.Data.Organization.Repositories.PageInfo.HasNextPage
	nextCursor := response.Data.Organization.Repositories.PageInfo.EndCursor

	seg.AddAnnotation("repos_fetched", len(repos))
	seg.AddAnnotation("has_next_page", hasNext)

	return repos, hasNext, nextCursor, nil
}

func fetchOrganizationTeams(ctx context.Context, org, token string) ([]OrgTeam, error) {
	_, seg := xray.BeginSubsegment(ctx, "fetch-organization-teams")
	defer seg.Close(nil)

	query := `
		query GetOrganizationTeams($org: String!, $first: Int!, $after: String) {
			organization(login: $org) {
				teams(first: $first, after: $after) {
					pageInfo {
						hasNextPage
						endCursor
					}
					nodes {
						name
						slug
						description
						privacy
						members(first: 100) {
							nodes {
								login
								name
								email
								role
								avatarUrl
							}
						}
						repositories(first: 100) {
							nodes {
								name
							}
						}
					}
				}
			}
		}
	`

	var allTeams []OrgTeam
	cursor := ""
	batchSize := 50

	for {
		variables := map[string]interface{}{
			"org":   org,
			"first": batchSize,
		}
		if cursor != "" {
			variables["after"] = cursor
		}

		payload := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}

		payloadBytes, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewBuffer(payloadBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var response struct {
			Data struct {
				Organization struct {
					Teams struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []struct {
							Name        string `json:"name"`
							Slug        string `json:"slug"`
							Description string `json:"description"`
							Privacy     string `json:"privacy"`
							Members     struct {
								Nodes []TeamMember `json:"nodes"`
							} `json:"members"`
							Repositories struct {
								Nodes []struct {
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"repositories"`
						} `json:"nodes"`
					} `json:"teams"`
				} `json:"organization"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			return nil, err
		}

		if len(response.Errors) > 0 {
			return nil, fmt.Errorf("GitHub API error: %s", response.Errors[0].Message)
		}

		// Convert to our team structure
		for _, team := range response.Data.Organization.Teams.Nodes {
			var repoNames []string
			for _, repo := range team.Repositories.Nodes {
				repoNames = append(repoNames, repo.Name)
			}

			orgTeam := OrgTeam{
				Name:         team.Name,
				Slug:         team.Slug,
				Description:  team.Description,
				Privacy:      team.Privacy,
				Members:      team.Members.Nodes,
				Repositories: repoNames,
			}
			allTeams = append(allTeams, orgTeam)
		}

		if !response.Data.Organization.Teams.PageInfo.HasNextPage {
			break
		}
		cursor = response.Data.Organization.Teams.PageInfo.EndCursor
	}

	seg.AddAnnotation("teams_fetched", len(allTeams))
	return allTeams, nil
}

func buildGraphQLQuery(batchSize int) string {
	return fmt.Sprintf(`
		query GetRepositoriesWithTeamsAndMembers($org: String!, $first: Int!, $after: String) {
			organization(login: $org) {
				repositories(first: $first, after: $after, orderBy: {field: PUSHED_AT, direction: DESC}) {
					pageInfo {
						hasNextPage
						endCursor
					}
					nodes {
						name
						description
						isPrivate
						isArchived
						owner {
							login
						}
						defaultBranchRef {
							name
						}
						pushedAt
						languages(first: 10, orderBy: {field: SIZE, direction: DESC}) {
							nodes {
								name
							}
						}
						repositoryTopics(first: 20) {
							nodes {
								topic {
									name
								}
							}
						}
						collaborators(first: 100) {
							nodes {
								login
								name
								email
								permission
								avatarUrl
							}
						}
						codeowners: object(expression: "HEAD:CODEOWNERS") {
							... on Blob {
								text
							}
						}
						codeownersInDocs: object(expression: "HEAD:docs/CODEOWNERS") {
							... on Blob {
								text
							}
						}
						codeownersInGithub: object(expression: "HEAD:.github/CODEOWNERS") {
							... on Blob {
								text
							}
						}
					}
				}
			}
		}
	`)
}

func shouldSkipRepository(repo Repository, cached CachedRepo) bool {
	if cached.Repository == "" {
		return false // Not cached, process it
	}

	// Skip if repository hasn't been updated since last scrape
	if repo.PushedAt.Before(cached.LastScraped) || repo.PushedAt.Equal(cached.LastScraped) {
		return true
	}

	return false
}

func processRepository(ctx context.Context, repo Repository) *RepoOwnership {
	_, seg := xray.BeginSubsegment(ctx, "process-repository")
	defer seg.Close(nil)

	repoKey := fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
	seg.AddAnnotation("repository", repoKey)

	// Find CODEOWNERS content from various locations
	var codeownersContent string
	var found bool

	if repo.Codeowners != nil && repo.Codeowners.Text != "" {
		codeownersContent = repo.Codeowners.Text
		found = true
	} else if repo.CodeownersInDocs != nil && repo.CodeownersInDocs.Text != "" {
		codeownersContent = repo.CodeownersInDocs.Text
		found = true
	} else if repo.CodeownersGithub != nil && repo.CodeownersGithub.Text != "" {
		codeownersContent = repo.CodeownersGithub.Text
		found = true
	}

	ownership := &RepoOwnership{
		Repository:      repoKey,
		CodeownersFound: found,
		LastModified:    repo.PushedAt,
	}

	if found {
		entries := parseCodeowners(codeownersContent, repoKey)
		ownership.Entries = entries
		ownership.CodeownersHash = calculateHash(codeownersContent)
		seg.AddAnnotation("entries_found", len(entries))
	}

	seg.AddAnnotation("codeowners_found", found)
	return ownership
}

func parseCodeowners(content, repository string) []CodeownersEntry {
	var entries []CodeownersEntry
	lines := strings.Split(content, "\n")
	
	// Pattern to match CODEOWNERS entries
	entryPattern := regexp.MustCompile(`^([^\s#]+)\s+(.+)$`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		matches := entryPattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		
		path := matches[1]
		ownersStr := matches[2]
		
		entry := CodeownersEntry{
			Path:       path,
			Owners:     parseOwners(ownersStr),
			Teams:      extractTeams(ownersStr),
			Users:      extractUsers(ownersStr),
			Repository: repository,
		}
		
		entries = append(entries, entry)
	}
	
	return entries
}

func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func parseOwners(ownersStr string) []string {
	owners := strings.Fields(ownersStr)
	var result []string
	for _, owner := range owners {
		if strings.HasPrefix(owner, "@") {
			result = append(result, owner)
		}
	}
	return result
}

func extractTeams(ownersStr string) []string {
	owners := strings.Fields(ownersStr)
	var teams []string
	for _, owner := range owners {
		if strings.HasPrefix(owner, "@") && strings.Contains(owner, "-team") {
			teams = append(teams, owner)
		}
	}
	return teams
}

func extractUsers(ownersStr string) []string {
	owners := strings.Fields(ownersStr)
	var users []string
	for _, owner := range owners {
		if strings.HasPrefix(owner, "@") && !strings.Contains(owner, "-team") {
			users = append(users, owner)
		}
	}
	return users
}

func getCachedRepositories(ctx context.Context, cfg aws.Config, org string, repos []Repository) (map[string]CachedRepo, error) {
	_, seg := xray.BeginSubsegment(ctx, "get-cached-repositories")
	defer seg.Close(nil)

	client := dynamodb.NewFromConfig(cfg)
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		return make(map[string]CachedRepo), nil
	}

	cached := make(map[string]CachedRepo)
	
	// Batch get items for efficiency
	var requestItems []types.TransactGetItem
	for _, repo := range repos {
		repoKey := fmt.Sprintf("%s/%s", repo.Owner.Login, repo.Name)
		requestItems = append(requestItems, types.TransactGetItem{
			Get: &types.Get{
				TableName: aws.String(tableName),
				Key: map[string]types.AttributeValue{
					"pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("REPO_CACHE#%s", org)},
					"sk": &types.AttributeValueMemberS{Value: repoKey},
				},
			},
		})
	}

	// DynamoDB has a limit of 100 items per transaction
	for i := 0; i < len(requestItems); i += 100 {
		end := i + 100
		if end > len(requestItems) {
			end = len(requestItems)
		}
		
		batch := requestItems[i:end]
		input := &dynamodb.TransactGetItemsInput{
			TransactItems: batch,
		}
		
		result, err := client.TransactGetItems(ctx, input)
		if err != nil {
			log.Printf("Warning: failed to get cached items: %v", err)
			continue
		}
		
		for j, response := range result.Responses {
			if len(response.Item) > 0 {
				// Parse cached item
				repoKey := batch[j].Get.Key["sk"].(*types.AttributeValueMemberS).Value
				if lastScrapedAttr, ok := response.Item["last_scraped"]; ok {
					if lastScrapedStr := lastScrapedAttr.(*types.AttributeValueMemberS).Value; lastScrapedStr != "" {
						if lastScraped, err := time.Parse(time.RFC3339, lastScrapedStr); err == nil {
							cached[repoKey] = CachedRepo{
								Repository:  repoKey,
								LastScraped: lastScraped,
							}
						}
					}
				}
			}
		}
	}

	seg.AddAnnotation("cached_repos_found", len(cached))
	return cached, nil
}

func updateRepositoryCache(ctx context.Context, cfg aws.Config, org string, repoOwnerships []RepoOwnership) error {
	_, seg := xray.BeginSubsegment(ctx, "update-repository-cache")
	defer seg.Close(nil)

	client := dynamodb.NewFromConfig(cfg)
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		return nil // No caching if no table configured
	}

	now := time.Now().Format(time.RFC3339)
	
	// Batch write items for efficiency
	var writeRequests []types.WriteRequest
	
	for _, ownership := range repoOwnerships {
		item := map[string]types.AttributeValue{
			"pk":              &types.AttributeValueMemberS{Value: fmt.Sprintf("REPO_CACHE#%s", org)},
			"sk":              &types.AttributeValueMemberS{Value: ownership.Repository},
			"last_scraped":    &types.AttributeValueMemberS{Value: now},
			"last_modified":   &types.AttributeValueMemberS{Value: ownership.LastModified.Format(time.RFC3339)},
			"codeowners_hash": &types.AttributeValueMemberS{Value: ownership.CodeownersHash},
			"codeowners_found": &types.AttributeValueMemberBOOL{Value: ownership.CodeownersFound},
			"ttl":             &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().AddDate(0, 1, 0).Unix())}, // 1 month TTL
		}
		
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: item},
		})
	}

	// DynamoDB has a limit of 25 items per batch write
	for i := 0; i < len(writeRequests); i += 25 {
		end := i + 25
		if end > len(writeRequests) {
			end = len(writeRequests)
		}
		
		batch := writeRequests[i:end]
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableName: batch,
			},
		}
		
		_, err := client.BatchWriteItem(ctx, input)
		if err != nil {
			log.Printf("Warning: failed to update cache batch: %v", err)
		}
	}

	seg.AddAnnotation("cache_items_updated", len(writeRequests))
	return nil
}

func main() {
	lambda.Start(HandleRequest)
}