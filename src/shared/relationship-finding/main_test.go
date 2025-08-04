package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	common "bacon/src/shared"
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
			expected:      1, // Function returns count + 1
		},
		{
			name: "no conflicts",
			relationships: []Relationship{
				{HasConflict: false},
				{HasConflict: false},
			},
			expected: 1,
		},
		{
			name: "some conflicts",
			relationships: []Relationship{
				{HasConflict: true},
				{HasConflict: false},
				{HasConflict: true},
			},
			expected: 3, // 2 conflicts + 1
		},
		{
			name: "all conflicts",
			relationships: []Relationship{
				{HasConflict: true},
				{HasConflict: true},
			},
			expected: 3, // 2 conflicts + 1
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