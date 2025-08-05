package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	common "bacon/src/shared"
)

func TestMain(m *testing.M) {
	// Setup test environment
	m.Run()
}

// Test HandleRequest function with comprehensive scenarios
func TestHandleRequest(t *testing.T) {
	ctx, cleanup := common.TestContext("mutation-resolver-test")
	defer cleanup()

	testCases := []struct {
		name           string
		event          AppSyncEvent
		expectError    bool
		expectedResult bool // true if result should be non-nil
	}{
		{
			name: "createRelationship valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "createRelationship",
					ParentTypeName: "Mutation",
				},
				Arguments: map[string]interface{}{
					"input": map[string]interface{}{
						"fromUserId":   "user-123",
						"toResourceId": "resource-456",
						"type":         "OWNS",
						"confidence":   0.8,
						"source":       "manual",
					},
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "updateRelationshipConfidence valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "updateRelationshipConfidence",
					ParentTypeName: "Mutation",
				},
				Arguments: map[string]interface{}{
					"id":         "rel-123",
					"confidence": 0.9,
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "resolveConflict valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "resolveConflict",
					ParentTypeName: "Mutation",
				},
				Arguments: map[string]interface{}{
					"id":       "conflict-123",
					"winnerId": "rel-456",
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "approveRelationships valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "approveRelationships",
					ParentTypeName: "Mutation",
				},
				Arguments: map[string]interface{}{
					"ids": []interface{}{"rel-1", "rel-2", "rel-3"},
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "rejectRelationships valid request",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "rejectRelationships",
					ParentTypeName: "Mutation",
				},
				Arguments: map[string]interface{}{
					"ids": []interface{}{"rel-1", "rel-2"},
				},
			},
			expectError:    false,
			expectedResult: true,
		},
		{
			name: "unknown field",
			event: AppSyncEvent{
				Info: RequestInfo{
					FieldName:      "unknownMutation",
					ParentTypeName: "Mutation",
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

// Test handleCreateRelationship with comprehensive edge cases
func TestHandleCreateRelationship(t *testing.T) {
	ctx, cleanup := common.TestContext("create-relationship-test")
	defer cleanup()

	testCases := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "complete relationship input",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "resource-456",
					"type":         "OWNS",
					"confidence":   0.8,
					"source":       "manual",
					"metadata":     map[string]interface{}{"key": "value"},
				},
			},
			expectError: false,
		},
		{
			name: "minimal relationship input",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "resource-456",
					"type":         "OWNS",
				},
			},
			expectError: false,
		},
		{
			name: "relationship with zero confidence",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "resource-456",
					"type":         "OWNS",
					"confidence":   0.0,
				},
			},
			expectError: false,
		},
		{
			name: "relationship with maximum confidence",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "resource-456",
					"type":         "OWNS",
					"confidence":   1.0,
				},
			},
			expectError: false,
		},
		{
			name: "relationship with confidence above 1",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "resource-456",
					"type":         "OWNS",
					"confidence":   1.5,
				},
			},
			expectError: false, // Function doesn't validate bounds
		},
		{
			name: "relationship with negative confidence",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "resource-456",
					"type":         "OWNS",
					"confidence":   -0.1,
				},
			},
			expectError: false, // Function doesn't validate bounds
		},
		{
			name: "empty user ID",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "",
					"toResourceId": "resource-456",
					"type":         "OWNS",
				},
			},
			expectError: false,
		},
		{
			name: "empty resource ID",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user-123",
					"toResourceId": "",
					"type":         "OWNS",
				},
			},
			expectError: false,
		},
		{
			name: "unicode characters in IDs",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "用户-123",
					"toResourceId": "リソース-456",
					"type":         "OWNS",
				},
			},
			expectError: false,
		},
		{
			name: "special characters in IDs",
			args: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user@#$%^&*()",
					"toResourceId": "resource-with-dashes_and_underscores",
					"type":         "OWNS",
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handleCreateRelationship(ctx, tc.args)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected non-nil result")
				}
				if result != nil {
					// Validate relationship structure
					if result.ID == "" {
						t.Error("Relationship ID should not be empty")
					}
					if result.Type == "" {
						t.Error("Relationship type should not be empty")
					}
					if result.ConfidenceLevel == "" {
						t.Error("Confidence level should not be empty")
					}
					if result.CreatedAt == "" {
						t.Error("CreatedAt should not be empty")
					}
					if result.UpdatedAt == "" {
						t.Error("UpdatedAt should not be empty")
					}
					// Validate timestamp format
					if _, err := time.Parse(time.RFC3339, result.CreatedAt); err != nil {
						t.Errorf("Invalid CreatedAt timestamp format: %s", result.CreatedAt)
					}
					if _, err := time.Parse(time.RFC3339, result.UpdatedAt); err != nil {
						t.Errorf("Invalid UpdatedAt timestamp format: %s", result.UpdatedAt)
					}
				}
			}
		})
	}
}

