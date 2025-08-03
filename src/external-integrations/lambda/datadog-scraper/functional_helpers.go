// Package main provides functional helpers for Datadog metric scraping and processing.
package main

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-xray-sdk-go/v2/xray"
)

// DatadogProcessingResult represents the result of processing Datadog events
type DatadogProcessingResult struct {
    Metric *DatadogMetric
    Error  error
}

func (r DatadogProcessingResult) IsSuccess() bool {
    return r.Error == nil
}

func (r DatadogProcessingResult) IsFailure() bool {
    return r.Error != nil
}

// Functional composition for Datadog event processing
func processDatadogEvent(ctx context.Context, event DatadogEvent) DatadogProcessingResult {
    result, err := withTracedOperation(ctx, "process-datadog-event", func(tracedCtx context.Context) (DatadogProcessingResult, error) {
        // Fetch metrics using functional approach
        metric, err := fetchDatadogMetrics(tracedCtx, event.MetricName, event.TimeRange)
        if err != nil {
            return DatadogProcessingResult{Error: err}, err
        }
        
        // Store metrics using functional approach
        err = storeMetricsData(tracedCtx, metric)
        if err != nil {
            return DatadogProcessingResult{Error: fmt.Errorf("failed to store metrics: %w", err)}, err
        }
        
        return DatadogProcessingResult{Metric: metric}, nil
    })
    
    if err != nil {
        return DatadogProcessingResult{Error: err}
    }
    
    return result
}

// Higher-order function for tracing operations
func withTracedOperation[T any](ctx context.Context, operationName string, operation func(context.Context) (T, error)) (T, error) {
    ctx, seg := xray.BeginSubsegment(ctx, operationName)
    defer seg.Close(nil)
    
    result, err := operation(ctx)
    if err != nil {
        _ = seg.AddError(err)
    }
    
    return result, err
}


// Pure functional operations for metric storage
type StoreOperation struct {
    TableName string
    Item      map[string]types.AttributeValue
}

func mapPointsToStoreOperations(points []Point, metricName string) []StoreOperation {
    operations := make([]StoreOperation, len(points))
    for i, point := range points {
        operations[i] = StoreOperation{
            Item: createMetricItem(metricName, point),
        }
    }
    return operations
}

func executeStoreOperations(ctx context.Context, client *dynamodb.Client, tableName string, operations []StoreOperation) error {
    for _, op := range operations {
        _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
            TableName: aws.String(tableName),
            Item:      op.Item,
        })
        if err != nil {
            return fmt.Errorf("failed to store metric point: %w", err)
        }
    }
    return nil
}