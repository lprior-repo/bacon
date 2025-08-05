package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	common "bacon/src/shared"
)

func TestMain(m *testing.M) {
	// Setup test environment
	m.Run()
}

// Test HandleRequest function with comprehensive scenarios
func TestHandleRequest(t *testing.T) {
	ctx, cleanup := common.TestContext("query-resolver-test")
	defer cleanup()

	testCases := []struct {
		name           string
		event          AppSyncEvent
		expectError    bool
		expectedResult bool // true if result should be non-nil
	}{
		{
			name: "getResource valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "getResource",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{
					"id": "test-resource-123",
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "getResourcesByConfidence valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "getResourcesByConfidence",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{
					"minConfidence": 0.8,
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "getConflictedRelationships valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "getConflictedRelationships",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "getRelationshipsBySource valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "getRelationshipsBySource",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{
					"source": "github-codeowners",
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "searchResources valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "searchResources",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{
					"text": "test-search",
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "searchResourcesByOwner valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "searchResourcesByOwner",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{
					"owner": "test-owner",
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "getOwnershipCoverage valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "getOwnershipCoverage",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "getConfidenceDistribution valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "getConfidenceDistribution",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "unknown field",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "unknownField",
					ParentTypeName: "Query",
				},
				Arguments: map[string]interface{}{},
			},
			expectError:    true,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := HandleRequest(ctx, tc.event)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tc.expectedResult && result == nil {
				t.Error("Expected non-nil result")
			}

			if !tc.expectedResult && tc.expectError && result != nil {
				t.Error("Expected nil result for error case")
			}
		})
	}
}

// Test handleGetResource with comprehensive edge cases
func TestHandleGetResource(t *testing.T) {
	ctx, cleanup := common.TestContext("get-resource-test")
	defer cleanup()

	testCases := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid resource ID",
			args: map[string]interface{}{
				"id": "valid-resource-123",
			},
			expectError: false,
		},
		{
			name: "empty resource ID",
			args: map[string]interface{}{
				"id": "",
			},
			expectError: false, // Function doesn't validate, just uses the value
		},
		{
			name: "very long resource ID",
			args: map[string]interface{}{
				"id": strings.Repeat("a", 1000),
			},
			expectError: false,
		},
		{
			name: "resource ID with special characters",
			args: map[string]interface{}{
				"id": "resource-with-@#$%^&*()_+-=",
			},
			expectError: false,
		},
		{
			name: "unicode resource ID",
			args: map[string]interface{}{
				"id": "资源-123-テスト",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource, err := handleGetResource(ctx, tc.args)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resource == nil {
					t.Error("Expected non-nil result")
				}
				if resource != nil {
					// Validate resource structure
					if resource.ID != tc.args["id"].(string) {
						t.Errorf("Expected ID %s, got %s", tc.args["id"].(string), resource.ID)
					}
					if resource.Name == "" {
						t.Error("Resource name should not be empty")
					}
					if resource.Type == "" {
						t.Error("Resource type should not be empty")
					}
					if len(resource.Relationships) == 0 {
						t.Error("Resource should have at least one relationship")
					}
				}
			}
		})
	}
}

