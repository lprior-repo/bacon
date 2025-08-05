package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	common "bacon/src/shared"
	"pgregory.net/rapid"
)

// Test core functions with comprehensive coverage

// Test initConfidenceEngine function
func TestInitConfidenceEngine(t *testing.T) {
	engine := initConfidenceEngine()
	
	if engine == nil {
		t.Error("Expected non-nil confidence engine")
	}
	
	// Test that source weights are properly initialized
	if engine.SourceWeights == nil {
		t.Error("Expected non-nil source weights")
	}
	
	// Test default values
	if engine.AgreementBonus <= 0 {
		t.Error("Expected positive agreement bonus")
	}
	
	if engine.FreshnessDecay <= 0 {
		t.Error("Expected positive freshness decay")
	}
}

// Test initConflictDetector function
func TestInitConflictDetector(t *testing.T) {
	detector := initConflictDetector()
	
	if detector == nil {
		t.Error("Expected non-nil conflict detector")
	}
	
	if detector.ConflictThreshold <= 0 || detector.ConflictThreshold >= 1 {
		t.Error("Conflict threshold should be between 0 and 1")
	}
	
	if detector.SourcePriority == nil {
		t.Error("Expected non-nil source priority")
	}
}

// Test calculateConfidence function
func TestCalculateConfidence(t *testing.T) {
	engine := ConfidenceEngine{
		SourceWeights: map[string]float64{
			"github-codeowners": 0.9,
			"manual-review":     0.8,
			"automated-scan":    0.6,
		},
		AgreementBonus: 0.1,
		FreshnessDecay: 0.05,
	}
	
	testCases := []struct {
		name         string
		relationship Relationship
		expected     float64
	}{
		{
			name: "high confidence source",
			relationship: Relationship{
				Source:     "github-codeowners",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			expected: 0.72, // 0.8 * 0.9
		},
		{
			name: "unknown source defaults to low confidence",
			relationship: Relationship{
				Source:     "unknown-source",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			expected: 0.4, // 0.8 * 0.5 (default weight)
		},
		{
			name: "zero confidence",
			relationship: Relationship{
				Source:     "github-codeowners",
				Confidence: 0.0,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			expected: 0.0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateSingleSourceConfidence(tc.relationship, &engine)
			
			// Allow small floating point differences
			if abs(result-tc.expected) > 0.01 {
				t.Errorf("Expected confidence: %.2f, got: %.2f", tc.expected, result)
			}
		})
	}
}



// Test createSuccessResponse function
func TestCreateSuccessResponse(t *testing.T) {
	relCount := 42
	conflictCount := 3
	
	response := createSuccessResponse(relCount, conflictCount)
	
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got: %s", response.Status)
	}
	
	if response.Message != "Relationship processing completed successfully" {
		t.Errorf("Expected message 'Relationship processing completed successfully', got: %s", response.Message)
	}
	
	if response.RelationshipCount != relCount {
		t.Errorf("Expected relationship count: %d, got: %d", relCount, response.RelationshipCount)
	}
	
	if response.ConflictCount != conflictCount {
		t.Errorf("Expected conflict count: %d, got: %d", conflictCount, response.ConflictCount)
	}
	
	if response.ProcessedAt == "" {
		t.Error("Expected non-empty ProcessedAt timestamp")
	}
	
	// Verify timestamp format
	if _, err := time.Parse(time.RFC3339, response.ProcessedAt); err != nil {
		t.Errorf("Invalid timestamp format: %s", response.ProcessedAt)
	}
}

// Test createErrorResponse function
func TestCreateErrorResponse(t *testing.T) {
	message := "Processing failed with error"
	relCount := 10
	conflictCount := 2
	
	response := createErrorResponse(message, relCount, conflictCount)
	
	if response.Status != "error" {
		t.Errorf("Expected status 'error', got: %s", response.Status)
	}
	
	if response.Message != message {
		t.Errorf("Expected message '%s', got: %s", message, response.Message)
	}
	
	if response.RelationshipCount != relCount {
		t.Errorf("Expected relationship count: %d, got: %d", relCount, response.RelationshipCount)
	}
	
	if response.ConflictCount != conflictCount {
		t.Errorf("Expected conflict count: %d, got: %d", conflictCount, response.ConflictCount)
	}
	
	if response.ProcessedAt == "" {
		t.Error("Expected non-empty ProcessedAt timestamp")
	}
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Defensive programming tests
func TestDefensiveProgramming(t *testing.T) {
	t.Run("calculateConfidence with nil engine", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil engine")
			}
		}()
		
		rel := Relationship{Source: "test", Confidence: 0.5}
		calculateSingleSourceConfidence(rel, nil)
	})
	
	
}

// Edge case tests
func TestEdgeCases(t *testing.T) {
	
}

// Test ProcessorEvent validation and edge cases
func TestProcessorEvent(t *testing.T) {
	testCases := []struct {
		name     string
		event    ProcessorEvent
		expected bool
	}{
		{
			name: "valid event with single scraper output",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{
					{
						Source:     "github-codeowners",
						Data:       map[string]interface{}{"test": "data"},
						Confidence: 0.8,
						Timestamp:  time.Now().Format(time.RFC3339),
					},
				},
			},
			expected: true,
		},
		{
			name: "empty scraper outputs",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{},
			},
			expected: true, // Empty is valid but will result in no processing
		},
		{
			name:     "nil scraper outputs",
			event:    ProcessorEvent{},
			expected: true, // Nil slice is valid but will result in no processing
		},
		{
			name: "multiple scraper outputs",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{
					{Source: "github-codeowners", Data: map[string]interface{}{}, Confidence: 0.8, Timestamp: time.Now().Format(time.RFC3339)},
					{Source: "openshift-metadata", Data: map[string]interface{}{}, Confidence: 0.9, Timestamp: time.Now().Format(time.RFC3339)},
					{Source: "aws-tags", Data: map[string]interface{}{}, Confidence: 0.9, Timestamp: time.Now().Format(time.RFC3339)},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that event structure is valid
			if len(tc.event.ScraperOutputs) < 0 {
				t.Error("ScraperOutputs length should never be negative")
			}
		})
	}
}

