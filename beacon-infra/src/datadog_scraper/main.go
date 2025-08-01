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
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-xray-sdk-go/xray"
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
    ctx, seg := xray.BeginSubsegment(ctx, "datadog-scraper-handler")
    defer seg.Close(nil)

    seg.AddAnnotation("metric_name", event.MetricName)
    seg.AddAnnotation("time_range", event.TimeRange)

    metrics, err := fetchDatadogMetrics(ctx, event.MetricName, event.TimeRange)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(err.Error()), err
    }

    err = storeMetricsData(ctx, metrics)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(fmt.Sprintf("failed to store metrics: %v", err)), err
    }

    seg.AddMetadata("metrics_data", map[string]interface{}{
        "point_count": len(metrics.Points),
        "metric_name": metrics.MetricName,
    })

    return createSuccessResponse("Datadog metrics scraped successfully"), nil
}

func fetchDatadogMetrics(ctx context.Context, metricName, timeRange string) (*DatadogMetric, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "fetch-datadog-metrics")
    defer seg.Close(nil)

    apiKey := os.Getenv("DATADOG_API_KEY")
    appKey := os.Getenv("DATADOG_APP_KEY")
    
    if apiKey == "" || appKey == "" {
        err := fmt.Errorf("missing Datadog API credentials")
        seg.AddError(err)
        return nil, err
    }

    url := fmt.Sprintf("https://api.datadoghq.com/api/v1/query?query=%s&from=%s&to=now", 
                      metricName, timeRange)
    seg.AddAnnotation("datadog_url", url)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("DD-API-KEY", apiKey)
    req.Header.Set("DD-APPLICATION-KEY", appKey)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to fetch metrics: %w", err)
    }
    defer resp.Body.Close()

    seg.AddAnnotation("response_status", resp.StatusCode)

    var result struct {
        Series []DatadogMetric `json:"series"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    if len(result.Series) == 0 {
        err := fmt.Errorf("no metrics found")
        seg.AddError(err)
        return nil, err
    }

    return &result.Series[0], nil
}

func storeMetricsData(ctx context.Context, metric *DatadogMetric) error {
    ctx, seg := xray.BeginSubsegment(ctx, "store-metrics-data")
    defer seg.Close(nil)

    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        seg.AddError(err)
        return fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := dynamodb.NewFromConfig(cfg)
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        tableName = "datadog-metrics"
    }

    seg.AddAnnotation("table_name", tableName)
    seg.AddAnnotation("point_count", len(metric.Points))

    for _, point := range metric.Points {
        item := map[string]types.AttributeValue{
            "metric_name": &types.AttributeValueMemberS{Value: metric.MetricName},
            "timestamp":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", point.Timestamp)},
            "value":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", point.Value)},
            "scraped_at":  &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
        }

        _, err = client.PutItem(ctx, &dynamodb.PutItemInput{
            TableName: aws.String(tableName),
            Item:      item,
        })

        if err != nil {
            seg.AddError(err)
            return fmt.Errorf("failed to store metric point: %w", err)
        }
    }

    return nil
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