// Test handleUpdateRelationshipConfidence with boundary conditions
func TestHandleUpdateRelationshipConfidence(t *testing.T) {
	ctx, cleanup := common.TestContext("update-relationship-confidence-test")
	defer cleanup()

	testCases := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid confidence update",
			args: map[string]interface{}{
				"id":         "rel-123",
				"confidence": 0.85,
			},
			expectError: false,
		},
		{
			name: "minimum confidence",
			args: map[string]interface{}{
				"id":         "rel-123",
				"confidence": 0.0,
			},
			expectError: false,
		},
		{
			name: "maximum confidence",
			args: map[string]interface{}{
				"id":         "rel-123",
				"confidence": 1.0,
			},
			expectError: false,
		},
		{
			name: "confidence above maximum",
			args: map[string]interface{}{
				"id":         "rel-123",
				"confidence": 1.5,
			},
			expectError: false, // Function doesn't validate
		},
		{
			name: "negative confidence",
			args: map[string]interface{}{
				"id":         "rel-123",
				"confidence": -0.1,
			},
			expectError: false, // Function doesn't validate
		},
		{
			name: "empty relationship ID",
			args: map[string]interface{}{
				"id":         "",
				"confidence": 0.8,
			},
			expectError: false,
		},
		{
			name: "very long relationship ID",
			args: map[string]interface{}{
				"id":         strings.Repeat("a", 1000),
				"confidence": 0.8,
			},
			expectError: false,
		},
		{
			name: "unicode relationship ID",
			args: map[string]interface{}{
				"id":         "関係-123-тест",
				"confidence": 0.8,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handleUpdateRelationshipConfidence(ctx, tc.args)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected non-nil result")
				}
				if result != nil {
					// Validate updated relationship
					if result.ID != tc.args["id"].(string) {
						t.Errorf("Expected ID %s, got %s", tc.args["id"].(string), result.ID)
					}
					expectedConfidence := tc.args["confidence"].(float64)
					if result.Confidence != expectedConfidence {
						t.Errorf("Expected confidence %f, got %f", expectedConfidence, result.Confidence)
					}
					// Validate confidence level calculation
					expectedLevel := calculateConfidenceLevel(expectedConfidence)
					if result.ConfidenceLevel != expectedLevel {
						t.Errorf("Expected confidence level %s, got %s", expectedLevel, result.ConfidenceLevel)
					}
				}
			}
		})
	}
}

