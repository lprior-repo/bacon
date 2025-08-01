module bacon/src/code-analysis/cache

go 1.22

require (
	bacon/src/code-analysis/types v0.0.0
	bacon/src/shared v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.37.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.6
)

replace bacon/src/code-analysis/types => ../types
replace bacon/src/shared => ../../shared