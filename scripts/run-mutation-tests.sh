#!/bin/bash

# Mutation Testing Runner for Bacon Project
# Uses go-mutesting from Avito-tech to achieve 95% mutation catching threshold

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MUTATION_THRESHOLD=95
OUTPUT_DIR="./mutation-results"
CONFIG_FILE="./mutesting.config.yaml"

echo -e "${BLUE}🧬 Starting Mutation Testing for Bacon Project${NC}"
echo -e "${BLUE}Target: ${MUTATION_THRESHOLD}% mutation catching threshold${NC}"
echo ""

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Function to run mutation testing for a specific module
run_mutation_test() {
    local module_path="$1"
    local module_name=$(basename "$module_path")
    
    echo -e "${YELLOW}🔬 Testing module: ${module_name} (${module_path})${NC}"
    
    # Check if module has tests
    if ! find "$module_path" -name "*_test.go" -type f | grep -q .; then
        echo -e "${YELLOW}⚠️  No tests found in ${module_path}, skipping mutation testing${NC}"
        return 0
    fi
    
    # Create module-specific output directory
    local module_output_dir="${OUTPUT_DIR}/${module_name}"
    mkdir -p "$module_output_dir"
    
    # Run mutation testing for this module
    local mutation_output="${module_output_dir}/mutations.log"
    local summary_output="${module_output_dir}/summary.txt"
    
    echo "Running mutation testing for ${module_path}..." > "$mutation_output"
    
    # Change to module directory and run mutation testing
    pushd "$module_path" > /dev/null
    
    # Run go-mutesting with proper configuration
    if go-mutesting \
        --verbose \
        --exec-timeout=60 \
        --test-recursive \
        --do-not-remove-tmp-folder \
        . >> "$mutation_output" 2>&1; then
        
        echo -e "${GREEN}✅ Mutation testing completed for ${module_name}${NC}"
        
        # Extract mutation score from output
        local mutation_score
        if mutation_score=$(grep -o "The mutation score is [0-9.]*%" "$mutation_output" | tail -1 | grep -o "[0-9.]*"); then
            echo "Module: ${module_name}" > "$summary_output"
            echo "Path: ${module_path}" >> "$summary_output"
            echo "Mutation Score: ${mutation_score}%" >> "$summary_output"
            echo "Threshold: ${MUTATION_THRESHOLD}%" >> "$summary_output"
            
            # Check if score meets threshold
            if (( $(echo "${mutation_score} >= ${MUTATION_THRESHOLD}" | bc -l) )); then
                echo "Status: PASSED ✅" >> "$summary_output"
                echo -e "${GREEN}🎯 ${module_name}: ${mutation_score}% (PASSED)${NC}"
            else
                echo "Status: FAILED ❌" >> "$summary_output"
                echo -e "${RED}❌ ${module_name}: ${mutation_score}% (BELOW THRESHOLD)${NC}"
            fi
        else
            echo "Status: ERROR - Could not parse mutation score" >> "$summary_output"
            echo -e "${RED}❌ ${module_name}: Could not determine mutation score${NC}"
        fi
    else
        echo -e "${RED}❌ Mutation testing failed for ${module_name}${NC}"
        echo "Status: ERROR - Mutation testing failed" >> "$summary_output"
    fi
    
    popd > /dev/null
    echo ""
}

