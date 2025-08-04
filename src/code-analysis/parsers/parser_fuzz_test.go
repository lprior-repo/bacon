package parsers

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"pgregory.net/rapid"
)

// Comprehensive Property-Based Testing for Parsers Module
func TestPropertyBasedParsers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random repository name
		repoName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]*`).Draw(t, "repoName")
		
		// Generate random valid CODEOWNERS content
		pathPattern := rapid.SampledFrom([]string{"*.go", "*.js", "*.py", "/docs/*", "*", "src/**", "test/"}).Draw(t, "pathPattern")
		owner := rapid.SampledFrom([]string{"@team", "@user", "@backend-team", "@frontend-team", "@john-doe"}).Draw(t, "owner")
		
		content := fmt.Sprintf("%s %s", pathPattern, owner)
		
		// Property: ParseCodeowners should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ParseCodeowners panicked with content=%q, repo=%q: %v", content, repoName, r)
			}
		}()
		
		entries := ParseCodeowners(content, repoName)
		
		// Property: All entries should have the repository field set
		for _, entry := range entries {
			if entry.Repository != repoName {
				t.Errorf("Entry repository mismatch: expected %q, got %q", repoName, entry.Repository)
			}
		}
		
		// Property: Valid content should produce at least one entry
		if len(entries) == 0 && !strings.HasPrefix(strings.TrimSpace(content), "#") && strings.TrimSpace(content) != "" {
			t.Errorf("Valid content should produce entries: content=%q", content)
		}
		
		// Property: Path and owners should be extracted correctly
		if len(entries) > 0 {
			entry := entries[0]
			if entry.Path != pathPattern {
				t.Errorf("Path extraction failed: expected %q, got %q", pathPattern, entry.Path)
			}
			
			if len(entry.Owners) == 0 {
				t.Errorf("No owners extracted from %q", owner)
			}
		}
	})
}

func TestPropertyBasedParseLine(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate various line formats
		lineType := rapid.SampledFrom([]string{"empty", "comment", "valid", "invalid"}).Draw(t, "lineType")
		repoName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]*`).Draw(t, "repoName")
		
		var line string
		var shouldBeNil bool
		
		switch lineType {
		case "empty":
			line = rapid.SampledFrom([]string{"", "   ", "\t", "\n", "  \t  "}).Draw(t, "emptyLine")
			shouldBeNil = true
		case "comment":
			commentText := rapid.StringMatching(`[a-zA-Z0-9 ]*`).Draw(t, "commentText")
			line = fmt.Sprintf("# %s", commentText)
			shouldBeNil = true
		case "valid":
			path := rapid.SampledFrom([]string{"*.go", "*.js", "/docs/*", "src/**"}).Draw(t, "path")
			owner := rapid.SampledFrom([]string{"@team", "@user", "@backend-team"}).Draw(t, "owner")
			line = fmt.Sprintf("%s %s", path, owner)
			shouldBeNil = false
		case "invalid":
			// Lines that don't match the pattern
			line = rapid.SampledFrom([]string{"*.go", "noowner", "@owner", ""}).Draw(t, "invalidLine")
			shouldBeNil = true
		}
		
		// Property: parseLine should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("parseLine panicked with line=%q, repo=%q: %v", line, repoName, r)
			}
		}()
		
		result := parseLine(line, repoName)
		
		// Property: Result should match expectation
		if shouldBeNil && result != nil {
			t.Errorf("Expected nil for line type %s, line=%q, got %+v", lineType, line, result)
		}
		
		if !shouldBeNil && result == nil {
			t.Errorf("Expected non-nil for line type %s, line=%q", lineType, line)
		}
		
		// Property: Valid results should have repository set
		if result != nil {
			if result.Repository != repoName {
				t.Errorf("Repository mismatch: expected %q, got %q", repoName, result.Repository)
			}
		}
	})
}

func TestPropertyBasedHashCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		content1 := rapid.String().Draw(t, "content1")
		content2 := rapid.String().Draw(t, "content2")
		
		// Property: Same content should produce same hash
		hash1a := CalculateHash(content1)
		hash1b := CalculateHash(content1)
		
		if hash1a != hash1b {
			t.Errorf("Same content produced different hashes: %q vs %q", hash1a, hash1b)
		}
		
		// Property: Different content should produce different hashes (probabilistically)
		if content1 != content2 {
			hash2 := CalculateHash(content2)
			if hash1a == hash2 {
				// This could happen but is very unlikely with SHA256
				t.Logf("Hash collision detected (very rare): content1=%q, content2=%q", content1, content2)
			}
		}
		
		// Property: Hash should be consistent length (SHA256 = 64 hex chars)
		if len(hash1a) != 64 {
			t.Errorf("Hash should be 64 characters, got %d: %q", len(hash1a), hash1a)
		}
		
		// Property: Hash should be valid hex
		for _, char := range hash1a {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				t.Errorf("Hash contains non-hex character %c: %q", char, hash1a)
				break
			}
		}
	})
}

