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
	
	// Functional approach using filter and map
	validLines := filterNonEmpty(lines)
	entries := mapToEntries(validLines, repository)
	
	return filterValidEntries(entries)
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
	return filterByPrefix(owners, "@")
}

func extractTeams(ownersStr string) []string {
	owners := strings.Fields(ownersStr)
	return filterByPrefixAndSuffix(owners, "@", "-team")
}

func extractUsers(ownersStr string) []string {
	owners := strings.Fields(ownersStr)
	return filterUsers(owners)
}

// Pure function for hash calculation
func CalculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// Pure helper functions for functional composition
func filterNonEmpty(lines []string) []string {
	return filter(lines, func(line string) bool {
		trimmed := strings.TrimSpace(line)
		return trimmed != "" && !strings.HasPrefix(trimmed, "#")
	})
}

func mapToEntries(lines []string, repository string) []*types.CodeownersEntry {
	return mapWithRepository(lines, repository, parseLine)
}

func filterValidEntries(entries []*types.CodeownersEntry) []types.CodeownersEntry {
	var result []types.CodeownersEntry
	for _, entry := range entries {
		if entry != nil {
			result = append(result, *entry)
		}
	}
	return result
}

func filterByPrefix(items []string, prefix string) []string {
	return filter(items, func(item string) bool {
		return strings.HasPrefix(item, prefix)
	})
}

func filterByPrefixAndSuffix(items []string, prefix, suffix string) []string {
	return filter(items, func(item string) bool {
		return strings.HasPrefix(item, prefix) && strings.Contains(item, suffix)
	})
}

func filterUsers(items []string) []string {
	return filter(items, func(item string) bool {
		return strings.HasPrefix(item, "@") && !strings.Contains(item, "-team")
	})
}

// Generic functional helpers
func filter[T any](items []T, predicate func(T) bool) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func mapWithRepository[T, R any](items []T, repository string, transform func(T, string) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = transform(item, repository)
	}
	return result
}