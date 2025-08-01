package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type Event struct {
	Namespace string `json:"namespace,omitempty"`
	Cluster   string `json:"cluster,omitempty"`
}

type KubernetesResource struct {
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`
	Owner       string            `json:"owner,omitempty"`
	Team        string            `json:"team,omitempty"`
	Contact     string            `json:"contact,omitempty"`
}

type OwnershipData struct {
	Cluster   string               `json:"cluster"`
	Namespace string               `json:"namespace"`
	Resources []KubernetesResource `json:"resources"`
	Timestamp string               `json:"timestamp"`
	Source    string               `json:"source"`
	Confidence float64             `json:"confidence"`
}

func HandleRequest(ctx context.Context, event Event) (string, error) {
	ctx, seg := xray.BeginSegment(ctx, "openshift-scraper")
	defer seg.Close(nil)

	log.Printf("Processing OpenShift metadata for cluster: %s, namespace: %s", event.Cluster, event.Namespace)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	_ = cfg // Use config if needed for secrets or other AWS services

	ownershipData, err := scrapeOpenShiftMetadata(ctx, event.Cluster, event.Namespace)
	if err != nil {
		return "", fmt.Errorf("failed to scrape OpenShift metadata: %w", err)
	}

	result, _ := json.Marshal(ownershipData)
	return string(result), nil
}

func scrapeOpenShiftMetadata(ctx context.Context, cluster, namespace string) (*OwnershipData, error) {
	_, seg := xray.BeginSubsegment(ctx, "scrape-k8s-resources")
	defer seg.Close(nil)

	// Mock Kubernetes resources for POC - in production would use k8s client
	resources := []KubernetesResource{
		{
			Kind:      "Deployment",
			Name:      "api-service",
			Namespace: "production",
			Annotations: map[string]string{
				"owner":                    "backend-team",
				"contact":                  "backend-team@company.com",
				"app.kubernetes.io/owner":  "john.doe@company.com",
				"deployment.kubernetes.io/revision": "5",
			},
			Labels: map[string]string{
				"app":         "api-service",
				"team":        "backend",
				"environment": "production",
				"version":     "v1.2.3",
			},
		},
		{
			Kind:      "Service",
			Name:      "frontend-service",
			Namespace: "production",
			Annotations: map[string]string{
				"owner":   "frontend-team",
				"contact": "frontend-team@company.com",
				"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
			},
			Labels: map[string]string{
				"app":         "frontend",
				"team":        "frontend",
				"environment": "production",
				"tier":        "web",
			},
		},
		{
			Kind:      "ConfigMap",
			Name:      "database-config",
			Namespace: "production",
			Annotations: map[string]string{
				"owner":                "platform-team",
				"contact":              "platform@company.com",
				"managed-by":           "terraform",
				"last-updated":         "2024-01-15",
			},
			Labels: map[string]string{
				"app":         "database",
				"team":        "platform",
				"environment": "production",
				"component":   "config",
			},
		},
		{
			Kind:      "Secret",
			Name:      "api-secrets",
			Namespace: "production",
			Annotations: map[string]string{
				"owner":                    "security-team",
				"contact":                  "security@company.com",
				"kubernetes.io/managed-by": "external-secrets-operator",
			},
			Labels: map[string]string{
				"app":            "api-service",
				"team":           "security",
				"environment":    "production",
				"secret-type":    "application",
			},
		},
	}

	// Extract ownership information from annotations and labels
	for i := range resources {
		extractOwnershipInfo(&resources[i])
	}

	ownershipData := &OwnershipData{
		Cluster:   cluster,
		Namespace: namespace,
		Resources: resources,
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "openshift-metadata",
		Confidence: 0.9, // Very high confidence for infrastructure-level ownership
	}

	seg.AddAnnotation("resources_found", len(resources))
	seg.AddMetadata("cluster", cluster)
	seg.AddMetadata("namespace", namespace)

	return ownershipData, nil
}

func extractOwnershipInfo(resource *KubernetesResource) {
	// Check annotations for ownership
	if owner := resource.Annotations["owner"]; owner != "" {
		resource.Owner = owner
	} else if owner := resource.Annotations["app.kubernetes.io/owner"]; owner != "" {
		resource.Owner = owner
	}

	// Check labels for team information
	if team := resource.Labels["team"]; team != "" {
		resource.Team = team
	}

	// Check annotations for contact information
	if contact := resource.Annotations["contact"]; contact != "" {
		resource.Contact = contact
	}

	// Fallback to labels if no explicit owner in annotations
	if resource.Owner == "" {
		if app := resource.Labels["app"]; app != "" {
			resource.Owner = fmt.Sprintf("%s-team", app)
		}
	}

	// Ensure team is set
	if resource.Team == "" && resource.Owner != "" {
		resource.Team = resource.Owner
	}
}

func main() {
	lambda.Start(HandleRequest)
}