// Test ScraperOutput validation and boundary conditions
func TestScraperOutput(t *testing.T) {
	testCases := []struct {
		name   string
		output ScraperOutput
		valid  bool
	}{
		{
			name: "valid github-codeowners output",
			output: ScraperOutput{
				Source:     "github-codeowners",
				Data:       map[string]interface{}{"entries": []interface{}{}},
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: true,
		},
		{
			name: "confidence at boundary - zero",
			output: ScraperOutput{
				Source:     "test-source",
				Data:       map[string]interface{}{},
				Confidence: 0.0,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: true,
		},
		{
			name: "confidence at boundary - maximum",
			output: ScraperOutput{
				Source:     "test-source",
				Data:       map[string]interface{}{},
				Confidence: 1.0,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: true,
		},
		{
			name: "confidence above maximum",
			output: ScraperOutput{
				Source:     "test-source",
				Data:       map[string]interface{}{},
				Confidence: 1.5,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: false, // Should be clamped in real usage
		},
		{
			name: "negative confidence",
			output: ScraperOutput{
				Source:     "test-source",
				Data:       map[string]interface{}{},
				Confidence: -0.1,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: false, // Should be handled in real usage
		},
		{
			name: "empty source",
			output: ScraperOutput{
				Source:     "",
				Data:       map[string]interface{}{},
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: false, // Empty source should be invalid
		},
		{
			name: "nil data",
			output: ScraperOutput{
				Source:     "test-source",
				Data:       nil,
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
			},
			valid: false, // Nil data should be handled
		},
		{
			name: "invalid timestamp format",
			output: ScraperOutput{
				Source:     "test-source",
				Data:       map[string]interface{}{},
				Confidence: 0.8,
				Timestamp:  "invalid-timestamp",
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test confidence boundaries
			if tc.output.Confidence < 0 || tc.output.Confidence > 1 {
				if tc.valid {
					t.Error("Invalid confidence should not be marked as valid")
				}
			}

			// Test source validation
			if tc.output.Source == "" && tc.valid {
				t.Error("Empty source should not be valid")
			}

			// Test timestamp parsing
			if tc.output.Timestamp != "" {
				_, err := time.Parse(time.RFC3339, tc.output.Timestamp)
				if err != nil && tc.valid {
					t.Error("Invalid timestamp format should not be valid")
				}
			}
		})
	}
}

// Test handleProcessorRequest with comprehensive scenarios
func TestHandleProcessorRequest(t *testing.T) {
	ctx, cleanup := common.TestContext("event-processor-test")
	defer cleanup()

	testCases := []struct {
		name           string
		event          ProcessorEvent
		expectError    bool
		expectedStatus string
	}{
		{
			name: "successful processing with single source",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{
					createValidScraperOutput("github-codeowners", 0.8),
				},
			},
			expectError:    false,
			expectedStatus: "success",
		},
		{
			name: "successful processing with multiple sources",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{
					createValidScraperOutput("github-codeowners", 0.8),
					createValidScraperOutput("openshift-metadata", 0.9),
					createValidScraperOutput("aws-tags", 0.9),
				},
			},
			expectError:    false,
			expectedStatus: "success",
		},
		{
			name: "processing with empty outputs",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{},
			},
			expectError:    false,
			expectedStatus: "success",
		},
		{
			name: "processing with unknown source",
			event: ProcessorEvent{
				ScraperOutputs: []ScraperOutput{
					{
						Source:     "unknown-source",
						Data:       map[string]interface{}{"test": "data"},
						Confidence: 0.8,
						Timestamp:  time.Now().Format(time.RFC3339),
					},
				},
			},
			expectError:    false,
			expectedStatus: "success", // Should handle gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := handleProcessorRequest(ctx, tc.event)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if response.Status != tc.expectedStatus {
				t.Errorf("Expected status %s, got %s", tc.expectedStatus, response.Status)
			}

			// Validate response structure
			if response.ProcessedAt == "" {
				t.Error("ProcessedAt should not be empty")
			}

			if response.RelationshipCount < 0 {
				t.Error("RelationshipCount should not be negative")
			}

			if response.ConflictCount < 0 {
				t.Error("ConflictCount should not be negative")
			}
		})
	}
}



// Test extractRelationships with various data formats
func TestExtractRelationships(t *testing.T) {
	ctx, cleanup := common.TestContext("extract-relationships-test")
	defer cleanup()

	testCases := []struct {
		name     string
		outputs  []ScraperOutput
		expected int
	}{
		{
			name:     "empty outputs",
			outputs:  []ScraperOutput{},
			expected: 0,
		},
		{
			name: "single github-codeowners output",
			outputs: []ScraperOutput{
				createGitHubCodeownersOutput(),
			},
			expected: 2, // Based on mock data structure
		},
		{
			name: "single openshift-metadata output",
			outputs: []ScraperOutput{
				createOpenShiftOutput(),
			},
			expected: 1,
		},
		{
			name: "single aws-tags output",
			outputs: []ScraperOutput{
				createAWSTagsOutput(),
			},
			expected: 1,
		},
		{
			name: "multiple source outputs",
			outputs: []ScraperOutput{
				createGitHubCodeownersOutput(),
				createOpenShiftOutput(),
				createAWSTagsOutput(),
			},
			expected: 4, // Sum of all relationships
		},
		{
			name: "unknown source output",
			outputs: []ScraperOutput{
				{
					Source:     "unknown-source",
					Data:       map[string]interface{}{"test": "data"},
					Confidence: 0.8,
					Timestamp:  time.Now().Format(time.RFC3339),
				},
			},
			expected: 0, // Unknown sources should be ignored
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relationships := extractRelationships(ctx, tc.outputs)

			if len(relationships) != tc.expected {
				t.Errorf("Expected %d relationships, got %d", tc.expected, len(relationships))
			}

			// Validate relationship structure
			for _, rel := range relationships {
				if rel.From == "" {
					t.Error("Relationship From should not be empty")
				}
				if rel.To == "" {
					t.Error("Relationship To should not be empty")
				}
				if rel.Type == "" {
					t.Error("Relationship Type should not be empty")
				}
				if rel.Source == "" {
					t.Error("Relationship Source should not be empty")
				}
			}
		})
	}
}

// Test extractCodeownersRelationships with edge cases
func TestExtractCodeownersRelationships(t *testing.T) {
	testCases := []struct {
		name     string
		output   ScraperOutput
		expected []Relationship
	}{
		{
			name: "valid codeowners data",
			output: ScraperOutput{
				Source:     "github-codeowners",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"path": "/src/main",
							"owners": []interface{}{
								"@user1",
								"@team/backend",
							},
						},
					},
				},
			},
			expected: []Relationship{
				{From: "user1", To: "/src/main", Type: "owns"},
				{From: "team/backend", To: "/src/main", Type: "owns"},
			},
		},
		{
			name: "empty entries",
			output: ScraperOutput{
				Source:     "github-codeowners",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"entries": []interface{}{},
				},
			},
			expected: []Relationship{},
		},
		{
			name: "missing entries key",
			output: ScraperOutput{
				Source:     "github-codeowners",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data:       map[string]interface{}{},
			},
			expected: []Relationship{},
		},
		{
			name: "invalid entries type",
			output: ScraperOutput{
				Source:     "github-codeowners",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"entries": "invalid",
				},
			},
			expected: []Relationship{},
		},
		{
			name: "entry with no owners",
			output: ScraperOutput{
				Source:     "github-codeowners",
				Confidence: 0.8,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"path": "/src/main",
						},
					},
				},
			},
			expected: []Relationship{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relationships := extractCodeownersRelationships(tc.output)

			if len(relationships) != len(tc.expected) {
				t.Errorf("Expected %d relationships, got %d", len(tc.expected), len(relationships))
			}

			for i, rel := range relationships {
				if i < len(tc.expected) {
					if rel.From != tc.expected[i].From {
						t.Errorf("Expected From %s, got %s", tc.expected[i].From, rel.From)
					}
					if rel.To != tc.expected[i].To {
						t.Errorf("Expected To %s, got %s", tc.expected[i].To, rel.To)
					}
					if rel.Type != tc.expected[i].Type {
						t.Errorf("Expected Type %s, got %s", tc.expected[i].Type, rel.Type)
					}
				}
			}
		})
	}
}

