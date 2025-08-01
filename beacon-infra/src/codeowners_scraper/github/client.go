package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"beacon-infra/src/codeowners_scraper/types"
	"beacon-infra/src/common"
)

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) FetchRepositories(ctx context.Context, org string, batchSize int, cursor string) ([]types.Repository, bool, string, error) {
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

	return c.executeGraphQLQuery(ctx, payload)
}

func (c *Client) executeGraphQLQuery(ctx context.Context, payload map[string]interface{}) ([]types.Repository, bool, string, error) {
	payloadBytes, _ := json.Marshal(payload)
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, false, "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, "", err
	}

	return parseGraphQLResponse(body)
}

func parseGraphQLResponse(body []byte) ([]types.Repository, bool, string, error) {
	var response struct {
		Data struct {
			Organization struct {
				Repositories struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []types.Repository `json:"nodes"`
				} `json:"repositories"`
			} `json:"organization"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false, "", err
	}

	if len(response.Errors) > 0 {
		return nil, false, "", fmt.Errorf("GitHub API error: %s", response.Errors[0].Message)
	}

	repos := response.Data.Organization.Repositories.Nodes
	hasNext := response.Data.Organization.Repositories.PageInfo.HasNextPage
	nextCursor := response.Data.Organization.Repositories.PageInfo.EndCursor

	return repos, hasNext, nextCursor, nil
}

func buildGraphQLQuery(batchSize int) string {
	return fmt.Sprintf(`
		query GetRepositoriesWithCodeowners($org: String!, $first: Int!, $after: String) {
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
						owner {
							login
						}
						defaultBranchRef {
							name
						}
						pushedAt
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