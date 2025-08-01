package parsers

import (
	"testing"

	"bacon/src/code-analysis/types"
)

func TestParseCodeowners(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		repository   string
		expectedLen  int
		expectedPath string
		expectedTeam string
	}{
		{
			name: "simple codeowners file",
			content: `# Global owners
* @global-team

# Frontend owners
/frontend/ @frontend-team

# API owners
/api/ @backend-team @security-team`,
			repository:   "test-repo",
			expectedLen:  3,
			expectedPath: "*",
			expectedTeam: "@global-team",
		},
		{
			name: "empty file",
			content: `# Just comments
# No actual rules`,
			repository:  "test-repo",
			expectedLen: 0,
		},
		{
			name: "complex patterns",
			content: `# Complex patterns
*.js @frontend-team
**/*.go @backend-team
docs/ @docs-team @product-team
/config/*.yml @devops-team`,
			repository:  "test-repo",
			expectedLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodeowners(tt.content, tt.repository)

			if len(result) != tt.expectedLen {
				t.Errorf("ParseCodeowners() returned %d entries, want %d", len(result), tt.expectedLen)
			}

			if tt.expectedLen > 0 {
				firstEntry := result[0]
				if firstEntry.Path != tt.expectedPath {
					t.Errorf("ParseCodeowners() first entry path = %v, want %v", firstEntry.Path, tt.expectedPath)
				}
				if firstEntry.Repository != tt.repository {
					t.Errorf("ParseCodeowners() first entry repository = %v, want %v", firstEntry.Repository, tt.repository)
				}
			}
		})
	}
}

func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		content1 string
		content2 string
		same     bool
	}{
		{
			name:     "identical content",
			content1: "* @team",
			content2: "* @team",
			same:     true,
		},
		{
			name:     "different content",
			content1: "* @team1",
			content2: "* @team2",
			same:     false,
		},
		{
			name:     "empty content",
			content1: "",
			content2: "",
			same:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := CalculateHash(tt.content1)
			hash2 := CalculateHash(tt.content2)

			if tt.same && hash1 != hash2 {
				t.Errorf("CalculateHash() hashes should be same: %v vs %v", hash1, hash2)
			}

			if !tt.same && hash1 == hash2 {
				t.Errorf("CalculateHash() hashes should be different: %v vs %v", hash1, hash2)
			}

			// Hash should be consistent
			hash1Again := CalculateHash(tt.content1)
			if hash1 != hash1Again {
				t.Errorf("CalculateHash() should be consistent: %v vs %v", hash1, hash1Again)
			}
		})
	}
}

func TestParseCodeowners_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name:    "only comments",
			content: "# This is just a comment\n# Another comment",
			wantLen: 0,
		},
		{
			name:    "empty lines",
			content: "\n\n\n",
			wantLen: 0,
		},
		{
			name:    "mixed valid and invalid lines",
			content: "* @team\n# comment\n\ninvalid-line-without-owner\n/path/ @another-team",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodeowners(tt.content, "test-repo")
			if len(result) != tt.wantLen {
				t.Errorf("ParseCodeowners() = %v entries, want %v", len(result), tt.wantLen)
			}
		})
	}
}

// BenchmarkParseCodeowners benchmarks the parser with a realistic CODEOWNERS file
func BenchmarkParseCodeowners(b *testing.B) {
	content := `# Global owners
* @global-team

# Frontend
/frontend/ @frontend-team
*.js @frontend-team
*.jsx @frontend-team
*.ts @frontend-team
*.tsx @frontend-team

# Backend
/backend/ @backend-team
*.go @backend-team
*.py @backend-team

# DevOps
/terraform/ @devops-team
/k8s/ @devops-team
*.yml @devops-team
*.yaml @devops-team
Dockerfile* @devops-team

# Documentation
/docs/ @docs-team
*.md @docs-team
README* @docs-team`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseCodeowners(content, "benchmark-repo")
	}
}

func BenchmarkCalculateHash(b *testing.B) {
	content := "This is a sample CODEOWNERS content with some text to hash"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateHash(content)
	}
}