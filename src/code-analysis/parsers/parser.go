package parsers

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"bacon/src/code-analysis/types"
)

var entryPattern = regexp.MustCompile(`^([^\s#]+)\s+(.+)$`)

func ParseCodeowners(content, repository string) []types.CodeownersEntry {
	lines := strings.Split(content, "\n")
	var entries []types.CodeownersEntry
	
	for _, line := range lines {
		entry := parseLine(line, repository)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	
	return entries
}

func parseLine(line, repository string) *types.CodeownersEntry {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	
	matches := entryPattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil
	}
	
	path := matches[1]
	ownersStr := matches[2]
	
	return &types.CodeownersEntry{
		Path:       path,
		Owners:     extractOwners(ownersStr),
		Teams:      extractTeams(ownersStr),
		Users:      extractUsers(ownersStr),
		Repository: repository,
	}
}

func extractOwners(ownersStr string) []string {
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

func CalculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}