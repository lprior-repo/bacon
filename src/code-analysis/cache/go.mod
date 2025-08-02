module bacon/src/code-analysis/cache

go 1.22

require (
	bacon/src/code-analysis/types v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.37.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.6
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.8.10 // indirect
	github.com/aws/smithy-go v1.22.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)

replace bacon/src/code-analysis/types => ../types

replace bacon/src/shared => ../../shared
