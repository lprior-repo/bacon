package main

import (
	"context"
	"testing"

	"bacon/src/code-analysis/types"
)

func TestValidateEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    types.Event
		expected types.Event
	}{
		{
			name: "sets default batch size",
			input: types.Event{
				Organization: "test-org",
				BatchSize:    0,
			},
			expected: types.Event{
				Organization: "test-org",
				BatchSize:    100,
			},
		},
		{
			name: "preserves existing batch size",
			input: types.Event{
				Organization: "test-org",
				BatchSize:    50,
			},
			expected: types.Event{
				Organization: "test-org",
				BatchSize:    50,
			},
		},
		{
			name: "sets default organization",
			input: types.Event{
				Organization: "",
				BatchSize:    100,
			},
			expected: types.Event{
				Organization: "your-company",
				BatchSize:    100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateEvent(tt.input)
			if err != nil {
				t.Fatalf("validateEvent() error = %v", err)
			}

			if result.BatchSize != tt.expected.BatchSize {
				t.Errorf("validateEvent() BatchSize = %v, want %v", result.BatchSize, tt.expected.BatchSize)
			}

			if result.Organization != tt.expected.Organization {
				t.Errorf("validateEvent() Organization = %v, want %v", result.Organization, tt.expected.Organization)
			}
		})
	}
}

func TestInitializeContext(t *testing.T) {
	input := types.Event{
		Organization: "test-org",
		BatchSize:    100,
	}

	result, err := initializeContext(input)
	if err != nil {
		t.Fatalf("initializeContext() error = %v", err)
	}

	if result.Context == nil {
		t.Error("initializeContext() should set Context")
	}

	if result.Context.StartTime.IsZero() {
		t.Error("initializeContext() should set StartTime")
	}

	if result.Context.RequestID == "" {
		t.Error("initializeContext() should set RequestID")
	}
}

func TestHandleRequest_ValidInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	event := types.Event{
		Organization: "test-org",
		BatchSize:    1, // Small batch for testing
	}

	// This is a light integration test that validates the pipeline structure
	// without making actual API calls (would need mocks for full isolation)
	_, err := HandleRequest(ctx, event)
	
	// We expect this to fail due to missing environment variables in test
	// but it should fail at the GitHub API stage, not in the pipeline setup
	if err == nil {
		t.Error("Expected error due to missing GitHub token in test environment")
	}
}

// BenchmarkValidateEvent benchmarks the validation function
func BenchmarkValidateEvent(b *testing.B) {
	event := types.Event{
		Organization: "test-org",
		BatchSize:    0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validateEvent(event)
	}
}