// Test handleResolveConflict
func TestHandleResolveConflict(t *testing.T) {
	ctx, cleanup := common.TestContext("resolve-conflict-test")
	defer cleanup()

	testCases := []struct {
		name       string
		conflictID string
		winnerID   string
	}{
		{"normal conflict resolution", "conflict-123", "rel-456"},
		{"empty conflict ID", "", "rel-456"},
		{"empty winner ID", "conflict-123", ""},
		{"both empty", "", ""},
		{"unicode IDs", "冲突-123", "获胜者-456"},
		{"special characters", "conflict@#$%", "winner_123"},
		{"very long IDs", strings.Repeat("c", 500), strings.Repeat("w", 500)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"id":       tc.conflictID,
				"winnerId": tc.winnerID,
			}

			result, err := handleResolveConflict(ctx, args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Error("Expected non-nil result")
			}

			if result != nil {
				// Validate resolved conflict
				if result.ID != tc.winnerID {
					t.Errorf("Expected winner ID %s, got %s", tc.winnerID, result.ID)
				}
				if result.HasConflict {
					t.Error("Resolved relationship should not have conflict flag set to false")
				}
				if result.Confidence != 0.95 {
					t.Errorf("Expected resolved confidence 0.95, got %f", result.Confidence)
				}
				if result.ConfidenceLevel != "VERY_HIGH" {
					t.Errorf("Expected confidence level VERY_HIGH, got %s", result.ConfidenceLevel)
				}
				if result.Source != "manual-resolution" {
					t.Errorf("Expected source manual-resolution, got %s", result.Source)
				}
			}
		})
	}
}

// Test handleApproveRelationships with various ID sets
func TestHandleApproveRelationships(t *testing.T) {
	ctx, cleanup := common.TestContext("approve-relationships-test")
	defer cleanup()

	testCases := []struct {
		name        string
		ids         []interface{}
		expectError bool
	}{
		{
			name:        "single relationship",
			ids:         []interface{}{"rel-1"},
			expectError: false,
		},
		{
			name:        "multiple relationships",
			ids:         []interface{}{"rel-1", "rel-2", "rel-3", "rel-4", "rel-5"},
			expectError: false,
		},
		{
			name:        "empty list",
			ids:         []interface{}{},
			expectError: false,
		},
		{
			name:        "duplicate IDs",
			ids:         []interface{}{"rel-1", "rel-1", "rel-2"},
			expectError: false,
		},
		{
			name:        "empty string IDs",
			ids:         []interface{}{"", "rel-2", ""},
			expectError: false,
		},
		{
			name:        "unicode IDs",
			ids:         []interface{}{"关系-1", "関係-2", "связь-3"},
			expectError: false,
		},
		{
			name:        "very long IDs",
			ids:         []interface{}{strings.Repeat("a", 1000), strings.Repeat("b", 500)},
			expectError: false,
		},
		{
			name:        "large number of IDs",
			ids:         generateIDList(1000),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"ids": tc.ids,
			}

			relationships, err := handleApproveRelationships(ctx, args)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if relationships == nil {
					t.Error("Expected non-nil result")
				}
				if len(relationships) != len(tc.ids) {
					t.Errorf("Expected %d relationships, got %d", len(tc.ids), len(relationships))
				}

				// Validate all approved relationships
				for i, rel := range relationships {
					if i < len(tc.ids) {
						expectedID := tc.ids[i].(string)
						if rel.ID != expectedID {
							t.Errorf("Expected ID %s, got %s", expectedID, rel.ID)
						}
					}
					if rel.Confidence != 0.90 {
						t.Errorf("Expected approved confidence 0.90, got %f", rel.Confidence)
					}
					if rel.ConfidenceLevel != "VERY_HIGH" {
						t.Errorf("Expected confidence level VERY_HIGH, got %s", rel.ConfidenceLevel)
					}
					if rel.Source != "manual-approval" {
						t.Errorf("Expected source manual-approval, got %s", rel.Source)
					}
					if rel.HasConflict {
						t.Error("Approved relationships should not have conflicts")
					}
				}
			}
		})
	}
}

// Test handleRejectRelationships
func TestHandleRejectRelationships(t *testing.T) {
	ctx, cleanup := common.TestContext("reject-relationships-test")
	defer cleanup()

	testCases := []struct {
		name string
		ids  []interface{}
	}{
		{"single relationship", []interface{}{"rel-1"}},
		{"multiple relationships", []interface{}{"rel-1", "rel-2", "rel-3"}},
		{"empty list", []interface{}{}},
		{"duplicate IDs", []interface{}{"rel-1", "rel-1"}},
		{"large list", generateIDList(500)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]interface{}{
				"ids": tc.ids,
			}

			relationships, err := handleRejectRelationships(ctx, args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if relationships == nil {
				t.Error("Expected non-nil result")
			}

			// Rejected relationships should return empty array
			if len(relationships) != 0 {
				t.Errorf("Expected empty array for rejected relationships, got %d", len(relationships))
			}
		})
	}
}