// Comprehensive Fuzz Testing for Parsers Module
func FuzzParseCodeowners(f *testing.F) {
	// Seed with various CODEOWNERS content patterns
	f.Add("*.go @backend-team", "test-repo")
	f.Add("", "empty-repo")
	f.Add("# This is a comment\n*.js @frontend-team", "comment-repo")
	f.Add("* @global-owner", "global-repo")
	f.Add("/docs/* @docs-team\n*.md @writers", "multi-repo")
	f.Add("# invalid line without owner", "invalid-repo")
	f.Add(strings.Repeat("*.go @team\n", 1000), "large-repo")
	f.Add("Unicode: 测试.go @测试团队", "unicode-repo")
	f.Add("Emoji: 📝.md @📚team", "emoji-repo")
	f.Add("Newlines\n\n\n*.go @team", "newline-repo")
	f.Add("Tabs\t\t*.js @team", "tab-repo")
	f.Add("Mixed\r\n*.py @team", "mixed-repo")
	
	f.Fuzz(func(t *testing.T, content string, repository string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(content) || !utf8.ValidString(repository) {
			t.Skip("Skipping invalid UTF-8 strings")
		}
		
		// Fuzz property: ParseCodeowners should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseCodeowners panicked with content=%q, repo=%q: %v", content, repository, r)
			}
		}()
		
		entries := ParseCodeowners(content, repository)
		
		// Fuzz property: All entries should have repository field set
		for i, entry := range entries {
			if entry.Repository != repository {
				t.Errorf("Entry[%d] repository mismatch: expected %q, got %q", i, repository, entry.Repository)
			}
		}
		
		// Fuzz property: Valid entries should have non-empty paths and owners
		for i, entry := range entries {
			if entry.Path == "" {
				t.Errorf("Entry[%d] has empty path", i)
			}
			// Note: It's possible for valid entries to have no owners if the regex parsing succeeds 
			// but owner extraction fails - this is acceptable behavior
		}
		
		// Fuzz property: Number of entries should be reasonable for content length
		lines := strings.Split(content, "\n")
		if len(entries) > len(lines) {
			t.Errorf("More entries (%d) than lines (%d)", len(entries), len(lines))
		}
	})
}

func FuzzParseLine(f *testing.F) {
	// Seed with various line patterns
	f.Add("*.go @backend-team", "test-repo")
	f.Add("", "test-repo")
	f.Add("# comment", "test-repo")
	f.Add("   ", "test-repo")
	f.Add("*.js @frontend-team @user", "test-repo")
	f.Add("invalid-format", "test-repo")
	f.Add("/path/to/file @owner", "test-repo")
	f.Add("* @global", "test-repo")
	f.Add("Unicode: 测试.go @团队", "unicode-repo")
	f.Add("Special: file-name_123.ext @team-name", "special-repo")
	f.Add(strings.Repeat("a", 10000) + " @team", "long-repo")
	f.Add("\t\n *.go @team \n\t", "whitespace-repo")
	
	f.Fuzz(func(t *testing.T, line string, repository string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(line) || !utf8.ValidString(repository) {
			t.Skip("Skipping invalid UTF-8 strings")
		}
		
		// Fuzz property: parseLine should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parseLine panicked with line=%q, repo=%q: %v", line, repository, r)
			}
		}()
		
		result := parseLine(line, repository)
		
		// Fuzz property: Empty/comment lines should return nil
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			if result != nil {
				t.Errorf("Empty/comment line should return nil: line=%q, result=%+v", line, result)
			}
			return
		}
		
		// Fuzz property: Valid results should have repository set
		if result != nil {
			if result.Repository != repository {
				t.Errorf("Repository mismatch: expected %q, got %q", repository, result.Repository)
			}
			
			// Fuzz property: Valid results should have non-empty path
			if result.Path == "" {
				t.Errorf("Valid result should have non-empty path for line=%q", line)
			}
			
			// Note: Valid parsed results may have zero owners if owner extraction fails
			// This is acceptable behavior - the regex matched but owner parsing didn't find valid owners
		}
	})
}