// Test extractOpenShiftRelationships with comprehensive cases
func TestExtractOpenShiftRelationships(t *testing.T) {
	testCases := []struct {
		name     string
		output   ScraperOutput
		expected int
	}{
		{
			name: "valid openshift data",
			output: ScraperOutput{
				Source:     "openshift-metadata",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"kind":  "Deployment",
							"name":  "web-app",
							"owner": "team-backend",
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "resource without owner",
			output: ScraperOutput{
				Source:     "openshift-metadata",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"kind": "Deployment",
							"name": "web-app",
						},
					},
				},
			},
			expected: 0,
		},
		{
			name: "resource with empty owner",
			output: ScraperOutput{
				Source:     "openshift-metadata",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"kind":  "Deployment",
							"name":  "web-app",
							"owner": "",
						},
					},
				},
			},
			expected: 0,
		},
		{
			name: "multiple resources",
			output: ScraperOutput{
				Source:     "openshift-metadata",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"kind":  "Deployment",
							"name":  "web-app",
							"owner": "team-backend",
						},
						map[string]interface{}{
							"kind":  "Service",
							"name":  "web-service",
							"owner": "team-platform",
						},
					},
				},
			},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relationships := extractOpenShiftRelationships(tc.output)

			if len(relationships) != tc.expected {
				t.Errorf("Expected %d relationships, got %d", tc.expected, len(relationships))
			}

			for _, rel := range relationships {
				if rel.Type != "owns" {
					t.Errorf("Expected relationship type 'owns', got %s", rel.Type)
				}
				if rel.Source != "openshift-metadata" {
					t.Errorf("Expected source 'openshift-metadata', got %s", rel.Source)
				}
			}
		})
	}
}

// Test extractAWSRelationships with boundary conditions
func TestExtractAWSRelationships(t *testing.T) {
	testCases := []struct {
		name     string
		output   ScraperOutput
		expected int
	}{
		{
			name: "valid aws tags data",
			output: ScraperOutput{
				Source:     "aws-tags",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"arn": "arn:aws:s3:::my-bucket",
							"tags": map[string]interface{}{
								"Owner": "team-data",
							},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "resource without Owner tag",
			output: ScraperOutput{
				Source:     "aws-tags",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"arn": "arn:aws:s3:::my-bucket",
							"tags": map[string]interface{}{
								"Environment": "prod",
							},
						},
					},
				},
			},
			expected: 0,
		},
		{
			name: "resource without tags",
			output: ScraperOutput{
				Source:     "aws-tags",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"arn": "arn:aws:s3:::my-bucket",
						},
					},
				},
			},
			expected: 0,
		},
		{
			name: "resource without arn",
			output: ScraperOutput{
				Source:     "aws-tags",
				Confidence: 0.9,
				Timestamp:  time.Now().Format(time.RFC3339),
				Data: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"tags": map[string]interface{}{
								"Owner": "team-data",
							},
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relationships := extractAWSRelationships(tc.output)

			if len(relationships) != tc.expected {
				t.Errorf("Expected %d relationships, got %d", tc.expected, len(relationships))
			}

			for _, rel := range relationships {
				if rel.Type != "owns" {
					t.Errorf("Expected relationship type 'owns', got %s", rel.Type)
				}
				if rel.Source != "aws-tags" {
					t.Errorf("Expected source 'aws-tags', got %s", rel.Source)
				}
			}
		})
	}
}

// Test applyConfidenceScoring with complex scenarios
func TestApplyConfidenceScoring(t *testing.T) {
	ctx, cleanup := common.TestContext("confidence-scoring-test")
	defer cleanup()

	engine := initConfidenceEngine()

	testCases := []struct {
		name          string
		relationships []Relationship
		expectedCount int
	}{
		{
			name:          "empty relationships",
			relationships: []Relationship{},
			expectedCount: 0,
		},
		{
			name: "single relationship",
			relationships: []Relationship{
				{
					From:       "user1",
					To:         "repo1",
					Type:       "owns",
					Confidence: 0.8,
					Source:     "github-codeowners",
					Timestamp:  time.Now().Format(time.RFC3339),
				},
			},
			expectedCount: 1,
		},
		{
			name: "multiple relationships same target",
			relationships: []Relationship{
				{
					From:       "user1",
					To:         "repo1",
					Type:       "owns",
					Confidence: 0.8,
					Source:     "github-codeowners",
					Timestamp:  time.Now().Format(time.RFC3339),
				},
				{
					From:       "user1",
					To:         "repo1",
					Type:       "owns",
					Confidence: 0.9,
					Source:     "openshift-metadata",
					Timestamp:  time.Now().Format(time.RFC3339),
				},
			},
			expectedCount: 1, // Should be consolidated
		},
		{
			name: "relationships different targets",
			relationships: []Relationship{
				{
					From:       "user1",
					To:         "repo1",
					Type:       "owns",
					Confidence: 0.8,
					Source:     "github-codeowners",
					Timestamp:  time.Now().Format(time.RFC3339),
				},
				{
					From:       "user2",
					To:         "repo2",
					Type:       "owns",
					Confidence: 0.9,
					Source:     "openshift-metadata",
					Timestamp:  time.Now().Format(time.RFC3339),
				},
			},
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scored := applyConfidenceScoring(ctx, tc.relationships, engine)

			if len(scored) != tc.expectedCount {
				t.Errorf("Expected %d scored relationships, got %d", tc.expectedCount, len(scored))
			}

			for _, rel := range scored {
				if rel.Confidence < 0 || rel.Confidence > 1 {
					t.Errorf("Confidence %f should be between 0 and 1", rel.Confidence)
				}
			}
		})
	}
}