// Test calculateConfidenceLevel function with comprehensive boundary conditions
func TestCalculateConfidenceLevel(t *testing.T) {
	testCases := []struct {
		name       string
		confidence float64
		expected   string
	}{
		// Boundary conditions
		{"exactly 0.9", 0.9, "VERY_HIGH"},
		{"just below 0.9", 0.8999999, "HIGH"},
		{"exactly 0.8", 0.8, "HIGH"},
		{"just below 0.8", 0.7999999, "MEDIUM"},
		{"exactly 0.6", 0.6, "MEDIUM"},
		{"just below 0.6", 0.5999999, "LOW"},
		{"exactly 0.4", 0.4, "LOW"},
		{"just below 0.4", 0.3999999, "VERY_LOW"},
		
		// Edge cases
		{"minimum value 0.0", 0.0, "VERY_LOW"},
		{"maximum value 1.0", 1.0, "VERY_HIGH"},
		{"negative value", -0.1, "VERY_LOW"},
		{"above maximum", 1.5, "VERY_HIGH"},
		
		// Typical values
		{"very high confidence", 0.95, "VERY_HIGH"},
		{"high confidence", 0.85, "HIGH"},
		{"medium confidence", 0.7, "MEDIUM"},
		{"low confidence", 0.5, "LOW"},
		{"very low confidence", 0.2, "VERY_LOW"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateConfidenceLevel(tc.confidence)
			if result != tc.expected {
				t.Errorf("Expected confidence level %s for %f, got %s", tc.expected, tc.confidence, result)
			}
		})
	}
}

// Property-based tests using rapid testing approach
func TestPropertyBasedCreateRelationship(t *testing.T) {
	ctx, cleanup := common.TestContext("property-based-create-test")
	defer cleanup()

	// Test property: all created relationships should have valid structure
	for i := 0; i < 100; i++ {
		// Generate random relationship data
		fromUser := generateRandomString(20)
		toResource := generateRandomString(30)
		relType := []string{"OWNS", "MANAGES", "USES", "MAINTAINS"}[rand.Intn(4)]
		confidence := rand.Float64()
		source := []string{"manual", "github-codeowners", "aws-tags", "openshift"}[rand.Intn(4)]

		args := map[string]interface{}{
			"input": map[string]interface{}{
				"fromUserId":   fromUser,
				"toResourceId": toResource,
				"type":         relType,
				"confidence":   confidence,
				"source":       source,
			},
		}

		result, err := handleCreateRelationship(ctx, args)

		if err != nil {
			t.Errorf("Property violated: random valid input caused error: %v", err)
		}

		if result == nil {
			t.Error("Property violated: valid input returned nil result")
		}

		if result != nil {
			// Validate property: created relationship has all required fields
			if result.ID == "" {
				t.Error("Property violated: created relationship missing ID")
			}
			if result.From != fromUser {
				t.Errorf("Property violated: expected From %s, got %s", fromUser, result.From)
			}
			if result.To != toResource {
				t.Errorf("Property violated: expected To %s, got %s", toResource, result.To)
			}
			if result.Type != relType {
				t.Errorf("Property violated: expected Type %s, got %s", relType, result.Type)
			}
			if result.Source != source {
				t.Errorf("Property violated: expected Source %s, got %s", source, result.Source)
			}
		}
	}
}

