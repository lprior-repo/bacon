package common

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
)

func LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return cfg, nil
}

func CreateDynamoClient(cfg aws.Config) *dynamodb.Client {
	return dynamodb.NewFromConfig(cfg)
}

func CreateSecretsClient(cfg aws.Config) *secretsmanager.Client {
	return secretsmanager.NewFromConfig(cfg)
}

func LoadAWSConfigWithTracing(ctx context.Context, segmentName string) (aws.Config, error) {
	_, seg := xray.BeginSubsegment(ctx, segmentName)
	defer seg.Close(nil)

	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		seg.AddError(err)
		return aws.Config{}, err
	}

	seg.AddAnnotation("aws_config_loaded", true)
	return cfg, nil
}