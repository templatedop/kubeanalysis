package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// K8sAnalyzer provides LLM-based Kubernetes analysis.
type K8sAnalyzer struct {
	client LLMClient
}

// NewK8sAnalyzer creates a new K8s analyzer.
func NewK8sAnalyzer(client LLMClient) *K8sAnalyzer {
	return &K8sAnalyzer{client: client}
}

// AnalyzeSecurity performs LLM-enhanced security analysis.
func (a *K8sAnalyzer) AnalyzeSecurity(ctx context.Context, findings, ragContext, clusterContext string) (*AnalysisResponse, error) {
	messages := BuildSecurityPrompt(findings, ragContext, clusterContext)

	resp, err := a.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return a.parseAnalysisResponse(resp.Content)
}

// AnalyzePerformance performs LLM-enhanced performance analysis.
func (a *K8sAnalyzer) AnalyzePerformance(ctx context.Context, findings, ragContext, clusterState, metrics string) (*AnalysisResponse, error) {
	messages := BuildPerformancePrompt(findings, ragContext, clusterState, metrics)

	resp, err := a.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return a.parseAnalysisResponse(resp.Content)
}

// AnalyzePredictions performs LLM-enhanced prediction analysis.
func (a *K8sAnalyzer) AnalyzePredictions(ctx context.Context, predictions, ragContext, historicalPatterns, currentState string) (*AnalysisResponse, error) {
	messages := BuildPredictionPrompt(predictions, ragContext, historicalPatterns, currentState)

	resp, err := a.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return a.parseAnalysisResponse(resp.Content)
}

// AnalyzeIncident performs LLM-enhanced incident analysis.
func (a *K8sAnalyzer) AnalyzeIncident(ctx context.Context, events, affectedResources, anomalies, logs, recentChanges, ragContext string) (*IncidentAnalysisResponse, error) {
	messages := BuildIncidentPrompt(events, affectedResources, anomalies, logs, recentChanges, ragContext)

	resp, err := a.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return a.parseIncidentResponse(resp.Content)
}

// TriageEvent performs quick event triage.
func (a *K8sAnalyzer) TriageEvent(ctx context.Context, eventType, reason, message, resource string, count int) (*TriageResponse, error) {
	messages := BuildTriagePrompt(eventType, reason, message, resource, count)

	resp, err := a.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM triage failed: %w", err)
	}

	return a.parseTriageResponse(resp.Content)
}

// GenerateSummary generates an executive summary from analysis results.
func (a *K8sAnalyzer) GenerateSummary(ctx context.Context, securityFindings, performanceFindings, predictions string) (string, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `You are a Kubernetes expert providing executive summaries for cluster health reports.
Be concise but comprehensive. Focus on actionable insights.`,
		},
		{
			Role: "user",
			Content: `Provide a brief executive summary for the following analysis:

SECURITY FINDINGS:
` + securityFindings + `

PERFORMANCE FINDINGS:
` + performanceFindings + `

PREDICTIONS:
` + predictions + `

Write a 2-3 paragraph executive summary covering:
1. Overall cluster health status
2. Most critical issues requiring attention
3. Key recommendations`,
		},
	}

	resp, err := a.client.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return resp.Content, nil
}

func (a *K8sAnalyzer) parseAnalysisResponse(content string) (*AnalysisResponse, error) {
	// Try to extract JSON from the response
	content = extractJSON(content)

	var resp AnalysisResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		// If JSON parsing fails, create a basic response
		return &AnalysisResponse{
			ExecutiveSummary: content,
			Confidence:       0.5,
		}, nil
	}

	return &resp, nil
}

func (a *K8sAnalyzer) parseIncidentResponse(content string) (*IncidentAnalysisResponse, error) {
	content = extractJSON(content)

	var resp IncidentAnalysisResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return &IncidentAnalysisResponse{
			Summary: content,
		}, nil
	}

	return &resp, nil
}

func (a *K8sAnalyzer) parseTriageResponse(content string) (*TriageResponse, error) {
	content = extractJSON(content)

	var resp TriageResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		// Default triage response
		return &TriageResponse{
			Severity:            "MEDIUM",
			Category:            "UNKNOWN",
			RequiresDeepAnalysis: true,
			SuggestedActions:    []string{"Review event details manually"},
		}, nil
	}

	return &resp, nil
}

// extractJSON attempts to extract JSON from a potentially markdown-wrapped response.
func extractJSON(content string) string {
	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)

	// Check for ```json ... ``` pattern
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	}

	// Find the first { and last }
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}

	return strings.TrimSpace(content)
}

// IncidentAnalysisResponse represents the response for incident analysis.
type IncidentAnalysisResponse struct {
	Summary            string       `json:"summary"`
	Severity           string       `json:"severity"`
	Timeline           []TimelineEvent `json:"timeline,omitempty"`
	RootCause          RootCause    `json:"root_cause,omitempty"`
	Impact             Impact       `json:"impact,omitempty"`
	Remediation        Remediation  `json:"remediation,omitempty"`
	SimilarIncidents   []SimilarIncident `json:"similar_incidents,omitempty"`
}

// TimelineEvent represents an event in the incident timeline.
type TimelineEvent struct {
	Timestamp    string `json:"timestamp"`
	Event        string `json:"event"`
	Significance string `json:"significance"`
}

// RootCause represents root cause analysis.
type RootCause struct {
	Primary             string   `json:"primary"`
	ContributingFactors []string `json:"contributing_factors,omitempty"`
	DefenseFailures     []string `json:"defense_failures,omitempty"`
}

// Impact represents the impact assessment.
type Impact struct {
	AffectedServices []string `json:"affected_services,omitempty"`
	UserImpact       string   `json:"user_impact,omitempty"`
	DataImpact       string   `json:"data_impact,omitempty"`
}

// Remediation represents remediation actions.
type Remediation struct {
	Immediate  []string `json:"immediate,omitempty"`
	Resolution []string `json:"resolution,omitempty"`
	Prevention []string `json:"prevention,omitempty"`
}

// SimilarIncident represents a similar past incident.
type SimilarIncident struct {
	ID               string `json:"id"`
	Similarity       string `json:"similarity"`
	LearningsApplied string `json:"learnings_applied,omitempty"`
}

// TriageResponse represents the response for event triage.
type TriageResponse struct {
	Severity             string   `json:"severity"`
	Category             string   `json:"category"`
	RequiresDeepAnalysis bool     `json:"requires_deep_analysis"`
	SuggestedActions     []string `json:"suggested_actions,omitempty"`
}