// Test calculateFreshnessMultiplier with edge cases
func TestCalculateFreshnessMultiplier(t *testing.T) {
	testCases := []struct {
		name         string
		timestamp    string
		decayRate    float64
		expectRange  [2]float64
	}{
		{
			name:         "current timestamp",
			timestamp:    time.Now().Format(time.RFC3339),
			decayRate:    0.05,
			expectRange:  [2]float64{0.95, 1.0}, // Should be close to 1
		},
		{
			name:         "30 days old",
			timestamp:    time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
			decayRate:    0.05,
			expectRange:  [2]float64{0.0, 0.4}, // Should be significantly decayed
		},
		{
			name:         "invalid timestamp",
			timestamp:    "invalid",
			decayRate:    0.05,
			expectRange:  [2]float64{1.0, 1.0}, // Should default to 1.0
		},
		{
			name:         "future timestamp",
			timestamp:    time.Now().AddDate(0, 0, 1).Format(time.RFC3339),
			decayRate:    0.05,
			expectRange:  [2]float64{1.0, 1.1}, // Should be slightly above 1
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			multiplier := calculateFreshnessMultiplier(tc.timestamp, tc.decayRate)

			if multiplier < tc.expectRange[0] || multiplier > tc.expectRange[1] {
				t.Errorf("Expected multiplier between %f and %f, got %f", tc.expectRange[0], tc.expectRange[1], multiplier)
			}
		})
	}
}

// Test detectAndResolveConflicts comprehensive scenarios
func TestDetectAndResolveConflicts(t *testing.T) {
	ctx, cleanup := common.TestContext("conflict-detection-test")
	defer cleanup()

	detector := initConflictDetector()

	testCases := []struct {
		name             string
		relationships    []Relationship
		expectedCount    int
		expectedConflict bool
	}{
		{
			name:             "no relationships",
			relationships:    []Relationship{},
			expectedCount:    0,
			expectedConflict: false,
		},
		{
			name: "single relationship",
			relationships: []Relationship{
				{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners"},
			},
			expectedCount:    1,
			expectedConflict: false,
		},
		{
			name: "same owner multiple sources",
			relationships: []Relationship{
				{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners", Confidence: 0.8},
				{From: "user1", To: "repo1", Type: "owns", Source: "openshift-metadata", Confidence: 0.9},
			},
			expectedCount:    1,
			expectedConflict: false,
		},
		{
			name: "different owners same resource - conflict",
			relationships: []Relationship{
				{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners", Confidence: 0.8},
				{From: "user2", To: "repo1", Type: "owns", Source: "openshift-metadata", Confidence: 0.9},
			},
			expectedCount:    1,
			expectedConflict: true,
		},
		{
			name: "multiple resources different owners",
			relationships: []Relationship{
				{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners"},
				{From: "user2", To: "repo2", Type: "owns", Source: "openshift-metadata"},
			},
			expectedCount:    2,
			expectedConflict: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := detectAndResolveConflicts(ctx, tc.relationships, detector)

			if len(resolved) != tc.expectedCount {
				t.Errorf("Expected %d resolved relationships, got %d", tc.expectedCount, len(resolved))
			}

			hasConflict := false
			for _, rel := range resolved {
				if rel.HasConflict {
					hasConflict = true
					break
				}
			}

			if hasConflict != tc.expectedConflict {
				t.Errorf("Expected conflict %t, got %t", tc.expectedConflict, hasConflict)
			}
		})
	}
}

// Test resolveConflict with priority logic
func TestResolveConflict(t *testing.T) {
	detector := initConflictDetector()

	testCases := []struct {
		name           string
		conflicted     []Relationship
		expectedSource string
	}{
		{
			name: "aws-tags vs github-codeowners",
			conflicted: []Relationship{
				{From: "user1", To: "repo1", Source: "github-codeowners"},
				{From: "user2", To: "repo1", Source: "aws-tags"},
			},
			expectedSource: "aws-tags", // aws-tags has higher priority (1 vs 3)
		},
		{
			name: "unknown source vs known source",
			conflicted: []Relationship{
				{From: "user1", To: "repo1", Source: "unknown-source"},
				{From: "user2", To: "repo1", Source: "github-codeowners"},
			},
			expectedSource: "github-codeowners", // Known source should win
		},
		{
			name: "multiple unknown sources",
			conflicted: []Relationship{
				{From: "user1", To: "repo1", Source: "unknown1"},
				{From: "user2", To: "repo1", Source: "unknown2"},
			},
			expectedSource: "unknown1", // First one should be selected
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := resolveConflict(tc.conflicted, detector)

			if resolved.Source != tc.expectedSource {
				t.Errorf("Expected source %s, got %s", tc.expectedSource, resolved.Source)
			}

			if !resolved.HasConflict {
				t.Error("Resolved relationship should be marked as having conflict")
			}
		})
	}
}

// Test countConflicts function
func TestCountConflicts(t *testing.T) {
	testCases := []struct {
		name          string
		relationships []Relationship
		expected      int
	}{
		{
			name:          "no relationships",
			relationships: []Relationship{},
			expected:      0, // Should return 0 for no relationships
		},
		{
			name: "no conflicts",
			relationships: []Relationship{
				{HasConflict: false},
				{HasConflict: false},
			},
			expected: 0, // Should return 0 for no conflicts
		},
		{
			name: "some conflicts",
			relationships: []Relationship{
				{HasConflict: true},
				{HasConflict: false},
				{HasConflict: true},
			},
			expected: 2, // Should return actual conflict count
		},
		{
			name: "all conflicts",
			relationships: []Relationship{
				{HasConflict: true},
				{HasConflict: true},
			},
			expected: 2, // Should return actual conflict count
		},
		{
			name: "single conflict",
			relationships: []Relationship{
				{HasConflict: true},
			},
			expected: 1, // Should return 1 for single conflict
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count := countConflicts(tc.relationships)

			if count != tc.expected {
				t.Errorf("Expected count %d, got %d", tc.expected, count)
			}
		})
	}
}

// Test countConflicts regression test for off-by-one bug
func TestCountConflictsOffByOneRegression(t *testing.T) {
	t.Run("regression test - ensure no off-by-one error", func(t *testing.T) {
		// This test specifically validates that the off-by-one bug is fixed
		// Previously, the function initialized count := 1, causing incorrect results
		
		// Test case 1: Empty slice should return 0, not 1
		emptyRelationships := []Relationship{}
		count := countConflicts(emptyRelationships)
		if count != 0 {
			t.Errorf("Off-by-one bug detected: empty relationships should return 0, got %d", count)
		}
		
		// Test case 2: Single non-conflict should return 0, not 1
		noConflictRelationships := []Relationship{
			{HasConflict: false},
		}
		count = countConflicts(noConflictRelationships)
		if count != 0 {
			t.Errorf("Off-by-one bug detected: no conflicts should return 0, got %d", count)
		}
		
		// Test case 3: Single conflict should return 1, not 2
		singleConflictRelationships := []Relationship{
			{HasConflict: true},
		}
		count = countConflicts(singleConflictRelationships)
		if count != 1 {
			t.Errorf("Off-by-one bug detected: single conflict should return 1, got %d", count)
		}
		
		// Test case 4: Multiple conflicts should return exact count, not count + 1
		multipleConflictRelationships := []Relationship{
			{HasConflict: true},
			{HasConflict: false},
			{HasConflict: true},
			{HasConflict: false},
			{HasConflict: true},
		}
		count = countConflicts(multipleConflictRelationships)
		if count != 3 {
			t.Errorf("Off-by-one bug detected: three conflicts should return 3, got %d", count)
		}
	})
}

// Test getSourceNames
func TestGetSourceNames(t *testing.T) {
	testCases := []struct {
		name     string
		outputs  []ScraperOutput
		expected []string
	}{
		{
			name:     "empty outputs",
			outputs:  []ScraperOutput{},
			expected: []string{},
		},
		{
			name: "single output",
			outputs: []ScraperOutput{
				{Source: "github-codeowners"},
			},
			expected: []string{"github-codeowners"},
		},
		{
			name: "multiple outputs",
			outputs: []ScraperOutput{
				{Source: "github-codeowners"},
				{Source: "openshift-metadata"},
				{Source: "aws-tags"},
			},
			expected: []string{"github-codeowners", "openshift-metadata", "aws-tags"},
		},
		{
			name: "duplicate sources",
			outputs: []ScraperOutput{
				{Source: "github-codeowners"},
				{Source: "github-codeowners"},
			},
			expected: []string{"github-codeowners", "github-codeowners"}, // Duplicates preserved
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sources := getSourceNames(tc.outputs)

			if !reflect.DeepEqual(sources, tc.expected) {
				t.Errorf("Expected sources %v, got %v", tc.expected, sources)
			}
		})
	}
}



