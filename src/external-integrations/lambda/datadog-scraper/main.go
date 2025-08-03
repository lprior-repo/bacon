package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DatadogEvent struct {
    MetricName string `json:"metric_name"`
    TimeRange  string `json:"time_range"`
}

type DatadogResponse struct {
    Status    string `json:"status"`
    Message   string `json:"message"`
    Timestamp string `json:"timestamp"`
}

type DatadogMetric struct {
    MetricName string  `json:"metric"`
    Points     []Point `json:"pointlist"`
}

type Point struct {
    Timestamp int64   `json:"timestamp"`
    Value     float64 `json:"value"`
}

func main() {
    lambda.Start(handleDatadogScrapeRequest)
}

func handleDatadogScrapeRequest(ctx context.Context, event DatadogEvent) (DatadogResponse, error) {
    return withTracedOperation(ctx, "datadog-scraper-handler", func(tracedCtx context.Context) (DatadogResponse, error) {
        // Functional pipeline for processing
        result := processDatadogEvent(tracedCtx, event)
        if result.IsFailure() {
            return createErrorResponse(result.Error.Error()), result.Error
        }
        
        return createSuccessResponse("Datadog metrics scraped successfully"), nil
    })
}

// Pure functions for building Datadog requests
func buildDatadogURL(metricName, timeRange string) string {
    return fmt.Sprintf("https://api.datadoghq.com/api/v1/query?query=%s&from=%s&to=now", metricName, timeRange)
}

func getDatadogCredentials() (string, string, error) {
    apiKey := os.Getenv("DATADOG_API_KEY")
    appKey := os.Getenv("DATADOG_APP_KEY")
    
    if apiKey == "" || appKey == "" {
        return "", "", fmt.Errorf("missing Datadog API credentials")
    }
    
    return apiKey, appKey, nil
}

func createDatadogRequest(ctx context.Context, url, apiKey, appKey string) (*http.Request, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("DD-API-KEY", apiKey)
    req.Header.Set("DD-APPLICATION-KEY", appKey)
    req.Header.Set("Content-Type", "application/json")
    
    return req, nil
}

func decodeDatadogResponse(resp *http.Response) (*DatadogMetric, error) {
    defer func() {
        if err := resp.Body.Close(); err != nil {
            // Log but don't fail on close error
        }
    }()
    
    var result struct {
        Series []DatadogMetric `json:"series"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    if len(result.Series) == 0 {
        return nil, fmt.Errorf("no metrics found")
    }
    
    return &result.Series[0], nil
}

func fetchDatadogMetrics(ctx context.Context, metricName, timeRange string) (*DatadogMetric, error) {
    return withTracedOperation(ctx, "fetch-datadog-metrics", func(tracedCtx context.Context) (*DatadogMetric, error) {
        apiKey, appKey, err := getDatadogCredentials()
        if err != nil {
            return nil, err
        }
        
        url := buildDatadogURL(metricName, timeRange)
        
        req, err := createDatadogRequest(tracedCtx, url, apiKey, appKey)
        if err != nil {
            return nil, err
        }
        
        client := &http.Client{Timeout: 30 * time.Second}
        resp, err := client.Do(req)
        if err != nil {
            return nil, fmt.Errorf("failed to fetch metrics: %w", err)
        }
        
        return decodeDatadogResponse(resp)
    })
}

// Pure function for creating metric items
func createMetricItem(metricName string, point Point) map[string]types.AttributeValue {
    return map[string]types.AttributeValue{
        "metric_name": &types.AttributeValueMemberS{Value: metricName},
        "timestamp":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", point.Timestamp)},
        "value":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", point.Value)},
        "scraped_at":  &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
    }
}

func getMetricsTableName() string {
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        return "datadog-metrics"
    }
    return tableName
}

func storeMetricsData(ctx context.Context, metric *DatadogMetric) error {
    err, _ := withTracedOperation(ctx, "store-metrics-data", func(tracedCtx context.Context) (error, error) {
        cfg, err := config.LoadDefaultConfig(tracedCtx)
        if err != nil {
            return fmt.Errorf("failed to load AWS config: %w", err), err
        }
        
        client := dynamodb.NewFromConfig(cfg)
        tableName := getMetricsTableName()
        
        // Functional approach: map points to storage operations
        storeOperations := mapPointsToStoreOperations(metric.Points, metric.MetricName)
        err = executeStoreOperations(tracedCtx, client, tableName, storeOperations)
        return err, err
    })
    return err
}

func createSuccessResponse(message string) DatadogResponse {
    return DatadogResponse{
        Status:    "success",
        Message:   message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

func createErrorResponse(message string) DatadogResponse {
    log.Printf("Error: %s", message)
    return DatadogResponse{
        Status:    "error",
        Message:   message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}