// Test handleGetResourcesByConfidence with boundary conditions
func TestHandleGetResourcesByConfidence(t *testing.T) {
	ctx, cleanup := common.TestContext("get-resources-by-confidence-test")
	defer cleanup()

	testCases := []struct {
		name           string
		args           map[string]interface{}
		expectError    bool
		expectedResult bool
	}{
		{
			name: "minimum confidence 0.0",
			args: map[string]interface{}{
				"minConfidence": 0.0,
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "maximum confidence 1.0",
			args: map[string]interface{}{
				"minConfidence": 1.0,
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "medium confidence 0.5",
			args: map[string]interface{}{
				"minConfidence": 0.5,
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "high confidence 0.9",
			args: map[string]interface{}{
				"minConfidence": 0.9,
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "confidence above maximum 1.5",
			args: map[string]interface{}{
				"minConfidence": 1.5,
			},
			expectError:    false, // Function doesn't validate
			expectedResult: true,
		},
		{
			name: "negative confidence -0.1",
			args: map[string]interface{}{
				"minConfidence": -0.1,
			},
			expectError:    false, // Function doesn't validate
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resources, err := handleGetResourcesByConfidence(ctx, tc.args)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tc.expectedResult && resources == nil {
				t.Error("Expected non-nil result")
			}

			if resources != nil {
				// Validate result structure
				for _, resource := range resources {
					if resource.Name == "" {
						t.Error("Resource name should not be empty")
					}
					if len(resource.Relationships) == 0 {
						t.Error("Resource should have relationships")
					}
					for _, rel := range resource.Relationships {
						if rel.Confidence < 0 || rel.Confidence > 1 {
							t.Errorf("Relationship confidence %f should be between 0 and 1", rel.Confidence)
						}
					}
				}
			}
		})
	}
}

// Test handleGetConflictedRelationships
func TestHandleGetConflictedRelationships(t *testing.T) {
	ctx, cleanup := common.TestContext("get-conflicted-relationships-test")
	defer cleanup()

	// Test with empty arguments
	relationships, err := handleGetConflictedRelationships(ctx, map[string]interface{}{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if relationships == nil {
		t.Error("Expected non-nil result")
	}
	for _, rel := range relationships {
		if !rel.HasConflict {
			t.Error("All relationships from this handler should have conflicts")
		}
		if rel.ConfidenceLevel != "DISPUTED" {
			t.Error("Conflicted relationships should have DISPUTED confidence level")
		}
		if rel.From == "" || rel.To == "" {
			t.Error("Relationship From and To should not be empty")
		}
	}
}

// Test handleGetRelationshipsBySource
func TestHandleGetRelationshipsBySource(t *testing.T) {
	ctx, cleanup := common.TestContext("get-relationships-by-source-test")
	defer cleanup()

	testCases := []struct {
		name   string
		source string
	}{
		{"github-codeowners source", "github-codeowners"},
		{"openshift-metadata source", "openshift-metadata"},
		{"aws-tags source", "aws-tags"},
		{"empty source", ""},
		{"unknown source", "unknown-source"},
		{"source with special chars", "source-@#$%"},
		{"unicode source", "源代码-소스"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"source": tc.source,
			}

			relationships, err := handleGetRelationshipsBySource(ctx, args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if relationships == nil {
				t.Error("Expected non-nil result")
			}
			for _, rel := range relationships {
				if rel.Source != tc.source {
					t.Errorf("Expected source %s, got %s", tc.source, rel.Source)
				}
				if rel.From == "" || rel.To == "" {
					t.Error("Relationship From and To should not be empty")
				}
				if rel.Type == "" {
					t.Error("Relationship Type should not be empty")
				}
			}
		})
	}
}

// Test handleSearchResources
func TestHandleSearchResources(t *testing.T) {
	ctx, cleanup := common.TestContext("search-resources-test")
	defer cleanup()

	testCases := []struct {
		name       string
		searchText string
	}{
		{"simple search", "test"},
		{"empty search", ""},
		{"long search text", strings.Repeat("search", 100)},
		{"search with spaces", "test search query"},
		{"search with special chars", "test@#$%^&*()"},
		{"unicode search", "测试搜索テスト"},
		{"numeric search", "12345"},
		{"mixed search", "test-123-query"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"text": tc.searchText,
			}

			resources, err := handleSearchResources(ctx, args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if resources == nil {
				t.Error("Expected non-nil result")
			}
			for _, resource := range resources {
				// Validate that search text is reflected in the result
				if tc.searchText != "" {
					expectedName := fmt.Sprintf("service-matching-%s", strings.ToLower(tc.searchText))
					if resource.Name != expectedName {
						t.Errorf("Expected name to contain search text, got %s", resource.Name)
					}
					expectedDesc := fmt.Sprintf("Service that matches search term: %s", tc.searchText)
					if resource.Description != expectedDesc {
						t.Errorf("Expected description to contain search text, got %s", resource.Description)
					}
				}
			}
		})
	}
}

// Test handleSearchResourcesByOwner
func TestHandleSearchResourcesByOwner(t *testing.T) {
	ctx, cleanup := common.TestContext("search-resources-by-owner-test")
	defer cleanup()

	testCases := []struct {
		name  string
		owner string
	}{
		{"simple owner", "test-owner"},
		{"empty owner", ""},
		{"owner with spaces", "test owner"},
		{"owner with special chars", "owner@#$%"},
		{"unicode owner", "所有者テスト"},
		{"numeric owner", "12345"},
		{"long owner name", strings.Repeat("owner", 50)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"owner": tc.owner,
			}

			resources, err := handleSearchResourcesByOwner(ctx, args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if resources == nil {
				t.Error("Expected non-nil result")
			}
			for _, resource := range resources {
				// Validate that owner is reflected in the result
				expectedName := fmt.Sprintf("%s-owned-service", tc.owner)
				if resource.Name != expectedName {
					t.Errorf("Expected name %s, got %s", expectedName, resource.Name)
				}
				expectedDesc := fmt.Sprintf("Service owned by %s", tc.owner)
				if resource.Description != expectedDesc {
					t.Errorf("Expected description %s, got %s", expectedDesc, resource.Description)
				}
			}
		})
	}
}

// Test handleGetOwnershipCoverage
func TestHandleGetOwnershipCoverage(t *testing.T) {
	ctx, cleanup := common.TestContext("get-ownership-coverage-test")
	defer cleanup()

	stats, err := handleGetOwnershipCoverage(ctx, map[string]interface{}{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil result")
	}

	// Validate ownership statistics structure
	if stats.TotalResources <= 0 {
		t.Error("Total resources should be positive")
	}

	if stats.OwnedResources < 0 {
		t.Error("Owned resources should not be negative")
	}

	if stats.UnownedResources < 0 {
		t.Error("Unowned resources should not be negative")
	}

	if stats.TotalResources != stats.OwnedResources+stats.UnownedResources {
		t.Error("Total resources should equal owned + unowned")
	}

	if stats.CoveragePercentage < 0 || stats.CoveragePercentage > 100 {
		t.Errorf("Coverage percentage %f should be between 0 and 100", stats.CoveragePercentage)
	}

	// Validate resource type stats
	for _, typeStats := range stats.ByResourceType {
		if typeStats.Type == "" {
			t.Error("Resource type should not be empty")
		}
		if typeStats.Total < 0 {
			t.Error("Total should not be negative")
		}
		if typeStats.Owned < 0 {
			t.Error("Owned should not be negative")
		}
		if typeStats.Coverage < 0 || typeStats.Coverage > 100 {
			t.Errorf("Coverage %f should be between 0 and 100", typeStats.Coverage)
		}
	}

	// Validate team stats
	for _, teamStats := range stats.ByTeam {
		if teamStats.Team == "" {
			t.Error("Team name should not be empty")
		}
		if teamStats.ResourceCount < 0 {
			t.Error("Resource count should not be negative")
		}
		if teamStats.AverageConfidence < 0 || teamStats.AverageConfidence > 1 {
			t.Errorf("Average confidence %f should be between 0 and 1", teamStats.AverageConfidence)
		}
	}
}

// Test handleGetConfidenceDistribution
func TestHandleGetConfidenceDistribution(t *testing.T) {
	ctx, cleanup := common.TestContext("get-confidence-distribution-test")
	defer cleanup()

	stats, err := handleGetConfidenceDistribution(ctx, map[string]interface{}{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil result")
	}

	// Validate confidence statistics structure
	if stats.High < 0 {
		t.Error("High confidence count should not be negative")
	}

	if stats.Medium < 0 {
		t.Error("Medium confidence count should not be negative")
	}

	if stats.Low < 0 {
		t.Error("Low confidence count should not be negative")
	}

	if stats.VeryLow < 0 {
		t.Error("Very low confidence count should not be negative")
	}

	if stats.AverageConfidence < 0 || stats.AverageConfidence > 1 {
		t.Errorf("Average confidence %f should be between 0 and 1", stats.AverageConfidence)
	}

	// Validate distribution by source
	for _, sourceStats := range stats.DistributionBySource {
		if sourceStats.Source == "" {
			t.Error("Source should not be empty")
		}
		if sourceStats.Count < 0 {
			t.Error("Count should not be negative")
		}
		if sourceStats.AverageConfidence < 0 || sourceStats.AverageConfidence > 1 {
			t.Errorf("Average confidence %f should be between 0 and 1", sourceStats.AverageConfidence)
		}
	}
}

// Property-based tests using rapid testing approach
func TestPropertyBasedAppSyncEventHandling(t *testing.T) {
	ctx, cleanup := common.TestContext("property-based-appsync-test")
	defer cleanup()

	// Test property: all valid field names should return non-nil results
	validFieldNames := []string{
		"getResource", "getResourcesByConfidence", "getConflictedRelationships",
		"getRelationshipsBySource", "searchResources", "searchResourcesByOwner",
		"getOwnershipCoverage", "getConfidenceDistribution",
	}

	for i := 0; i < 100; i++ {
		fieldName := validFieldNames[rand.Intn(len(validFieldNames))]
		
		// Generate random arguments based on field
		args := generateRandomArgsForField(fieldName)
		
		event := AppSyncEvent{
			Info: RequestInfo{
				FieldName:      fieldName,
				ParentTypeName: "Query",
			},
			Arguments: args,
		}

		result, err := HandleRequest(ctx, event)

		if err != nil {
			t.Errorf("Property violated: valid field %s returned error: %v", fieldName, err)
		}

		if result == nil {
			t.Errorf("Property violated: valid field %s returned nil result", fieldName)
		}
	}
}

// Test property: invalid field names should always return errors
func TestPropertyBasedInvalidFieldNames(t *testing.T) {
	ctx, cleanup := common.TestContext("property-based-invalid-field-test")
	defer cleanup()

	for i := 0; i < 50; i++ {
		// Generate random invalid field names
		invalidFieldName := fmt.Sprintf("invalid-%d-%s", i, generateRandomString(10))
		
		event := AppSyncEvent{
			Info: RequestInfo{
				FieldName:      invalidFieldName,
				ParentTypeName: "Query",
			},
			Arguments: map[string]interface{}{},
		}

		result, err := HandleRequest(ctx, event)

		if err == nil {
			t.Errorf("Property violated: invalid field %s should return error", invalidFieldName)
		}

		if result != nil {
			t.Errorf("Property violated: invalid field %s should return nil result", invalidFieldName)
		}
	}
}

// Test property: confidence values should always be between 0 and 1
func TestPropertyBasedConfidenceValues(t *testing.T) {
	ctx, cleanup := common.TestContext("property-based-confidence-test")
	defer cleanup()

	for i := 0; i < 100; i++ {
		// Generate random confidence values
		minConfidence := rand.Float64() * 2 - 0.5 // Range: -0.5 to 1.5
		
		args := map[string]interface{}{
			"minConfidence": minConfidence,
		}

		resources, err := handleGetResourcesByConfidence(ctx, args)

		if err != nil {
			t.Errorf("Unexpected error for confidence %f: %v", minConfidence, err)
		}

		// Verify all returned relationships have valid confidences
		if resources != nil {
			for _, resource := range resources {
				for _, rel := range resource.Relationships {
					if rel.Confidence < 0 || rel.Confidence > 1 {
						t.Errorf("Property violated: relationship confidence %f should be between 0 and 1", rel.Confidence)
					}
				}
			}
		}
	}
}

// Defensive programming tests
func TestDefensiveProgramming(t *testing.T) {
	ctx, cleanup := common.TestContext("defensive-programming-test")
	defer cleanup()

	t.Run("HandleRequest with nil context", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function should handle nil context gracefully, but panicked: %v", r)
			}
		}()

		event := AppSyncEvent{
			Info: RequestInfo{
				FieldName: "getResource",
			},
			Arguments: map[string]interface{}{"id": "test"},
		}

		// This might return an error but shouldn't panic
		_, err := HandleRequest(nil, event)
		if err != nil {
			t.Logf("Expected behavior with nil context: %v", err)
		}
	})

	t.Run("getResource with missing arguments", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function should handle missing arguments gracefully, but panicked: %v", r)
			}
		}()

		// Missing "id" argument should be handled gracefully
		_, err := handleGetResource(ctx, map[string]interface{}{})
		if err != nil {
			t.Logf("Expected behavior with missing arguments: %v", err)
		}
	})

	t.Run("confidence with wrong type", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function should handle wrong argument types gracefully, but panicked: %v", r)
			}
		}()

		// Wrong type for minConfidence
		args := map[string]interface{}{
			"minConfidence": "not-a-number",
		}

		_, err := handleGetResourcesByConfidence(ctx, args)
		if err != nil {
			t.Logf("Expected behavior with wrong argument type: %v", err)
		}
	})
}

