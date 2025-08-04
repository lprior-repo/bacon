#!/bin/bash

# Build all Lambda functions in the DDD plugin architecture
echo "🏗️  Building Lambda functions..."

# Find all Lambda function directories
lambda_dirs=$(find src -name "main.go" -path "*/lambda/*" -type f | xargs dirname)

if [ -z "$lambda_dirs" ]; then
    echo "❌ No Lambda functions found"
    exit 1
fi

# Build configuration
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
export GO111MODULE=on

# Track build results
success_count=0
total_count=0
failed_builds=()

echo "📦 Found Lambda functions:"
for dir in $lambda_dirs; do
    function_name=$(basename "$dir")
    echo "  - $function_name (in $dir)"
    ((total_count++))
done

echo ""
echo "🔨 Building Lambda functions..."

# Build each Lambda function
for dir in $lambda_dirs; do
    function_name=$(basename "$dir")
    echo -n "Building $function_name... "
    
    # Build the Lambda function
    if (cd "$dir" && go build -ldflags "-s -w" -trimpath -o main .); then
        echo "✅"
        ((success_count++))
    else
        echo "❌"
        failed_builds+=("$function_name")
    fi
done

echo ""
echo "📊 Build Summary:"
echo "  ✅ Successful: $success_count/$total_count"
echo "  ❌ Failed: $((total_count - success_count))"

if [ ${#failed_builds[@]} -gt 0 ]; then
    echo ""
    echo "❌ Failed builds:"
    for failed in "${failed_builds[@]}"; do
        echo "  - $failed"
    done
    exit 1
fi

echo ""
echo "🎉 All Lambda functions built successfully!"