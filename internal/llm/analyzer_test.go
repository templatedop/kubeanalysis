package llm

import (
	"context"
	"encoding/json"
	"testing"
)

func TestK8sAnalyzerAnalyzeSecurity(t *testing.T) {
	mockResponse := `{
		"executive_summary": "Critical security issues found",
		"risk_level": "HIGH",
		"key_risks": ["Privileged containers", "No network policies"],
		"confidence": 0.9
	}`

	client := NewMockClient(mockResponse)
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	resp, err := analyzer.AnalyzeSecurity(ctx, "findings", "rag context", "cluster context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ExecutiveSummary != "Critical security issues found" {
		t.Errorf("unexpected executive summary: %s", resp.ExecutiveSummary)
	}
	if resp.RiskLevel != "HIGH" {
		t.Errorf("unexpected risk level: %s", resp.RiskLevel)
	}
	if len(resp.KeyRisks) != 2 {
		t.Errorf("expected 2 key risks, got %d", len(resp.KeyRisks))
	}
}

func TestK8sAnalyzerAnalyzePerformance(t *testing.T) {
	mockResponse := `{
		"executive_summary": "Performance optimization needed",
		"risk_level": "MEDIUM",
		"confidence": 0.85
	}`

	client := NewMockClient(mockResponse)
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	resp, err := analyzer.AnalyzePerformance(ctx, "findings", "rag", "state", "metrics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ExecutiveSummary != "Performance optimization needed" {
		t.Errorf("unexpected summary: %s", resp.ExecutiveSummary)
	}
}

func TestK8sAnalyzerAnalyzePredictions(t *testing.T) {
	mockResponse := `{
		"executive_summary": "Capacity issues predicted",
		"risk_level": "HIGH",
		"confidence": 0.75
	}`

	client := NewMockClient(mockResponse)
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	resp, err := analyzer.AnalyzePredictions(ctx, "predictions", "rag", "patterns", "state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestK8sAnalyzerAnalyzeIncident(t *testing.T) {
	mockResponse := `{
		"summary": "OOM incident analyzed",
		"severity": "CRITICAL",
		"root_cause": {
			"primary": "Memory leak in application",
			"contributing_factors": ["Insufficient limits", "No HPA"]
		}
	}`

	client := NewMockClient(mockResponse)
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	resp, err := analyzer.AnalyzeIncident(ctx, "events", "resources", "anomalies", "logs", "changes", "rag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Severity != "CRITICAL" {
		t.Errorf("unexpected severity: %s", resp.Severity)
	}
}

func TestK8sAnalyzerTriageEvent(t *testing.T) {
	mockResponse := `{
		"severity": "HIGH",
		"category": "HEALTH",
		"requires_deep_analysis": true,
		"suggested_actions": ["Check logs", "Restart pod"]
	}`

	client := NewMockClient(mockResponse)
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	resp, err := analyzer.TriageEvent(ctx, "Warning", "CrashLoopBackOff", "Container crashed", "default/pod", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Severity != "HIGH" {
		t.Errorf("unexpected severity: %s", resp.Severity)
	}
	if resp.Category != "HEALTH" {
		t.Errorf("unexpected category: %s", resp.Category)
	}
	if !resp.RequiresDeepAnalysis {
		t.Error("expected requires_deep_analysis to be true")
	}
}

func TestK8sAnalyzerGenerateSummary(t *testing.T) {
	client := NewMockClient("This is the executive summary of the cluster.")
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	summary, err := analyzer.GenerateSummary(ctx, "security findings", "perf findings", "predictions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestK8sAnalyzerParseInvalidJSON(t *testing.T) {
	// Test with invalid JSON - should return raw content
	client := NewMockClient("This is not valid JSON")
	analyzer := NewK8sAnalyzer(client)
	ctx := context.Background()

	resp, err := analyzer.AnalyzeSecurity(ctx, "findings", "rag", "context")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still return a response with raw content as summary
	if resp == nil {
		t.Fatal("expected non-nil response even for invalid JSON")
	}
	if resp.ExecutiveSummary == "" {
		t.Error("expected executive summary to contain raw content")
	}
}

func TestBuildSecurityPrompt(t *testing.T) {
	messages := BuildSecurityPrompt("findings", "rag context", "cluster context")

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Errorf("first message should be system, got %s", messages[0].Role)
	}
	if messages[1].Role != "user" {
		t.Errorf("second message should be user, got %s", messages[1].Role)
	}
}

func TestBuildPerformancePrompt(t *testing.T) {
	messages := BuildPerformancePrompt("findings", "rag", "state", "metrics")

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestBuildPredictionPrompt(t *testing.T) {
	messages := BuildPredictionPrompt("predictions", "rag", "patterns", "state")

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestBuildIncidentPrompt(t *testing.T) {
	messages := BuildIncidentPrompt("events", "resources", "anomalies", "logs", "changes", "rag")

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestBuildTriagePrompt(t *testing.T) {
	messages := BuildTriagePrompt("Warning", "CrashLoopBackOff", "Container crashed", "default/pod", 5)

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestAnalysisResponseJSONParsing(t *testing.T) {
	response := AnalysisResponse{
		ExecutiveSummary: "Test summary",
		RiskLevel:        "HIGH",
		KeyRisks:         []string{"Risk 1", "Risk 2"},
		PrioritizedFindings: []PrioritizedFinding{
			{
				Rank:           1,
				FindingID:      "finding-1",
				Exploitability: "HIGH",
				Impact:         "Critical",
			},
		},
		RemediationRoadmap: &RemediationRoadmap{
			Immediate: []RemediationAction{
				{Action: "Fix issue", Effort: "LOW"},
			},
		},
		Compliance: &ComplianceAnalysis{
			NSACISA: &FrameworkStatus{
				Passed: 10,
				Failed: 2,
				Total:  12,
				Score:  83.3,
			},
		},
		Confidence: 0.9,
	}

	// Marshal and unmarshal to verify structure
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed AnalysisResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ExecutiveSummary != response.ExecutiveSummary {
		t.Error("summary mismatch after JSON round-trip")
	}
	if len(parsed.KeyRisks) != 2 {
		t.Error("key risks mismatch after JSON round-trip")
	}
}

func TestIncidentAnalysisResponse(t *testing.T) {
	response := IncidentAnalysisResponse{
		Summary:  "Incident summary",
		Severity: "CRITICAL",
		Timeline: []TimelineEvent{
			{Timestamp: "12:00", Event: "First event", Significance: "Trigger"},
		},
		RootCause: RootCause{
			Primary:             "Memory leak",
			ContributingFactors: []string{"No limits"},
			DefenseFailures:     []string{"No HPA"},
		},
		Impact: Impact{
			AffectedServices: []string{"service-a"},
			UserImpact:       "High latency",
		},
		Remediation: Remediation{
			Immediate:  []string{"Restart pods"},
			Resolution: []string{"Fix memory leak"},
			Prevention: []string{"Add HPA"},
		},
		SimilarIncidents: []SimilarIncident{
			{ID: "inc-1", Similarity: "Similar memory issue"},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed IncidentAnalysisResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Severity != "CRITICAL" {
		t.Error("severity mismatch")
	}
}
