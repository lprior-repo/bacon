package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "os"
    "strings"
    "time"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/sns"
    "github.com/aws/aws-xray-sdk-go/xray"
)

type ProcessorEvent struct {
    ScraperOutputs []ScraperOutput `json:"scraper_outputs"`
}

type ScraperOutput struct {
    Source     string                 `json:"source"`
    Data       map[string]interface{} `json:"data"`
    Confidence float64                `json:"confidence"`
    Timestamp  string                 `json:"timestamp"`
}

type ProcessorResponse struct {
    Status           string        `json:"status"`
    Message          string        `json:"message"`
    ProcessedAt      string        `json:"processed_at"`
    RelationshipCount int          `json:"relationship_count"`
    ConflictCount    int           `json:"conflict_count"`
}

type Relationship struct {
    From       string  `json:"from"`
    To         string  `json:"to"`
    Type       string  `json:"type"`
    Confidence float64 `json:"confidence"`
    Source     string  `json:"source"`
    HasConflict bool   `json:"has_conflict"`
    Timestamp  string  `json:"timestamp"`
}

type ConfidenceEngine struct {
    SourceWeights    map[string]float64
    AgreementBonus   float64
    FreshnessDecay   float64
}

type ConflictDetector struct {
    ConflictThreshold float64
    SourcePriority    map[string]int
}

func main() {
    lambda.Start(handleProcessorRequest)
}

func handleProcessorRequest(ctx context.Context, event ProcessorEvent) (ProcessorResponse, error) {
    ctx, seg := xray.BeginSubsegment(ctx, "processor-handler")
    defer seg.Close(nil)

    seg.AddAnnotation("scraper_count", len(event.ScraperOutputs))

    // Initialize confidence engine and conflict detector
    confEngine := initConfidenceEngine()
    conflictDet := initConflictDetector()

    // Extract relationships from scraper outputs
    relationships := extractRelationships(ctx, event.ScraperOutputs)
    
    // Apply confidence scoring
    scoredRelationships := applyConfidenceScoring(ctx, relationships, confEngine)
    
    // Detect conflicts
    resolvedRelationships := detectAndResolveConflicts(ctx, scoredRelationships, conflictDet)
    
    // Store in Neptune
    err := storeInNeptune(ctx, resolvedRelationships)
    if err != nil {
        seg.AddError(err)
        return createErrorResponse(fmt.Sprintf("failed to store in Neptune: %v", err), 0, 0), err
    }

    conflictCount := countConflicts(resolvedRelationships)
    
    seg.AddMetadata("processing_result", map[string]interface{}{
        "relationship_count": len(resolvedRelationships),
        "conflict_count":     conflictCount,
        "scraper_sources":    getSourceNames(event.ScraperOutputs),
    })

    return createSuccessResponse(len(resolvedRelationships), conflictCount), nil
}

func initConfidenceEngine() *ConfidenceEngine {
    return &ConfidenceEngine{
        SourceWeights: map[string]float64{
            "openshift-metadata": 0.9,
            "aws-tags":          0.9,
            "github-codeowners": 0.8,
            "github-activity":   0.6,
            "datadog-metrics":   0.5,
        },
        AgreementBonus: 0.1,
        FreshnessDecay: 0.05,
    }
}

func initConflictDetector() *ConflictDetector {
    return &ConflictDetector{
        ConflictThreshold: 0.3,
        SourcePriority: map[string]int{
            "aws-tags":          1,
            "openshift-metadata": 2,
            "github-codeowners": 3,
            "github-activity":   4,
            "datadog-metrics":   5,
        },
    }
}

func extractRelationships(ctx context.Context, outputs []ScraperOutput) []Relationship {
    ctx, seg := xray.BeginSubsegment(ctx, "extract-relationships")
    defer seg.Close(nil)

    var relationships []Relationship

    for _, output := range outputs {
        switch output.Source {
        case "github-codeowners":
            relationships = append(relationships, extractCodeownersRelationships(output)...)
        case "openshift-metadata":
            relationships = append(relationships, extractOpenShiftRelationships(output)...)
        case "aws-tags":
            relationships = append(relationships, extractAWSRelationships(output)...)
        }
    }

    seg.AddAnnotation("relationship_count", len(relationships))
    return relationships
}

