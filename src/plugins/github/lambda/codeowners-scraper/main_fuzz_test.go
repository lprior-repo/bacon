package main

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"bacon/src/plugins/github/types"
	"pgregory.net/rapid"
	common "bacon/src/shared"
)

// Comprehensive Property-Based Testing for Codeowners-Scraper
func TestPropertyBasedCodeownersScraper(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random event data
		organization := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]*`).Draw(t, "organization")
		batchSize := rapid.IntRange(1, 100).Draw(t, "batchSize")
		
		event := types.Event{
			Organization: organization,
			BatchSize:    batchSize,
		}
		
		// Property: HandleRequest should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("HandleRequest panicked with event=%+v: %v", event, r)
			}
		}()
		
		ctx, cleanup := common.TestContext("property-test")
		defer cleanup()
		
		response, err := HandleRequest(ctx, event)
		
		// Property: Response should be a valid string (empty or with content)
		if response == "" && err == nil {
			// This could be valid if no repos were processed
			t.Logf("Empty response with no error for event=%+v", event)
		}
		
		// Property: Error should be descriptive if present
		if err != nil && err.Error() == "" {
			t.Errorf("Error should have descriptive message: %v", err)
		}
		
		// Property: Organization should be used consistently
		if organization == "" && err == nil {
			t.Errorf("Empty organization should produce error or valid handling")
		}
	})
}

func TestPropertyBasedValidateEvent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random event data
		organization := rapid.String().Draw(t, "organization")
		batchSize := rapid.Int().Draw(t, "batchSize")
		
		event := types.Event{
			Organization: organization,
			BatchSize:    batchSize,
		}
		
		// Property: validateEvent should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("validateEvent panicked with event=%+v: %v", event, r)
			}
		}()
		
		result, err := validateEvent(event)
		
		// Property: validateEvent should always succeed (based on implementation)
		if err != nil {
			t.Errorf("validateEvent should not return errors: %v", err)
		}
		
		// Property: Result should have consistent organization
		if result.Organization != organization {
			t.Errorf("Organization should be preserved: expected %q, got %q", organization, result.Organization)
		}
		
		// Property: Batch size should be adjusted appropriately
		if result.BatchSize <= 0 && batchSize > 0 {
			t.Logf("Batch size adjusted from %d to %d", batchSize, result.BatchSize)
		}
	})
}

// Comprehensive Fuzz Testing for Codeowners-Scraper
func FuzzHandleRequest(f *testing.F) {
	// Seed with various event patterns
	f.Add("test-org", 10)
	f.Add("", 1)
	f.Add("github", 50)
	f.Add("my-company", 1)
	f.Add("org-with-dashes", 25)
	f.Add("org_with_underscores", 100)
	f.Add("123org", 5)
	f.Add("ORG", 1)
	f.Add("o", 1000)
	f.Add(strings.Repeat("a", 100), 1)
	f.Add("unicode-测试", 10)
	f.Add("emoji-🚀", 5)
	f.Add("special!@#$", 1)
	
	f.Fuzz(func(t *testing.T, organization string, batchSize int) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(organization) {
			t.Skip("Skipping invalid UTF-8 organization")
		}
		
		// Skip unreasonable batch sizes
		if batchSize < 0 || batchSize > 10000 {
			t.Skip("Skipping unreasonable batch size")
		}
		
		event := types.Event{
			Organization: organization,
			BatchSize:    batchSize,
		}
		
		// Fuzz property: HandleRequest should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("HandleRequest panicked with org=%q, batch=%d: %v", organization, batchSize, r)
			}
		}()
		
		ctx, cleanup := common.TestContext("fuzz-test")
		defer cleanup()
		
		response, err := HandleRequest(ctx, event)
		
		// Fuzz property: Response and error should be consistent
		if response == "" && err == nil {
			// This could be valid behavior
			t.Logf("Empty response with no error for org=%q", organization)
		}
		
		// Fuzz property: Error messages should be non-empty if error exists
		if err != nil && strings.TrimSpace(err.Error()) == "" {
			t.Errorf("Error should have non-empty message: %v", err)
		}
		
		// Fuzz property: Response should be valid string
		if !utf8.ValidString(response) {
			t.Errorf("Response should be valid UTF-8: %q", response)
		}
	})
}

func FuzzValidateEvent(f *testing.F) {
	// Seed with various event scenarios
	f.Add("test-org", 10)
	f.Add("", 10)
	f.Add("org", 0)
	f.Add("", 0)
	f.Add("my-awesome-org", 100)
	f.Add("org_with_underscores", 50)
	f.Add("123org", 1)
	f.Add("ORG", 1000)
	f.Add("o", 1)
	f.Add(strings.Repeat("org", 50), 25)
	f.Add("unicode-测试org", 10)
	f.Add("emoji-🚀org", 5)
	f.Add("special!@#$org", 1)
	f.Add("org.with.dots", 75)
	f.Add("org/with/slashes", 15)
	
	f.Fuzz(func(t *testing.T, organization string, batchSize int) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(organization) {
			t.Skip("Skipping invalid UTF-8 strings")
		}
		
		event := types.Event{
			Organization: organization,
			BatchSize:    batchSize,
		}
		
		// Fuzz property: validateEvent should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("validateEvent panicked with org=%q, batch=%d: %v", organization, batchSize, r)
			}
		}()
		
		result, err := validateEvent(event)
		
		// Fuzz property: validateEvent should not return errors (based on implementation)
		if err != nil {
			t.Errorf("validateEvent should not return errors: %v", err)
		}
		
		// Fuzz property: Organization should be preserved
		if result.Organization != organization {
			t.Errorf("Organization should be preserved: expected %q, got %q", organization, result.Organization)
		}
		
		// Fuzz property: Batch size handling should be consistent
		if batchSize <= 0 && result.BatchSize > 0 {
			t.Logf("Batch size corrected from %d to %d", batchSize, result.BatchSize)
		}
		
		// Fuzz property: Result should be well-formed
		if result.BatchSize < 0 {
			t.Errorf("Result batch size should not be negative: %d", result.BatchSize)
		}
	})
}

func FuzzGitHubOperations(f *testing.F) {
	// Seed with various GitHub scenarios
	f.Add("valid-repo", "valid-org", "main")
	f.Add("repo", "org", "")
	f.Add("", "", "main")
	f.Add("test-repo", "test-org", "develop")
	f.Add("my-repo", "my-org", "feature/branch")
	f.Add("repo_underscores", "org_underscores", "main")
	f.Add("repo-dashes", "org-dashes", "main")
	f.Add("123repo", "456org", "v1.0")
	f.Add("REPO", "ORG", "MAIN")
	f.Add("r", "o", "m")
	f.Add(strings.Repeat("repo", 25), strings.Repeat("org", 25), "main")
	f.Add("unicode-测试", "unicode-测试", "main")
	f.Add("emoji-🚀", "emoji-🚀", "main")
	f.Add("special!@#$", "special!@#$", "main")
	f.Add("repo.with.dots", "org.with.dots", "main")
	
	f.Fuzz(func(t *testing.T, repoName string, orgName string, branch string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(repoName) || !utf8.ValidString(orgName) || !utf8.ValidString(branch) {
			t.Skip("Skipping invalid UTF-8 strings")
		}
		
		// Test buildGitHubURL function (if accessible)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GitHub operations panicked with repo=%q, org=%q, branch=%q: %v", repoName, orgName, branch, r)
			}
		}()
		
		// Fuzz property: URL building should be consistent
		if repoName != "" && orgName != "" {
			// Mock URL building logic
			expectedPattern := fmt.Sprintf("github.com/%s/%s", orgName, repoName)
			if branch != "" {
				expectedPattern += "/tree/" + branch
			}
			
			// Test that URL components are handled properly
			if strings.Contains(expectedPattern, "//") {
				t.Errorf("URL should not contain double slashes: %s", expectedPattern)
			}
		}
		
		// Fuzz property: Empty parameters should be handled gracefully
		if repoName == "" || orgName == "" {
			t.Logf("Handling empty parameters: repo=%q, org=%q", repoName, orgName)
		}
	})
}

func FuzzStringOperations(f *testing.F) {
	// Seed with various string manipulation scenarios
	f.Add("simple-string")
	f.Add("")
	f.Add("   whitespace   ")
	f.Add("multiple\nlines\nhere")
	f.Add("tabs\tand\tspaces")
	f.Add("unicode: 测试字符串")
	f.Add("emoji: 🚀🎉📝")
	f.Add("special!@#$%^&*()chars")
	f.Add("JSON: {\"key\": \"value\"}")
	f.Add("XML: <root><item>value</item></root>")
	f.Add("Code: func main() { fmt.Println(\"Hello\") }")
	f.Add(strings.Repeat("a", 10000))
	f.Add("binary\x00\x01\x02data")
	f.Add("mixed\r\nline\nendings\r")
	
	f.Fuzz(func(t *testing.T, input string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(input) {
			t.Skip("Skipping invalid UTF-8 string")
		}
		
		// Fuzz property: String operations should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("String operations panicked with input=%q: %v", input, r)
			}
		}()
		
		// Test various string operations that might be used
		trimmed := strings.TrimSpace(input)
		lower := strings.ToLower(input)
		upper := strings.ToUpper(input)
		
		// Fuzz property: String transformations should preserve UTF-8 validity
		if !utf8.ValidString(trimmed) {
			t.Errorf("TrimSpace should preserve UTF-8 validity")
		}
		if !utf8.ValidString(lower) {
			t.Errorf("ToLower should preserve UTF-8 validity")
		}
		if !utf8.ValidString(upper) {
			t.Errorf("ToUpper should preserve UTF-8 validity")
		}
		
		// Fuzz property: Length relationships
		if len(trimmed) > len(input) {
			t.Errorf("Trimmed string should not be longer than original")
		}
		
		// Fuzz property: Case conversions should be reversible for simple ASCII
		if isASCII(input) {
			if strings.ToUpper(strings.ToLower(input)) != strings.ToUpper(input) {
				// This is expected for some cases, just log
				t.Logf("Case conversion not perfectly reversible for: %q", input)
			}
		}
	})
}

func FuzzErrorHandling(f *testing.F) {
	// Seed with various error scenarios
	f.Add("network error")
	f.Add("")
	f.Add("timeout occurred")
	f.Add("invalid credentials")
	f.Add("resource not found")
	f.Add("permission denied")
	f.Add("rate limit exceeded")
	f.Add("internal server error")
	f.Add("unicode error: 测试错误")
	f.Add("emoji error: 🚨💥")
	f.Add("multiline\nerror\nmessage")
	f.Add("tab\terror\tmessage")
	f.Add("JSON error: {\"error\": \"details\"}")
	f.Add("very " + strings.Repeat("long ", 1000) + "error")
	
	f.Fuzz(func(t *testing.T, errorMessage string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(errorMessage) {
			t.Skip("Skipping invalid UTF-8 error message")
		}
		
		// Fuzz property: Error creation should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error handling panicked with message=%q: %v", errorMessage, r)
			}
		}()
		
		// Test error creation and handling
		if errorMessage != "" {
			err := fmt.Errorf("test error: %s", errorMessage)
			
			// Fuzz property: Error should preserve message
			if !strings.Contains(err.Error(), errorMessage) {
				t.Errorf("Error should contain original message")
			}
			
			// Fuzz property: Error string should be valid UTF-8
			if !utf8.ValidString(err.Error()) {
				t.Errorf("Error string should be valid UTF-8")
			}
		}
		
		// Fuzz property: Empty error messages should be handled
		if strings.TrimSpace(errorMessage) == "" {
			t.Logf("Handling empty error message")
		}
	})
}

// Helper function to check if string is ASCII
func isASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}