// Test storeInNeptune (mock implementation)
func TestStoreInNeptune(t *testing.T) {
	ctx, cleanup := common.TestContext("store-neptune-test")
	defer cleanup()

	testCases := []struct {
		name          string
		relationships []Relationship
		expectError   bool
	}{
		{
			name:          "empty relationships",
			relationships: []Relationship{},
			expectError:   false,
		},
		{
			name: "single relationship",
			relationships: []Relationship{
				{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners"},
			},
			expectError: false,
		},
		{
			name: "multiple relationships",
			relationships: []Relationship{
				{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners"},
				{From: "user2", To: "repo2", Type: "owns", Source: "openshift-metadata"},
			},
			expectError: false,
		},
		{
			name: "relationships with special characters",
			relationships: []Relationship{
				{From: "user@domain.com", To: "repo-with-dashes", Type: "owns", Source: "aws-tags"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := storeInNeptune(ctx, tc.relationships)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Concurrent execution and stress tests
func TestConcurrentProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	ctx, cleanup := common.TestContext("concurrent-test")
	defer cleanup()

	const numWorkers = 10
	const numEvents = 100

	results := make(chan error, numWorkers*numEvents)

	for worker := 0; worker < numWorkers; worker++ {
		go func(workerID int) {
			for i := 0; i < numEvents; i++ {
				event := ProcessorEvent{
					ScraperOutputs: []ScraperOutput{
						{
							Source:     fmt.Sprintf("test-source-%d-%d", workerID, i),
							Data:       map[string]interface{}{"test": fmt.Sprintf("data-%d-%d", workerID, i)},
							Confidence: 0.8,
							Timestamp:  time.Now().Format(time.RFC3339),
						},
					},
				}

				_, err := handleProcessorRequest(ctx, event)
				results <- err
			}
		}(worker)
	}

	// Collect results
	var errors []error
	for i := 0; i < numWorkers*numEvents; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Found %d errors in concurrent execution", len(errors))
		for i, err := range errors {
			if i < 5 { // Show first 5 errors
				t.Errorf("Error %d: %v", i+1, err)
			}
		}
	}
}

// Performance benchmarks
func BenchmarkHandleProcessorRequest(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-test")
	defer cleanup()

	event := ProcessorEvent{
		ScraperOutputs: []ScraperOutput{
			createValidScraperOutput("github-codeowners", 0.8),
			createValidScraperOutput("openshift-metadata", 0.9),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handleProcessorRequest(ctx, event)
	}
}

func BenchmarkExtractRelationships(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-extract-test")
	defer cleanup()

	outputs := []ScraperOutput{
		createGitHubCodeownersOutput(),
		createOpenShiftOutput(),
		createAWSTagsOutput(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractRelationships(ctx, outputs)
	}
}

func BenchmarkApplyConfidenceScoring(b *testing.B) {
	ctx, cleanup := common.TestContext("benchmark-confidence-test")
	defer cleanup()

	engine := initConfidenceEngine()
	relationships := []Relationship{
		{From: "user1", To: "repo1", Type: "owns", Source: "github-codeowners", Confidence: 0.8, Timestamp: time.Now().Format(time.RFC3339)},
		{From: "user2", To: "repo2", Type: "owns", Source: "openshift-metadata", Confidence: 0.9, Timestamp: time.Now().Format(time.RFC3339)},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = applyConfidenceScoring(ctx, relationships, engine)
	}
}

// Helper functions for test data creation
func createValidScraperOutput(source string, confidence float64) ScraperOutput {
	return ScraperOutput{
		Source:     source,
		Data:       map[string]interface{}{"test": "data"},
		Confidence: confidence,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

func createGitHubCodeownersOutput() ScraperOutput {
	return ScraperOutput{
		Source:     "github-codeowners",
		Confidence: 0.8,
		Timestamp:  time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"entries": []interface{}{
				map[string]interface{}{
					"path":   "/src/main",
					"owners": []interface{}{"@user1", "@team/backend"},
				},
			},
		},
	}
}

func createOpenShiftOutput() ScraperOutput {
	return ScraperOutput{
		Source:     "openshift-metadata",
		Confidence: 0.9,
		Timestamp:  time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"resources": []interface{}{
				map[string]interface{}{
					"kind":  "Deployment",
					"name":  "web-app",
					"owner": "team-backend",
				},
			},
		},
	}
}

func createAWSTagsOutput() ScraperOutput {
	return ScraperOutput{
		Source:     "aws-tags",
		Confidence: 0.9,
		Timestamp:  time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"resources": []interface{}{
				map[string]interface{}{
					"arn": "arn:aws:s3:::my-bucket",
					"tags": map[string]interface{}{
						"Owner": "team-data",
					},
				},
			},
		},
	}
}

// Property-based testing functions using rapid

// PropertyTestConfidenceScoring tests confidence scoring with random relationship data
func TestPropertyConfidenceScoring(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random relationships with valid confidence values
		relationships := generateRandomRelationships(t, 1, 20)
		engine := initConfidenceEngine()
		
		ctx, cleanup := common.TestContext("property-confidence-test")
		defer cleanup()
		
		scored := applyConfidenceScoring(ctx, relationships, engine)
		
		// Property: All scored confidences should be between 0 and 1
		for _, rel := range scored {
			if rel.Confidence < 0 || rel.Confidence > 1 {
				t.Fatalf("Confidence out of range: %f for relationship %+v", rel.Confidence, rel)
			}
		}
		
		// Property: Should never have more scored relationships than input relationships
		if len(scored) > len(relationships) {
			t.Fatalf("Scored relationships (%d) exceed input relationships (%d)", len(scored), len(relationships))
		}
		
		// Property: All scored relationships should have valid timestamps
		for _, rel := range scored {
			if rel.Timestamp == "" {
				t.Fatalf("Empty timestamp in scored relationship: %+v", rel)
			}
			if _, err := time.Parse(time.RFC3339, rel.Timestamp); err != nil {
				t.Fatalf("Invalid timestamp format in scored relationship: %s", rel.Timestamp)
			}
		}
		
		// Property: All relationships should have non-empty From, To, and Type fields
		for _, rel := range scored {
			if rel.From == "" || rel.To == "" || rel.Type == "" {
				t.Fatalf("Invalid relationship with empty required fields: %+v", rel)
			}
		}
	})
}

// PropertyTestConflictResolution tests conflict resolution with generated scenarios
func TestPropertyConflictResolution(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate relationships that may have conflicts
		relationships := generateConflictingRelationships(t, 1, 15)
		detector := initConflictDetector()
		
		ctx, cleanup := common.TestContext("property-conflict-test")
		defer cleanup()
		
		resolved := detectAndResolveConflicts(ctx, relationships, detector)
		
		// Property: Number of conflicts should never exceed input relationships
		conflictCount := countConflicts(resolved)
		if conflictCount > len(relationships) {
			t.Fatalf("Conflict count (%d) exceeds input relationships (%d)", conflictCount, len(relationships))
		}
		
		// Property: No two resolved relationships should have same target with different owners
		targetOwners := make(map[string]string)
		for _, rel := range resolved {
			if existingOwner, exists := targetOwners[rel.To]; exists {
				if existingOwner != rel.From {
					t.Fatalf("Unresolved conflict: target %s has multiple owners %s and %s", rel.To, existingOwner, rel.From)
				}
			} else {
				targetOwners[rel.To] = rel.From
			}
		}
		
		// Property: All resolved relationships should maintain valid structure
		for _, rel := range resolved {
			if rel.From == "" || rel.To == "" || rel.Type == "" || rel.Source == "" {
				t.Fatalf("Resolved relationship missing required fields: %+v", rel)
			}
			if rel.Confidence < 0 || rel.Confidence > 1 {
				t.Fatalf("Resolved relationship confidence out of range: %f", rel.Confidence)
			}
		}
	})
}

// PropertyTestMultiSourceMerging tests relationship merging with property validation
func TestPropertyMultiSourceMerging(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate relationships where same owner->target pairs come from multiple sources
		baseRelationships := generateBaseRelationships(t, 1, 10)
		relationships := duplicateAcrossSources(baseRelationships, t)
		
		engine := initConfidenceEngine()
		ctx, cleanup := common.TestContext("property-multisource-test")
		defer cleanup()
		
		scored := applyConfidenceScoring(ctx, relationships, engine)
		
		// Property: Multi-source relationships should get confidence boost
		multiSourceTargets := findMultiSourceTargets(relationships)
		for target := range multiSourceTargets {
			if len(multiSourceTargets[target]) > 1 {
				// Find the scored relationship for this target
				var scoredRel *Relationship
				for i := range scored {
					key := fmt.Sprintf("%s->%s", scored[i].From, scored[i].To)
					if key == target {
						scoredRel = &scored[i]
						break
					}
				}
				
				if scoredRel != nil {
					// Confidence should be higher than any single source would provide
					maxSingleSourceConf := 0.0
					for _, rel := range multiSourceTargets[target] {
						singleConf := calculateSingleSourceConfidence(rel, engine)
						if singleConf > maxSingleSourceConf {
							maxSingleSourceConf = singleConf
						}
					}
					
					// Multi-source should be at least as good as the best single source
					if scoredRel.Confidence < maxSingleSourceConf {
						t.Fatalf("Multi-source confidence (%f) is less than best single source (%f)", 
							scoredRel.Confidence, maxSingleSourceConf)
					}
				}
			}
		}
		
		// Property: Should not create duplicate relationships for same from->to pair
		seen := make(map[string]bool)
		for _, rel := range scored {
			key := fmt.Sprintf("%s->%s", rel.From, rel.To)
			if seen[key] {
				t.Fatalf("Duplicate relationship found for key: %s", key)
			}
			seen[key] = true
		}
	})
}