# Function to generate overall report
generate_report() {
    local report_file="${OUTPUT_DIR}/mutation-test-report.md"
    
    echo "# Mutation Testing Report" > "$report_file"
    echo "" >> "$report_file"
    echo "Generated on: $(date)" >> "$report_file"
    echo "Target Threshold: ${MUTATION_THRESHOLD}%" >> "$report_file"
    echo "" >> "$report_file"
    
    echo "## Summary" >> "$report_file"
    echo "" >> "$report_file"
    
    local total_modules=0
    local passed_modules=0
    local failed_modules=0
    local error_modules=0
    
    # Process all summary files
    for summary_file in "${OUTPUT_DIR}"/*/summary.txt; do
        if [[ -f "$summary_file" ]]; then
            total_modules=$((total_modules + 1))
            
            local module_name=$(grep "Module:" "$summary_file" | cut -d' ' -f2)
            local mutation_score=$(grep "Mutation Score:" "$summary_file" | grep -o "[0-9.]*")
            local status=$(grep "Status:" "$summary_file" | cut -d' ' -f2-)
            
            echo "| ${module_name} | ${mutation_score}% | ${status} |" >> "$report_file"
            
            if grep -q "PASSED" "$summary_file"; then
                passed_modules=$((passed_modules + 1))
            elif grep -q "FAILED" "$summary_file"; then
                failed_modules=$((failed_modules + 1))
            else
                error_modules=$((error_modules + 1))
            fi
        fi
    done
    
    # Add table header
    sed -i '7i| Module | Score | Status |' "$report_file"
    sed -i '8i|---------|-------|--------|' "$report_file"
    
    echo "" >> "$report_file"
    echo "## Statistics" >> "$report_file"
    echo "- Total Modules: ${total_modules}" >> "$report_file"
    echo "- Passed: ${passed_modules}" >> "$report_file"
    echo "- Failed: ${failed_modules}" >> "$report_file"
    echo "- Errors: ${error_modules}" >> "$report_file"
    
    local success_rate=0
    if [[ $total_modules -gt 0 ]]; then
        success_rate=$(echo "scale=1; $passed_modules * 100 / $total_modules" | bc)
    fi
    echo "- Success Rate: ${success_rate}%" >> "$report_file"
    
    echo "" >> "$report_file"
    echo "## Detailed Results" >> "$report_file"
    echo "" >> "$report_file"
    echo "Check individual module reports in the \`mutation-results/\` directory." >> "$report_file"
    
    echo -e "${BLUE}📊 Report generated: ${report_file}${NC}"
    echo -e "${BLUE}📈 Success Rate: ${success_rate}% (${passed_modules}/${total_modules} modules)${NC}"
}

# Main execution
main() {
    # Check if go-mutesting is installed
    if ! command -v go-mutesting &> /dev/null; then
        echo -e "${RED}❌ go-mutesting is not installed. Please run: go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest${NC}"
        exit 1
    fi
    
    # Check if bc is installed (for floating point arithmetic)
    if ! command -v bc &> /dev/null; then
        echo -e "${YELLOW}⚠️  bc is not installed. Installing...${NC}"
        sudo apt-get update && sudo apt-get install -y bc
    fi
    
    echo -e "${BLUE}🔍 Discovering Go modules...${NC}"
    
    # List of all Go modules to test
    local modules=(
        "src/code-analysis/cache"
        "src/code-analysis/clients"
        "src/code-analysis/parsers" 
        "src/code-analysis/types"
        "src/code-analysis/lambda/codeowners-scraper"
        "src/code-analysis/lambda/github-scraper"
        "src/data-processing/lambda/event-processor"
        "src/external-integrations/lambda/datadog-scraper"
        "src/external-integrations/lambda/openshift-scraper"
        "src/graphql-api/resolvers/mutation"
        "src/graphql-api/resolvers/query"
        "src/shared"
    )
    
    echo -e "${BLUE}Found ${#modules[@]} modules to test${NC}"
    echo ""
    
    # Run mutation testing for each module
    for module in "${modules[@]}"; do
        if [[ -d "$module" ]]; then
            run_mutation_test "$module"
        else
            echo -e "${YELLOW}⚠️  Module not found: ${module}${NC}"
        fi
    done
    
    # Generate comprehensive report
    generate_report
    
    echo -e "${GREEN}🎉 Mutation testing completed!${NC}"
    echo -e "${BLUE}📄 Check ${OUTPUT_DIR}/mutation-test-report.md for full results${NC}"
}

# Run main function
main "$@"