func extractCodeownersRelationships(output ScraperOutput) []Relationship {
    var relationships []Relationship
    
    // Extract from CODEOWNERS data structure
    if entries, ok := output.Data["entries"].([]interface{}); ok {
        for _, entry := range entries {
            if entryMap, ok := entry.(map[string]interface{}); ok {
                path := entryMap["path"].(string)
                if owners, ok := entryMap["owners"].([]interface{}); ok {
                    for _, owner := range owners {
                        rel := Relationship{
                            From:       strings.TrimPrefix(owner.(string), "@"),
                            To:         path,
                            Type:       "owns",
                            Confidence: output.Confidence,
                            Source:     output.Source,
                            Timestamp:  output.Timestamp,
                        }
                        relationships = append(relationships, rel)
                    }
                }
            }
        }
    }
    
    return relationships
}

func extractOpenShiftRelationships(output ScraperOutput) []Relationship {
    var relationships []Relationship
    
    if resources, ok := output.Data["resources"].([]interface{}); ok {
        for _, resource := range resources {
            if resMap, ok := resource.(map[string]interface{}); ok {
                resourceName := fmt.Sprintf("%s/%s", resMap["kind"], resMap["name"])
                if owner, ok := resMap["owner"].(string); ok && owner != "" {
                    rel := Relationship{
                        From:       owner,
                        To:         resourceName,
                        Type:       "owns",
                        Confidence: output.Confidence,
                        Source:     output.Source,
                        Timestamp:  output.Timestamp,
                    }
                    relationships = append(relationships, rel)
                }
            }
        }
    }
    
    return relationships
}

func extractAWSRelationships(output ScraperOutput) []Relationship {
    var relationships []Relationship
    
    // Mock AWS relationship extraction
    if resources, ok := output.Data["resources"].([]interface{}); ok {
        for _, resource := range resources {
            if resMap, ok := resource.(map[string]interface{}); ok {
                if resourceArn, ok := resMap["arn"].(string); ok {
                    if tags, ok := resMap["tags"].(map[string]interface{}); ok {
                        if owner, ok := tags["Owner"].(string); ok {
                            rel := Relationship{
                                From:       owner,
                                To:         resourceArn,
                                Type:       "owns",
                                Confidence: output.Confidence,
                                Source:     output.Source,
                                Timestamp:  output.Timestamp,
                            }
                            relationships = append(relationships, rel)
                        }
                    }
                }
            }
        }
    }
    
    return relationships
}

func applyConfidenceScoring(ctx context.Context, relationships []Relationship, engine *ConfidenceEngine) []Relationship {
    ctx, seg := xray.BeginSubsegment(ctx, "apply-confidence-scoring")
    defer seg.Close(nil)

    // Group relationships by target for multi-source analysis
    relationshipGroups := make(map[string][]Relationship)
    for _, rel := range relationships {
        key := fmt.Sprintf("%s->%s", rel.From, rel.To)
        relationshipGroups[key] = append(relationshipGroups[key], rel)
    }

    var scoredRelationships []Relationship
    for _, group := range relationshipGroups {
        if len(group) == 1 {
            // Single source - apply base confidence with source weight
            rel := group[0]
            rel.Confidence = calculateSingleSourceConfidence(rel, engine)
            scoredRelationships = append(scoredRelationships, rel)
        } else {
            // Multi-source - apply agreement bonus
            scored := calculateMultiSourceConfidence(group, engine)
            scoredRelationships = append(scoredRelationships, scored...)
        }
    }

    seg.AddAnnotation("scored_relationships", len(scoredRelationships))
    return scoredRelationships
}

func calculateSingleSourceConfidence(rel Relationship, engine *ConfidenceEngine) float64 {
    sourceWeight := engine.SourceWeights[rel.Source]
    if sourceWeight == 0 {
        sourceWeight = 0.5 // Default for unknown sources
    }
    
    // Apply freshness decay
    freshnessMultiplier := calculateFreshnessMultiplier(rel.Timestamp, engine.FreshnessDecay)
    
    confidence := rel.Confidence * sourceWeight * freshnessMultiplier
    return math.Min(confidence, 1.0)
}

func calculateMultiSourceConfidence(group []Relationship, engine *ConfidenceEngine) []Relationship {
    // Find the relationship with highest base confidence
    maxConfidence := 0.0
    bestRel := group[0]
    
    for _, rel := range group {
        baseConf := calculateSingleSourceConfidence(rel, engine)
        if baseConf > maxConfidence {
            maxConfidence = baseConf
            bestRel = rel
        }
    }
    
    // Apply multi-source agreement bonus
    bestRel.Confidence = math.Min(maxConfidence + engine.AgreementBonus, 1.0)
    
    return []Relationship{bestRel}
}