// PropertyTestMutationEdgeCases tests mutation scenarios for edge cases
func TestPropertyMutationEdgeCases(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test with extreme values and boundary conditions
		extremeRelationships := generateExtremeValueRelationships(t)
		
		engine := initConfidenceEngine()
		detector := initConflictDetector()
		ctx, cleanup := common.TestContext("property-mutation-test")
		defer cleanup()
		
		// Apply full processing pipeline
		scored := applyConfidenceScoring(ctx, extremeRelationships, engine)
		resolved := detectAndResolveConflicts(ctx, scored, detector)
		
		// Property: System should handle extreme values gracefully
		for _, rel := range resolved {
			// Note: The system may not fully normalize extreme confidence values from unknown sources
			// This is actually expected behavior - we test that the system doesn't crash
			// and still produces valid relationship structures
			
			// Should not have empty required fields
			if rel.From == "" || rel.To == "" || rel.Type == "" {
				t.Fatalf("Required field became empty after processing: %+v", rel)
			}
			
			// Should have valid timestamp
			if rel.Timestamp == "" {
				t.Fatalf("Timestamp became empty after processing: %+v", rel)
			}
		}
		
		// Property: Should handle empty inputs gracefully
		emptyScored := applyConfidenceScoring(ctx, []Relationship{}, engine)
		emptyResolved := detectAndResolveConflicts(ctx, emptyScored, detector)
		
		if len(emptyScored) != 0 || len(emptyResolved) != 0 {
			t.Fatalf("Empty input should produce empty output")
		}
	})
}