// Mutation testing scenarios
func TestMutationTargets(t *testing.T) {
	ctx, cleanup := common.TestContext("mutation-targets-test")
	defer cleanup()

	// Test critical mutations that could break functionality

	t.Run("empty resource ID should not break getResource", func(t *testing.T) {
		args := map[string]interface{}{
			"id": "",
		}

		result, err := handleGetResource(ctx, args)

		// Should handle empty ID gracefully
		if err != nil {
			t.Logf("Handling empty ID: %v", err)
		}
		if result != nil && result.ID != "" {
			t.Error("Empty resource ID should result in empty ID in response")
		}
	})

	t.Run("zero confidence should not break confidence filtering", func(t *testing.T) {
		args := map[string]interface{}{
			"minConfidence": 0.0,
		}

		result, err := handleGetResourcesByConfidence(ctx, args)

		if err != nil {
			t.Errorf("Zero confidence should not cause error: %v", err)
		}
		if result == nil {
			t.Error("Zero confidence should return valid result")
		}
	})

	t.Run("search with empty text should not break", func(t *testing.T) {
		args := map[string]interface{}{
			"text": "",
		}

		result, err := handleSearchResources(ctx, args)

		if err != nil {
			t.Errorf("Empty search text should not cause error: %v", err)
		}
		if result == nil {
			t.Error("Empty search should return valid result")
		}
	})
}

