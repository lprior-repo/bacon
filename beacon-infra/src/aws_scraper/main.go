package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
    "github.com/aws/aws-xray-sdk-go/xray"
)

type AWSEvent struct {
    Service    string `json:"service"`
    MetricName string `json:"metric_name"`
    Namespace  string `json:"namespace"`
}

type AWSResponse struct {
    Status    string `json:"status"`
    Message   string `json:"message"`
    Timestamp string `json:"timestamp"`
}

type AWSMetric struct {
    MetricName string                   `json:"metric_name"`
    Namespace  string                   `json:"namespace"`
    Datapoints []types.Datapoint        `json:"datapoints"`
    Instances  []ec2Types.Instance      `json:"instances,omitempty"`
}

func main() {
    lambda.Start(handleAWSScrapeRequest)
}

func handleAWSScrapeRequest(ctx context.Context, event AWSEvent) (AWSResponse, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "aws-scraper-handler")
    defer seg.Close(nil)

    seg.AddAnnotation("service", event.Service)
    seg.AddAnnotation("metric_name", event.MetricName)
    seg.AddAnnotation("namespace", event.Namespace)

    data, err := fetchAWSData(ctx, event.Service, event.MetricName, event.Namespace)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(err.Error()), err
    }

    err = storeAWSData(ctx, data)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(fmt.Sprintf("failed to store AWS data: %v", err)), err
    }

    seg.AddMetadata("aws_data", map[string]interface{}{
        "datapoint_count": len(data.Datapoints),
        "instance_count":  len(data.Instances),
        "service":         event.Service,
    })

    return createSuccessResponse("AWS data scraped successfully"), nil
}

func fetchAWSData(ctx context.Context, service, metricName, namespace string) (*AWSMetric, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "fetch-aws-data")
    defer seg.Close(nil)

    seg.AddAnnotation("aws_service", service)

    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    switch service {
    case "cloudwatch":
        return fetchCloudWatchMetrics(ctx, cfg, metricName, namespace)
    case "ec2":
        return fetchEC2Instances(ctx, cfg)
    default:
        err := fmt.Errorf("unsupported service: %s", service)
        seg.AddError(err)
        return nil, err
    }
}

func fetchCloudWatchMetrics(ctx context.Context, cfg aws.Config, metricName, namespace string) (*AWSMetric, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "fetch-cloudwatch-metrics")
    defer seg.Close(nil)

    seg.AddAnnotation("metric_name", metricName)
    seg.AddAnnotation("namespace", namespace)

    client := cloudwatch.NewFromConfig(cfg)
    
    endTime := time.Now()
    startTime := endTime.Add(-1 * time.Hour)

    input := &cloudwatch.GetMetricStatisticsInput{
        MetricName: aws.String(metricName),
        Namespace:  aws.String(namespace),
        StartTime:  aws.Time(startTime),
        EndTime:    aws.Time(endTime),
        Period:     aws.Int32(300),
        Statistics: []types.Statistic{types.StatisticAverage},
    }

    result, err := client.GetMetricStatistics(ctx, input)
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to get CloudWatch metrics: %w", err)
    }

    seg.AddAnnotation("datapoint_count", len(result.Datapoints))

    return &AWSMetric{
        MetricName: metricName,
        Namespace:  namespace,
        Datapoints: result.Datapoints,
    }, nil
}

func fetchEC2Instances(ctx context.Context, cfg aws.Config) (*AWSMetric, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "fetch-ec2-instances")
    defer seg.Close(nil)

    client := ec2.NewFromConfig(cfg)

    result, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
    if err != nil {
        seg.AddError(err)
        return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
    }

    var instances []ec2Types.Instance
    for _, reservation := range result.Reservations {
        instances = append(instances, reservation.Instances...)
    }

    seg.AddAnnotation("instance_count", len(instances))

    return &AWSMetric{
        MetricName: "EC2Instances",
        Namespace:  "AWS/EC2",
        Instances:  instances,
    }, nil
}

func storeAWSData(ctx context.Context, metric *AWSMetric) error {
    ctx, seg := xray.BeginSubsegment(ctx, "store-aws-data")
    defer seg.Close(nil)

    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        seg.AddError(err)
        return fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := dynamodb.NewFromConfig(cfg)
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        tableName = "aws-metrics"
    }

    seg.AddAnnotation("table_name", tableName)

    if len(metric.Datapoints) > 0 {
        seg.AddAnnotation("storage_type", "cloudwatch_data")
        return storeCloudWatchData(ctx, client, tableName, metric)
    }

    if len(metric.Instances) > 0 {
        seg.AddAnnotation("storage_type", "ec2_data")
        return storeEC2Data(ctx, client, tableName, metric)
    }

    return nil
}

func storeCloudWatchData(ctx context.Context, client *dynamodb.Client, tableName string, metric *AWSMetric) error {
    for _, point := range metric.Datapoints {
        item := map[string]dynamoTypes.AttributeValue{
            "metric_name": &dynamoTypes.AttributeValueMemberS{Value: metric.MetricName},
            "namespace":   &dynamoTypes.AttributeValueMemberS{Value: metric.Namespace},
            "timestamp":   &dynamoTypes.AttributeValueMemberS{Value: point.Timestamp.Format(time.RFC3339)},
            "value":       &dynamoTypes.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", *point.Average)},
            "scraped_at":  &dynamoTypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
        }

        _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
            TableName: aws.String(tableName),
            Item:      item,
        })

        if err != nil {
            return fmt.Errorf("failed to store CloudWatch data: %w", err)
        }
    }

    return nil
}

func storeEC2Data(ctx context.Context, client *dynamodb.Client, tableName string, metric *AWSMetric) error {
    for _, instance := range metric.Instances {
        item := map[string]dynamoTypes.AttributeValue{
            "instance_id":   &dynamoTypes.AttributeValueMemberS{Value: *instance.InstanceId},
            "instance_type": &dynamoTypes.AttributeValueMemberS{Value: string(instance.InstanceType)},
            "state":         &dynamoTypes.AttributeValueMemberS{Value: string(instance.State.Name)},
            "scraped_at":    &dynamoTypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
        }

        _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
            TableName: aws.String(tableName),
            Item:      item,
        })

        if err != nil {
            return fmt.Errorf("failed to store EC2 data: %w", err)
        }
    }

    return nil
}

func createSuccessResponse(message string) AWSResponse {
    return AWSResponse{
        Status:    "success",
        Message:   message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

func createErrorResponse(message string) AWSResponse {
    log.Printf("Error: %s", message)
    return AWSResponse{
        Status:    "error",
        Message:   message,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}