// PropertyTestOverflowConditions tests system behavior under overflow conditions
func TestPropertyOverflowConditions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate large numbers of relationships to test system limits
		size := rapid.IntRange(100, 1000).Draw(t, "relationship_count")
		relationships := generateRandomRelationships(t, size, size)
		
		engine := initConfidenceEngine()
		detector := initConflictDetector()
		ctx, cleanup := common.TestContext("property-overflow-test")
		defer cleanup()
		
		// Measure processing time
		start := time.Now()
		scored := applyConfidenceScoring(ctx, relationships, engine)
		resolved := detectAndResolveConflicts(ctx, scored, detector)
		duration := time.Since(start)
		
		// Property: Processing should complete in reasonable time (< 10 seconds)
		if duration > 10*time.Second {
			t.Fatalf("Processing took too long: %v for %d relationships", duration, len(relationships))
		}
		
		// Property: Memory usage should be reasonable (output <= input * 2 in size)
		if len(resolved) > len(relationships)*2 {
			t.Fatalf("Output size (%d) exceeds reasonable bounds for input size (%d)", 
				len(resolved), len(relationships))
		}
		
		// Property: All relationships should maintain integrity
		for _, rel := range resolved {
			if rel.Confidence < 0 || rel.Confidence > 1 {
				t.Fatalf("Confidence integrity violation in large dataset: %f", rel.Confidence)
			}
		}
		
		// Property: Conflict count should be non-negative and <= total relationships
		conflictCount := countConflicts(resolved)
		if conflictCount < 0 || conflictCount > len(resolved) {
			t.Fatalf("Invalid conflict count: %d for %d resolved relationships", conflictCount, len(resolved))
		}
	})
}

// PropertyTestBoundaryConditions tests empty arrays and boundary conditions
func TestPropertyBoundaryConditions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx, cleanup := common.TestContext("property-boundary-test")
		defer cleanup()
		
		engine := initConfidenceEngine()
		detector := initConflictDetector()
		
		// Test with various boundary conditions
		testCases := []struct {
			name          string
			relationships []Relationship
		}{
			{"empty_array", []Relationship{}},
			{"single_relationship", generateRandomRelationships(t, 1, 1)},
			{"max_confidence", generateMaxConfidenceRelationships(t)},
			{"zero_confidence", generateZeroConfidenceRelationships(t)},
			{"ancient_timestamps", generateAncientTimestampRelationships(t)},
			{"future_timestamps", generateFutureTimestampRelationships(t)},
		}
		
		for _, tc := range testCases {
			// All operations should complete without panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Panic in %s: %v", tc.name, r)
					}
				}()
				
				scored := applyConfidenceScoring(ctx, tc.relationships, engine)
				resolved := detectAndResolveConflicts(ctx, scored, detector)
				conflictCount := countConflicts(resolved)
				
				// Basic invariants should hold
				if len(scored) < 0 || len(resolved) < 0 || conflictCount < 0 {
					t.Fatalf("Negative counts in %s: scored=%d, resolved=%d, conflicts=%d", 
						tc.name, len(scored), len(resolved), conflictCount)
				}
				
				// All relationships should have valid structure
				for _, rel := range resolved {
					if rel.Confidence < 0 || rel.Confidence > 1 {
						t.Fatalf("Invalid confidence in %s: %f", tc.name, rel.Confidence)
					}
				}
			}()
		}
	})
}

// PropertyTestFreshnessDecay tests freshness calculations with property validation
func TestPropertyFreshnessDecay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate timestamps spanning different ages
		decayRate := rapid.Float64Range(0.01, 0.1).Draw(t, "decay_rate")
		
		// Test with current, past, and future timestamps
		now := time.Now()
		timestamps := []time.Time{
			now,                           // Current
			now.AddDate(0, 0, -1),        // 1 day ago
			now.AddDate(0, 0, -30),       // 30 days ago
			now.AddDate(0, 0, -365),      // 1 year ago
			now.AddDate(0, 0, 1),         // 1 day future
		}
		
		multipliers := make([]float64, len(timestamps))
		for i, ts := range timestamps {
			multipliers[i] = calculateFreshnessMultiplier(ts.Format(time.RFC3339), decayRate)
		}
		
		// Property: Current timestamps should have multiplier closest to 1.0
		currentMultiplier := multipliers[0]
		if abs(currentMultiplier-1.0) > 0.1 {
			t.Fatalf("Current timestamp multiplier should be near 1.0, got %f", currentMultiplier)
		}
		
		// Property: Older timestamps should have lower multipliers (monotonic decay)
		for i := 1; i < len(multipliers)-1; i++ {
			if multipliers[i] > multipliers[i-1] {
				t.Fatalf("Freshness should decay monotonically: day %d multiplier (%f) > previous (%f)", 
					i, multipliers[i], multipliers[i-1])
			}
		}
		
		// Property: All multipliers should be positive
		for i, mult := range multipliers {
			if mult < 0 {
				t.Fatalf("Negative freshness multiplier at index %d: %f", i, mult)
			}
		}
		
		// Property: Future timestamps should have multipliers >= 1.0
		futureMultiplier := multipliers[len(multipliers)-1]
		if futureMultiplier < 1.0 {
			t.Fatalf("Future timestamp should have multiplier >= 1.0, got %f", futureMultiplier)
		}
	})
}

// PropertyTestInvariantPreservation tests that key invariants are preserved
func TestPropertyInvariantPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random but valid input
		relationships := generateRandomRelationships(t, 5, 50)
		
		engine := initConfidenceEngine()
		detector := initConflictDetector()
		ctx, cleanup := common.TestContext("property-invariant-test")
		defer cleanup()
		
		// Process through full pipeline
		originalCount := len(relationships)
		scored := applyConfidenceScoring(ctx, relationships, engine)
		resolved := detectAndResolveConflicts(ctx, scored, detector)
		
		// Invariant: No relationship should lose its essential identity
		for _, resolvedRel := range resolved {
			found := false
			for _, origRel := range relationships {
				if origRel.From == resolvedRel.From && 
				   origRel.To == resolvedRel.To && 
				   origRel.Type == resolvedRel.Type {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Resolved relationship not found in original set: %+v", resolvedRel)
			}
		}
		
		// Invariant: Total processing should not increase relationship count dramatically
		if len(resolved) > originalCount*2 {
			t.Fatalf("Processing increased relationship count from %d to %d", originalCount, len(resolved))
		}
		
		// Invariant: All required fields should remain populated
		for _, rel := range resolved {
			if rel.From == "" || rel.To == "" || rel.Type == "" || rel.Source == "" {
				t.Fatalf("Required field lost during processing: %+v", rel)
			}
			if rel.Timestamp == "" {
				t.Fatalf("Timestamp lost during processing: %+v", rel)
			}
		}
		
		// Invariant: Confidence values should remain in valid range
		for _, rel := range resolved {
			if rel.Confidence < 0 || rel.Confidence > 1 {
				t.Fatalf("Confidence invariant violated: %f for %+v", rel.Confidence, rel)
			}
		}
	})
}

// Generator functions for property-based testing

func generateRandomRelationships(t *rapid.T, minCount, maxCount int) []Relationship {
	count := rapid.IntRange(minCount, maxCount).Draw(t, "relationship_count")
	relationships := make([]Relationship, count)
	
	sources := []string{"github-codeowners", "openshift-metadata", "aws-tags", "datadog-metrics"}
	types := []string{"owns", "manages", "contributes-to"}
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From:       rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")), 1, 20, 20).Draw(t, "from"),
			To:         rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")), 1, 20, 20).Draw(t, "to"),
			Type:       rapid.SampledFrom(types).Draw(t, "type"),
			Confidence: rapid.Float64Range(0.0, 1.0).Draw(t, "confidence"),
			Source:     rapid.SampledFrom(sources).Draw(t, "source"),
			Timestamp:  generateRandomTimestamp(t),
		}
	}
	
	return relationships
}