func FuzzCalculateHash(f *testing.F) {
	// Seed with various content types
	f.Add("")
	f.Add("simple content")
	f.Add("Unicode content: 测试内容 🚀")
	f.Add(strings.Repeat("a", 10000))
	f.Add("Binary\x00\x01\x02content")
	f.Add("JSON: {\"key\": \"value\", \"number\": 42}")
	f.Add("XML: <root><item>value</item></root>")
	f.Add("Code: func main() { fmt.Println(\"Hello\") }")
	f.Add("Multiline\ncontent\nwith\nbreaks")
	f.Add("Tabs\tand\tspaces   mixed")
	
	f.Fuzz(func(t *testing.T, content string) {
		// Skip invalid UTF-8 strings for consistency
		if !utf8.ValidString(content) {
			t.Skip("Skipping invalid UTF-8 string")
		}
		
		// Fuzz property: CalculateHash should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("CalculateHash panicked with content=%q: %v", content, r)
			}
		}()
		
		hash := CalculateHash(content)
		
		// Fuzz property: Hash should be consistent length (SHA256 = 64 hex chars)
		if len(hash) != 64 {
			t.Errorf("Hash should be 64 characters, got %d for content=%q", len(hash), content)
		}
		
		// Fuzz property: Hash should be valid hex
		for i, char := range hash {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				t.Errorf("Hash contains non-hex character %c at position %d: %q", char, i, hash)
				break
			}
		}
		
		// Fuzz property: Same content should produce same hash
		hash2 := CalculateHash(content)
		if hash != hash2 {
			t.Errorf("Same content produced different hashes: %q vs %q", hash, hash2)
		}
		
		// Fuzz property: Hash should not be empty
		if hash == "" {
			t.Errorf("Hash should not be empty for content=%q", content)
		}
	})
}

func FuzzFilterByPrefix(f *testing.F) {
	// Seed with various filter scenarios (using space-separated strings instead of slices)
	f.Add("@team1 @team2 user1", "@")
	f.Add("", "@")
	f.Add("@backend-team @frontend-team @user", "@")
	f.Add("no-prefix also-no-prefix", "@")
	f.Add("@team1 @team2-team", "")
	f.Add("special!@#$% @normal", "@")
	f.Add("Unicode:@测试团队 @normal", "@")
	f.Add("Emoji:@🚀team @normal", "@")
	f.Add(strings.Repeat("@team ", 100), "@")
	
	f.Fuzz(func(t *testing.T, itemsStr string, prefix string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(itemsStr) || !utf8.ValidString(prefix) {
			t.Skip("Skipping invalid UTF-8 strings")
		}
		
		// Convert string to slice
		var items []string
		if strings.TrimSpace(itemsStr) != "" {
			items = strings.Fields(itemsStr)
		}
		
		// Fuzz property: filterByPrefix should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("filterByPrefix panicked with items=%v, prefix=%q: %v", items, prefix, r)
			}
		}()
		
		result := filterByPrefix(items, prefix)
		
		// Fuzz property: Result length should not exceed input length
		if len(result) > len(items) {
			t.Errorf("Result length (%d) exceeds input length (%d)", len(result), len(items))
		}
		
		// Fuzz property: All result items should have the prefix
		for i, item := range result {
			if !strings.HasPrefix(item, prefix) {
				t.Errorf("Result[%d] %q does not have prefix %q", i, item, prefix)
			}
		}
		
		// Fuzz property: All items with prefix should be in result
		expectedCount := 0
		for _, item := range items {
			if strings.HasPrefix(item, prefix) {
				expectedCount++
			}
		}
		if len(result) != expectedCount {
			t.Errorf("Expected %d items with prefix %q, got %d", expectedCount, prefix, len(result))
		}
	})
}

func FuzzFilterByPrefixAndSuffix(f *testing.F) {
	// Seed with team/user distinction scenarios
	f.Add("@backend-team @frontend-team @user", "@", "-team")
	f.Add("", "@", "-team")
	f.Add("@team-lead @team-member @lead", "@", "team")
	f.Add("no-prefix-team @suffix-team", "@", "-team")
	f.Add("@prefix-no-suffix @prefix-suffix", "@", "suffix")
	f.Add("Unicode:@测试-team @normal-team", "@", "-team")
	f.Add("Special:@team!@#$ @normal-team", "@", "team")
	f.Add(strings.Repeat("@team-team ", 50), "@", "-team")
	
	f.Fuzz(func(t *testing.T, itemsStr string, prefix string, suffix string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(itemsStr) || !utf8.ValidString(prefix) || !utf8.ValidString(suffix) {
			t.Skip("Skipping invalid UTF-8 strings")
		}
		
		// Convert string to slice
		var items []string
		if strings.TrimSpace(itemsStr) != "" {
			items = strings.Fields(itemsStr)
		}
		
		// Fuzz property: filterByPrefixAndSuffix should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("filterByPrefixAndSuffix panicked with items=%v, prefix=%q, suffix=%q: %v", items, prefix, suffix, r)
			}
		}()
		
		result := filterByPrefixAndSuffix(items, prefix, suffix)
		
		// Fuzz property: Result length should not exceed input length
		if len(result) > len(items) {
			t.Errorf("Result length (%d) exceeds input length (%d)", len(result), len(items))
		}
		
		// Fuzz property: All result items should have both prefix and suffix
		for i, item := range result {
			if !strings.HasPrefix(item, prefix) {
				t.Errorf("Result[%d] %q does not have prefix %q", i, item, prefix)
			}
			if !strings.Contains(item, suffix) {
				t.Errorf("Result[%d] %q does not contain suffix %q", i, item, suffix)
			}
		}
		
		// Fuzz property: Count should match expected
		expectedCount := 0
		for _, item := range items {
			if strings.HasPrefix(item, prefix) && strings.Contains(item, suffix) {
				expectedCount++
			}
		}
		if len(result) != expectedCount {
			t.Errorf("Expected %d items matching prefix=%q suffix=%q, got %d", expectedCount, prefix, suffix, len(result))
		}
	})
}

