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
    return withTracedSubsegment(ctx, "process-datadog-event", func(tracedCtx context.Context, seg *xray.Segment) (DatadogProcessingResult, error) {
        seg.AddAnnotation("metric_name", event.MetricName)
        seg.AddAnnotation("time_range", event.TimeRange)
        
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
        
        // Add metadata
        seg.AddMetadata("metrics_data", map[string]interface{}{
            "point_count": len(metric.Points),
            "metric_name": metric.MetricName,
        })
        
        return DatadogProcessingResult{Metric: metric}, nil
    })
}

// Higher-order function for tracing operations
func withTracedOperation[T any](ctx context.Context, operationName string, operation func(context.Context) (T, error)) (T, error) {
    ctx, seg := xray.BeginSubsegment(ctx, operationName)
    defer seg.Close(nil)
    
    result, err := operation(ctx)
    if err != nil {
        seg.AddError(err)
    }
    
    return result, err
}

func withTracedSubsegment[T any](ctx context.Context, segmentName string, operation func(context.Context, *xray.Segment) (T, error)) (T, error) {
    ctx, seg := xray.BeginSubsegment(ctx, segmentName)
    defer seg.Close(nil)
    
    result, err := operation(ctx, seg)
    if err != nil {
        seg.AddError(err)
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