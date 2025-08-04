module bacon

replace bacon/src/shared => ./src/shared

replace bacon/src/code-analysis/types => ./src/code-analysis/types

replace bacon/src/code-analysis/parsers => ./src/code-analysis/parsers

replace bacon/src/code-analysis/clients => ./src/code-analysis/clients

replace bacon/src/code-analysis/cache => ./src/code-analysis/cache

go 1.24.5

require (
	bacon/src/code-analysis/cache v0.0.0-00010101000000-000000000000
	bacon/src/code-analysis/clients v0.0.0-00010101000000-000000000000
	bacon/src/code-analysis/parsers v0.0.0-00010101000000-000000000000
	bacon/src/code-analysis/types v0.0.0
	bacon/src/shared v0.0.0
	github.com/aws/aws-lambda-go v1.49.0
	github.com/aws/aws-sdk-go-v2 v1.37.1
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.6
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.25.5
	github.com/aws/aws-xray-sdk-go/v2 v2.0.0
	github.com/magefile/mage v1.15.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.8.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5 // indirect
	github.com/aws/smithy-go v1.22.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.52.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/grpc v1.64.1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	pgregory.net/rapid v1.2.0 // indirect
)
