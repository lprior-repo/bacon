package types

import "time"

type Event struct {
	Organization string `json:"organization"`
	BatchSize    int    `json:"batch_size,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
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
	Codeowners       *Blob     `json:"codeowners"`
	CodeownersInDocs *Blob     `json:"codeownersInDocs"`
	CodeownersGithub *Blob     `json:"codeownersInGithub"`
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

type RepoOwnership struct {
	Repository      string            `json:"repository"`
	Entries         []CodeownersEntry `json:"entries"`
	CodeownersHash  string            `json:"codeowners_hash"`
	LastModified    time.Time         `json:"last_modified"`
	CodeownersFound bool              `json:"codeowners_found"`
}

type OwnershipData struct {
	Organization     string          `json:"organization"`
	Repositories     []RepoOwnership `json:"repositories"`
	Timestamp        string          `json:"timestamp"`
	Source           string          `json:"source"`
	Confidence       float64         `json:"confidence"`
	ProcessedCount   int             `json:"processed_count"`
	SkippedCount     int             `json:"skipped_count"`
	HasMore          bool            `json:"has_more"`
	NextCursor       string          `json:"next_cursor,omitempty"`
}

type CachedRepo struct {
	Repository     string    `json:"repository"`
	LastScraped    time.Time `json:"last_scraped"`
	LastPushed     time.Time `json:"last_pushed"`
	CodeownersHash string    `json:"codeowners_hash"`
}