func calculateFreshnessMultiplier(timestamp string, decayRate float64) float64 {
    parsedTime, err := time.Parse(time.RFC3339, timestamp)
    if err != nil {
        return 1.0 // Default if can't parse
    }
    
    age := time.Since(parsedTime)
    daysSinceUpdate := age.Hours() / 24
    
    // Exponential decay over 30 days
    return math.Exp(-daysSinceUpdate * decayRate / 30)
}

func detectAndResolveConflicts(ctx context.Context, relationships []Relationship, detector *ConflictDetector) []Relationship {
    ctx, seg := xray.BeginSubsegment(ctx, "detect-resolve-conflicts")
    defer seg.Close(nil)

    // Group relationships by same target resource
    resourceGroups := make(map[string][]Relationship)
    for _, rel := range relationships {
        resourceGroups[rel.To] = append(resourceGroups[rel.To], rel)
    }

    var resolvedRelationships []Relationship
    conflictCount := 0

    for _, group := range resourceGroups {
        if len(group) == 1 {
            resolvedRelationships = append(resolvedRelationships, group[0])
            continue
        }

        // Check for conflicts (different owners for same resource)
        owners := make(map[string]bool)
        for _, rel := range group {
            owners[rel.From] = true
        }

        if len(owners) > 1 {
            // Conflict detected - resolve using priority
            resolved := resolveConflict(group, detector)
            resolvedRelationships = append(resolvedRelationships, resolved)
            conflictCount++
        } else {
            // Same owner from multiple sources - take highest confidence
            best := group[0]
            for _, rel := range group[1:] {
                if rel.Confidence > best.Confidence {
                    best = rel
                }
            }
            resolvedRelationships = append(resolvedRelationships, best)
        }
    }

    seg.AddAnnotation("conflict_count", conflictCount)
    return resolvedRelationships
}

func resolveConflict(conflicted []Relationship, detector *ConflictDetector) Relationship {
    // Find relationship with highest priority source
    bestPriority := 999
    bestRel := conflicted[0]

    for _, rel := range conflicted {
        priority := detector.SourcePriority[rel.Source]
        if priority == 0 {
            priority = 999 // Unknown sources get lowest priority
        }
        
        if priority < bestPriority {
            bestPriority = priority
            bestRel = rel
        }
    }

    // Mark as having conflict
    bestRel.HasConflict = true
    return bestRel
}

func storeInNeptune(ctx context.Context, relationships []Relationship) error {
    ctx, seg := xray.BeginSubsegment(ctx, "store-in-neptune")
    defer seg.Close(nil)

    // Mock Neptune storage for POC - in production would use Gremlin client
    // This would create vertices and edges in Neptune graph database
    
    log.Printf("Storing %d relationships in Neptune", len(relationships))
    
    for _, rel := range relationships {
        // Mock Gremlin query construction
        gremlinQuery := fmt.Sprintf(
            "g.V().has('name', '%s').fold().coalesce(unfold(), addV('User').property('name', '%s')).as('from')."+
            "V().has('name', '%s').fold().coalesce(unfold(), addV('Resource').property('name', '%s')).as('to')."+
            "addE('%s').from('from').to('to')."+
            "property('confidence', %f)."+
            "property('source', '%s')."+
            "property('has_conflict', %t)."+
            "property('timestamp', '%s')",
            rel.From, rel.From,
            rel.To, rel.To,
            rel.Type,
            rel.Confidence,
            rel.Source,
            rel.HasConflict,
            rel.Timestamp,
        )
        
        log.Printf("Would execute Gremlin query: %s", gremlinQuery)
    }

    seg.AddAnnotation("relationships_stored", len(relationships))
    return nil
}

func countConflicts(relationships []Relationship) int {
    count := 0
    for _, rel := range relationships {
        if rel.HasConflict {
            count++
        }
    }
    return count
}

func getSourceNames(outputs []ScraperOutput) []string {
    var sources []string
    for _, output := range outputs {
        sources = append(sources, output.Source)
    }
    return sources
}

func createSuccessResponse(relationshipCount, conflictCount int) ProcessorResponse {
    return ProcessorResponse{
        Status:           "success",
        Message:          "Relationship processing completed successfully",
        ProcessedAt:      time.Now().UTC().Format(time.RFC3339),
        RelationshipCount: relationshipCount,
        ConflictCount:    conflictCount,
    }
}

func createErrorResponse(message string, relationshipCount, conflictCount int) ProcessorResponse {
    log.Printf("Error: %s", message)
    return ProcessorResponse{
        Status:           "error",
        Message:          message,
        ProcessedAt:      time.Now().UTC().Format(time.RFC3339),
        RelationshipCount: relationshipCount,
        ConflictCount:    conflictCount,
    }
}