// Benchmark tests for performance
func BenchmarkHandleRequest(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-test")
	defer cleanup()

	event := AppSyncEvent{
		Info: RequestInfo{
			FieldName: "getResource",
		},
		Arguments: map[string]interface{}{
			"id": "benchmark-resource",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HandleRequest(ctx, event)
	}
}

func BenchmarkGetOwnershipCoverage(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-coverage-test")
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handleGetOwnershipCoverage(ctx, map[string]interface{}{})
	}
}

// Helper functions for property-based testing
func generateRandomArgsForField(fieldName string) map[string]interface{} {
	switch fieldName {
	case "getResource":
		return map[string]interface{}{
			"id": generateRandomString(20),
		}
	case "getResourcesByConfidence":
		return map[string]interface{}{
			"minConfidence": rand.Float64(),
		}
	case "getRelationshipsBySource":
		sources := []string{"github-codeowners", "openshift-metadata", "aws-tags", "datadog-metrics"}
		return map[string]interface{}{
			"source": sources[rand.Intn(len(sources))],
		}
	case "searchResources":
		return map[string]interface{}{
			"text": generateRandomString(15),
		}
	case "searchResourcesByOwner":
		return map[string]interface{}{
			"owner": generateRandomString(10),
		}
	default:
		return map[string]interface{}{}
	}
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}