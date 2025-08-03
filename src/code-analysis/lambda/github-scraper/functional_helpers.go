// Package main provides functional helper types and methods for GitHub repository processing operations.
package main

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