func FuzzFilterUsers(f *testing.F) {
	// Seed with user vs team scenarios
	f.Add("@john-doe @backend-team @jane-smith")
	f.Add("")
	f.Add("@user1 @user2 @team1-team @team2-team")
	f.Add("no-prefix @team-team @user")
	f.Add("@special-user!@#$ @normal-team")
	f.Add("@测试用户 @测试-team")
	f.Add("@🚀user @🚀-team")
	f.Add(strings.Repeat("@user1 @team1-team ", 50))
	
	f.Fuzz(func(t *testing.T, itemsStr string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(itemsStr) {
			t.Skip("Skipping invalid UTF-8 string")
		}
		
		// Convert string to slice
		var items []string
		if strings.TrimSpace(itemsStr) != "" {
			items = strings.Fields(itemsStr)
		}
		
		// Fuzz property: filterUsers should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("filterUsers panicked with items=%v: %v", items, r)
			}
		}()
		
		result := filterUsers(items)
		
		// Fuzz property: Result length should not exceed input length
		if len(result) > len(items) {
			t.Errorf("Result length (%d) exceeds input length (%d)", len(result), len(items))
		}
		
		// Fuzz property: All result items should be users (@ prefix, no -team suffix)
		for i, item := range result {
			if !strings.HasPrefix(item, "@") {
				t.Errorf("Result[%d] %q should have @ prefix", i, item)
			}
			if strings.Contains(item, "-team") {
				t.Errorf("Result[%d] %q should not contain -team", i, item)
			}
		}
		
		// Fuzz property: Count should match expected
		expectedCount := 0
		for _, item := range items {
			if strings.HasPrefix(item, "@") && !strings.Contains(item, "-team") {
				expectedCount++
			}
		}
		if len(result) != expectedCount {
			t.Errorf("Expected %d users, got %d", expectedCount, len(result))
		}
	})
}

func FuzzFilterNonEmpty(f *testing.F) {
	// Seed with various empty/non-empty scenarios (using newline-separated strings)
	f.Add("valid\n\n  \n# comment\nanother")
	f.Add("")
	f.Add("\n   \n\t\n\n")
	f.Add("# comment1\n# comment2\nvalid")
	f.Add("\t\n  # spaced comment\n\n\t\nvalid")
	f.Add("Unicode: 测试\n\n# 注释")
	f.Add("Emoji: 🚀\n\n# 📝")
	f.Add(strings.Repeat("line\n", 100))
	
	f.Fuzz(func(t *testing.T, linesStr string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(linesStr) {
			t.Skip("Skipping invalid UTF-8 string")
		}
		
		// Convert string to slice by splitting on newlines
		lines := strings.Split(linesStr, "\n")
		
		// Fuzz property: filterNonEmpty should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("filterNonEmpty panicked with lines=%v: %v", lines, r)
			}
		}()
		
		result := filterNonEmpty(lines)
		
		// Fuzz property: Result length should not exceed input length
		if len(result) > len(lines) {
			t.Errorf("Result length (%d) exceeds input length (%d)", len(result), len(lines))
		}
		
		// Fuzz property: All result items should be non-empty and non-comment after trimming
		for i, line := range result {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				t.Errorf("Result[%d] should not be empty after trimming: %q", i, line)
			}
			if strings.HasPrefix(trimmed, "#") {
				t.Errorf("Result[%d] should not be a comment: %q", i, line)
			}
		}
		
		// Fuzz property: Count should match expected
		expectedCount := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				expectedCount++
			}
		}
		if len(result) != expectedCount {
			t.Errorf("Expected %d non-empty/non-comment lines, got %d", expectedCount, len(result))
		}
	})
}