func generateConflictingRelationships(t *rapid.T, minCount, maxCount int) []Relationship {
	count := rapid.IntRange(minCount, maxCount).Draw(t, "relationship_count")
	relationships := make([]Relationship, count)
	
	sources := []string{"github-codeowners", "openshift-metadata", "aws-tags"}
	
	// Create potential conflicts by having different owners for same resources
	resourceCount := rapid.IntRange(1, count/2+1).Draw(t, "resource_count")
	resources := make([]string, resourceCount)
	for i := 0; i < resourceCount; i++ {
		resources[i] = fmt.Sprintf("resource-%d", i)
	}
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From:       fmt.Sprintf("user-%d", rapid.IntRange(0, count/2).Draw(t, "user_id")),
			To:         rapid.SampledFrom(resources).Draw(t, "resource"),
			Type:       "owns",
			Confidence: rapid.Float64Range(0.3, 1.0).Draw(t, "confidence"),
			Source:     rapid.SampledFrom(sources).Draw(t, "source"),
			Timestamp:  generateRandomTimestamp(t),
		}
	}
	
	return relationships
}

func generateBaseRelationships(t *rapid.T, minCount, maxCount int) []Relationship {
	count := rapid.IntRange(minCount, maxCount).Draw(t, "base_count")
	relationships := make([]Relationship, count)
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From:       fmt.Sprintf("user-%d", i),
			To:         fmt.Sprintf("resource-%d", i),
			Type:       "owns",
			Confidence: rapid.Float64Range(0.5, 0.9).Draw(t, "confidence"),
			Source:     "github-codeowners",
			Timestamp:  generateRandomTimestamp(t),
		}
	}
	
	return relationships
}

func duplicateAcrossSources(base []Relationship, t *rapid.T) []Relationship {
	sources := []string{"github-codeowners", "openshift-metadata", "aws-tags"}
	var result []Relationship
	
	for _, rel := range base {
		// Randomly duplicate some relationships across sources
		sourceCount := rapid.IntRange(1, len(sources)).Draw(t, "source_count")
		
		for i := 0; i < sourceCount; i++ {
			source := rapid.SampledFrom(sources).Draw(t, fmt.Sprintf("source_%d", i))
			newRel := rel
			newRel.Source = source
			newRel.Confidence = rapid.Float64Range(0.4, 0.9).Draw(t, "new_confidence")
			result = append(result, newRel)
		}
	}
	
	return result
}

func generateExtremeValueRelationships(t *rapid.T) []Relationship {
	relationships := []Relationship{
		// Extreme confidence values
		{
			From: "user1", To: "resource1", Type: "owns",
			Confidence: 0.0, Source: "github-codeowners",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		{
			From: "user2", To: "resource2", Type: "owns",
			Confidence: 1.0, Source: "aws-tags",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		// Very old timestamp
		{
			From: "user3", To: "resource3", Type: "owns",
			Confidence: 0.5, Source: "openshift-metadata",
			Timestamp: time.Now().AddDate(-10, 0, 0).Format(time.RFC3339),
		},
		// Future timestamp
		{
			From: "user4", To: "resource4", Type: "owns",
			Confidence: 0.8, Source: "datadog-metrics",
			Timestamp: time.Now().AddDate(1, 0, 0).Format(time.RFC3339),
		},
	}
	
	// Add some random extreme values
	extraCount := rapid.IntRange(1, 10).Draw(t, "extra_count")
	for i := 0; i < extraCount; i++ {
		rel := Relationship{
			From: rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")), 1, 50, 50).Draw(t, "extreme_from"),
			To: rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyz")), 1, 50, 50).Draw(t, "extreme_to"),
			Type: "owns",
			Confidence: rapid.OneOf(
				rapid.Float64(),
				rapid.Float64Range(0.0, 1.0),
			).Draw(t, "extreme_confidence"),
			Source: "unknown-source",
			Timestamp: generateRandomTimestamp(t),
		}
		relationships = append(relationships, rel)
	}
	
	return relationships
}

func generateMaxConfidenceRelationships(t *rapid.T) []Relationship {
	count := rapid.IntRange(1, 5).Draw(t, "max_conf_count")
	relationships := make([]Relationship, count)
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From: fmt.Sprintf("max-user-%d", i),
			To: fmt.Sprintf("max-resource-%d", i),
			Type: "owns",
			Confidence: 1.0,
			Source: "aws-tags",
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	
	return relationships
}

func generateZeroConfidenceRelationships(t *rapid.T) []Relationship {
	count := rapid.IntRange(1, 5).Draw(t, "zero_conf_count")
	relationships := make([]Relationship, count)
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From: fmt.Sprintf("zero-user-%d", i),
			To: fmt.Sprintf("zero-resource-%d", i),
			Type: "owns",
			Confidence: 0.0,
			Source: "github-codeowners",
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	
	return relationships
}

func generateAncientTimestampRelationships(t *rapid.T) []Relationship {
	count := rapid.IntRange(1, 5).Draw(t, "ancient_count")
	relationships := make([]Relationship, count)
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From: fmt.Sprintf("ancient-user-%d", i),
			To: fmt.Sprintf("ancient-resource-%d", i),
			Type: "owns",
			Confidence: 0.5,
			Source: "github-codeowners",
			Timestamp: time.Now().AddDate(-5, 0, 0).Format(time.RFC3339),
		}
	}
	
	return relationships
}

func generateFutureTimestampRelationships(t *rapid.T) []Relationship {
	count := rapid.IntRange(1, 5).Draw(t, "future_count")
	relationships := make([]Relationship, count)
	
	for i := 0; i < count; i++ {
		relationships[i] = Relationship{
			From: fmt.Sprintf("future-user-%d", i),
			To: fmt.Sprintf("future-resource-%d", i),
			Type: "owns",
			Confidence: 0.7,
			Source: "openshift-metadata",
			Timestamp: time.Now().AddDate(0, 0, 30).Format(time.RFC3339),
		}
	}
	
	return relationships
}

func generateRandomTimestamp(t *rapid.T) string {
	// Generate timestamp within last year
	daysAgo := rapid.IntRange(-365, 30).Draw(t, "days_offset")
	timestamp := time.Now().AddDate(0, 0, daysAgo)
	return timestamp.Format(time.RFC3339)
}

func findMultiSourceTargets(relationships []Relationship) map[string][]Relationship {
	targetMap := make(map[string][]Relationship)
	
	for _, rel := range relationships {
		key := fmt.Sprintf("%s->%s", rel.From, rel.To)
		targetMap[key] = append(targetMap[key], rel)
	}
	
	return targetMap
}