// Test property: confidence levels should be consistent with confidence values
func TestPropertyBasedConfidenceLevelConsistency(t *testing.T) {
	for i := 0; i < 1000; i++ {
		confidence := rand.Float64() * 2 - 0.5 // Range: -0.5 to 1.5
		level := calculateConfidenceLevel(confidence)

		// Property: confidence level should match confidence value ranges
		switch level {
		case "VERY_HIGH":
			if confidence < 0.9 {
				t.Errorf("Property violated: VERY_HIGH level but confidence %f < 0.9", confidence)
			}
		case "HIGH":
			if confidence < 0.8 || confidence >= 0.9 {
				t.Errorf("Property violated: HIGH level but confidence %f not in [0.8, 0.9)", confidence)
			}
		case "MEDIUM":
			if confidence < 0.6 || confidence >= 0.8 {
				t.Errorf("Property violated: MEDIUM level but confidence %f not in [0.6, 0.8)", confidence)
			}
		case "LOW":
			if confidence < 0.4 || confidence >= 0.6 {
				t.Errorf("Property violated: LOW level but confidence %f not in [0.4, 0.6)", confidence)
			}
		case "VERY_LOW":
			if confidence >= 0.4 {
				t.Errorf("Property violated: VERY_LOW level but confidence %f >= 0.4", confidence)
			}
		}
	}
}

// Test property: bulk operations should preserve individual properties
func TestPropertyBasedBulkOperations(t *testing.T) {
	ctx, cleanup := common.TestContext("property-based-bulk-test")
	defer cleanup()

	for i := 0; i < 50; i++ {
		// Generate random number of relationship IDs
		numIds := rand.Intn(100) + 1
		ids := make([]interface{}, numIds)
		for j := 0; j < numIds; j++ {
			ids[j] = fmt.Sprintf("rel-%d-%d", i, j)
		}

		// Test approval
		args := map[string]interface{}{
			"ids": ids,
		}

		relationships, err := handleApproveRelationships(ctx, args)

		if err != nil {
			t.Errorf("Property violated: bulk approval failed: %v", err)
		}

		// Property: bulk approval should return same number of relationships
		if len(relationships) != len(ids) {
			t.Errorf("Property violated: expected %d relationships, got %d", len(ids), len(relationships))
		}

		// Property: all approved relationships should have correct properties
		for j, rel := range relationships {
			if rel.ID != ids[j].(string) {
				t.Errorf("Property violated: expected ID %s, got %s", ids[j].(string), rel.ID)
			}
			if rel.Confidence != 0.90 {
				t.Errorf("Property violated: approved confidence should be 0.90, got %f", rel.Confidence)
			}
			if rel.ConfidenceLevel != "VERY_HIGH" {
				t.Errorf("Property violated: approved confidence level should be VERY_HIGH, got %s", rel.ConfidenceLevel)
			}
		}

		// Test rejection
		rejectedRels, err := handleRejectRelationships(ctx, args)
		if err != nil {
			t.Errorf("Property violated: bulk rejection failed: %v", err)
		}

		// Property: rejection should always return empty array
		if len(rejectedRels) != 0 {
			t.Errorf("Property violated: rejection should return empty array, got %d relationships", len(rejectedRels))
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
				FieldName: "createRelationship",
			},
			Arguments: map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   "user",
					"toResourceId": "resource",
					"type":         "OWNS",
				},
			},
		}

		_, err := HandleRequest(nil, event)
		if err != nil {
			t.Logf("Expected behavior with nil context: %v", err)
		}
	})

	t.Run("createRelationship with missing input", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function should handle missing input gracefully, but panicked: %v", r)
			}
		}()

		_, err := handleCreateRelationship(ctx, map[string]interface{}{})
		if err != nil {
			t.Logf("Expected behavior with missing input: %v", err)
		}
	})

	t.Run("confidence update with wrong types", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function should handle wrong types gracefully, but panicked: %v", r)
			}
		}()

		args := map[string]interface{}{
			"id":         123, // Wrong type
			"confidence": "not-a-number", // Wrong type
		}

		_, err := handleUpdateRelationshipConfidence(ctx, args)
		if err != nil {
			t.Logf("Expected behavior with wrong types: %v", err)
		}
	})

	t.Run("bulk operations with non-array IDs", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Function should handle non-array IDs gracefully, but panicked: %v", r)
			}
		}()

		args := map[string]interface{}{
			"ids": "not-an-array",
		}

		_, err := handleApproveRelationships(ctx, args)
		if err != nil {
			t.Logf("Expected behavior with non-array IDs: %v", err)
		}
	})
}

// Mutation testing scenarios
func TestMutationTargets(t *testing.T) {
	ctx, cleanup := common.TestContext("mutation-targets-test")
	defer cleanup()

	// Test critical mutations that could break functionality

	t.Run("confidence level calculation boundaries", func(t *testing.T) {
		// Test exact boundary values that mutations often target
		boundaryTests := []struct {
			confidence float64
			expected   string
		}{
			{0.9, "VERY_HIGH"}, // Exact boundary
			{0.8, "HIGH"},      // Exact boundary
			{0.6, "MEDIUM"},    // Exact boundary
			{0.4, "LOW"},       // Exact boundary
		}

		for _, test := range boundaryTests {
			result := calculateConfidenceLevel(test.confidence)
			if result != test.expected {
				t.Errorf("Boundary mutation target failed: confidence %f should be %s, got %s", 
					test.confidence, test.expected, result)
			}
		}
	})

	t.Run("relationship ID generation uniqueness", func(t *testing.T) {
		// Test that relationship IDs are actually unique
		ids := make(map[string]bool)
		
		for i := 0; i < 100; i++ {
			args := map[string]interface{}{
				"input": map[string]interface{}{
					"fromUserId":   fmt.Sprintf("user-%d", i),
					"toResourceId": fmt.Sprintf("resource-%d", i),
					"type":         "OWNS",
				},
			}

			result, err := handleCreateRelationship(ctx, args)
			if err != nil {
				t.Errorf("Error creating relationship: %v", err)
				continue
			}

			if ids[result.ID] {
				t.Errorf("Duplicate relationship ID generated: %s", result.ID)
			}
			ids[result.ID] = true
		}
	})

	t.Run("empty vs nil confidence handling", func(t *testing.T) {
		// Test mutation target: nil confidence vs default
		args1 := map[string]interface{}{
			"input": map[string]interface{}{
				"fromUserId":   "user",
				"toResourceId": "resource",
				"type":         "OWNS",
				"confidence":   nil,
			},
		}

		args2 := map[string]interface{}{
			"input": map[string]interface{}{
				"fromUserId":   "user",
				"toResourceId": "resource",
				"type":         "OWNS",
				// confidence omitted
			},
		}

		result1, _ := handleCreateRelationship(ctx, args1)
		result2, _ := handleCreateRelationship(ctx, args2)

		// Both should use default confidence of 0.5
		if result1 != nil && result2 != nil {
			if result1.Confidence != result2.Confidence {
				t.Errorf("Nil vs omitted confidence should be handled identically")
			}
		}
	})
}

// Benchmark tests for performance
func BenchmarkHandleRequest(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-test")
	defer cleanup()

	event := AppSyncEvent{
		Info: RequestInfo{
			FieldName: "createRelationship",
		},
		Arguments: map[string]interface{}{
			"input": map[string]interface{}{
				"fromUserId":   "benchmark-user",
				"toResourceId": "benchmark-resource",
				"type":         "OWNS",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HandleRequest(ctx, event)
	}
}

func BenchmarkCalculateConfidenceLevel(b *testing.B) {
	confidences := make([]float64, 1000)
	for i := range confidences {
		confidences[i] = rand.Float64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateConfidenceLevel(confidences[i%len(confidences)])
	}
}

func BenchmarkBulkApproval(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-bulk-test")
	defer cleanup()

	// Generate large list of IDs
	ids := make([]interface{}, 100)
	for i := range ids {
		ids[i] = fmt.Sprintf("rel-%d", i)
	}

	args := map[string]interface{}{
		"ids": ids,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handleApproveRelationships(ctx, args)
	}
}

// Helper functions for testing
func generateIDList(count int) []interface{} {
	ids := make([]interface{}, count)
	for i := 0; i < count; i++ {
		ids[i] = fmt.Sprintf("rel-%d", i)
	}
	return ids
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}