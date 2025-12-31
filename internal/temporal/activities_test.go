package temporal

import (
	"context"
	"testing"
	"time"
)

func TestCollectClusterStateActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	input := AnalysisInput{
		ClusterID:    "test-cluster",
		AnalysisType: AnalysisTypeFull,
		Namespaces:   []string{"default", "kube-system"},
	}

	state, err := activities.CollectClusterStateActivity(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.ClusterID != "test-cluster" {
		t.Errorf("expected cluster ID 'test-cluster', got %s", state.ClusterID)
	}
	if state.CollectedAt.IsZero() {
		t.Error("expected non-zero collected time")
	}
}

func TestSecurityAnalysisActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	state := &ClusterState{
		ClusterID:   "test-cluster",
		CollectedAt: time.Now(),
		Pods: []PodInfo{
			{
				Name:       "privileged-pod",
				Namespace:  "default",
				Status:     "Running",
				Privileged: true,
				RunAsRoot:  true,
			},
			{
				Name:             "insecure-pod",
				Namespace:        "default",
				Status:           "Running",
				HasNetworkPolicy: false,
			},
		},
	}

	report, err := activities.SecurityAnalysisActivity(ctx, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Should find security issues
	if len(report.Findings) == 0 {
		t.Error("expected security findings for insecure pods")
	}

	// Check for privileged container finding
	foundPrivileged := false
	for _, f := range report.Findings {
		if f.Title == "Privileged container detected" {
			foundPrivileged = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity for privileged container, got %s", f.Severity)
			}
		}
	}
	if !foundPrivileged {
		t.Error("expected to find privileged container issue")
	}

	// Verify summary
	if report.Summary.TotalFindings != len(report.Findings) {
		t.Errorf("summary total doesn't match findings count")
	}
}

func TestPerformanceAnalysisActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	state := &ClusterState{
		ClusterID:   "test-cluster",
		CollectedAt: time.Now(),
		Nodes: []NodeInfo{
			{
				Name:           "high-usage-node",
				Status:         "Ready",
				CPUCapacity:    4000,
				MemoryCapacity: 16 * 1024 * 1024 * 1024,
				CPUUsage:       3600, // 90% usage
				MemoryUsage:    14 * 1024 * 1024 * 1024, // ~87% usage
			},
		},
		Pods: []PodInfo{
			{
				Name:         "crashing-pod",
				Namespace:    "default",
				Status:       "CrashLoopBackOff",
				RestartCount: 10,
			},
		},
	}

	report, err := activities.PerformanceAnalysisActivity(ctx, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Should find performance issues
	if len(report.Findings) == 0 {
		t.Error("expected performance findings")
	}

	// Check for restart loop finding
	foundRestartLoop := false
	for _, f := range report.Findings {
		if f.Title == "Pod restart loop detected" {
			foundRestartLoop = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity for restart loop, got %s", f.Severity)
			}
		}
	}
	if !foundRestartLoop {
		t.Error("expected to find restart loop issue")
	}

	// Verify summary
	if report.Summary.TotalFindings != len(report.Findings) {
		t.Error("summary total doesn't match findings count")
	}
}

func TestPredictionAnalysisActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	state := &ClusterState{
		ClusterID:   "test-cluster",
		CollectedAt: time.Now(),
		Nodes: []NodeInfo{
			{
				Name:           "high-usage-node",
				Status:         "Ready",
				CPUCapacity:    4000,
				MemoryCapacity: 16 * 1024 * 1024 * 1024,
				CPUUsage:       3200, // 80% usage
				MemoryUsage:    12 * 1024 * 1024 * 1024, // 75% usage
			},
		},
		Pods: []PodInfo{
			{
				Name:         "unstable-pod",
				Namespace:    "default",
				RestartCount: 5,
			},
		},
	}

	report, err := activities.PredictionAnalysisActivity(ctx, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Should generate predictions
	if len(report.Predictions) == 0 {
		t.Error("expected predictions for high usage nodes")
	}

	// Verify predictions have required fields
	for _, p := range report.Predictions {
		if p.ID == "" {
			t.Error("prediction should have ID")
		}
		if p.Type == "" {
			t.Error("prediction should have type")
		}
		if p.Confidence <= 0 || p.Confidence > 1 {
			t.Errorf("prediction confidence should be between 0 and 1, got %f", p.Confidence)
		}
	}

	// Verify summary
	if report.Summary.TotalPredictions != len(report.Predictions) {
		t.Error("summary total doesn't match predictions count")
	}
	if len(report.Predictions) > 0 && report.Summary.AverageConfidence == 0 {
		t.Error("expected non-zero average confidence")
	}
}

func TestTriageEventActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	tests := []struct {
		name             string
		event            K8sEvent
		expectedSeverity Severity
		expectedCategory string
		requiresDeep     bool
	}{
		{
			name: "OOMKilled",
			event: K8sEvent{
				ID:      "event-1",
				Type:    "Warning",
				Reason:  "OOMKilled",
				Message: "Container killed due to OOM",
			},
			expectedSeverity: SeverityCritical,
			expectedCategory: "RESOURCE",
			requiresDeep:     true,
		},
		{
			name: "CrashLoopBackOff",
			event: K8sEvent{
				ID:      "event-2",
				Type:    "Warning",
				Reason:  "CrashLoopBackOff",
				Message: "Container crashing",
			},
			expectedSeverity: SeverityCritical,
			expectedCategory: "HEALTH",
			requiresDeep:     true,
		},
		{
			name: "FailedScheduling",
			event: K8sEvent{
				ID:      "event-3",
				Type:    "Warning",
				Reason:  "FailedScheduling",
				Message: "Cannot schedule pod",
			},
			expectedSeverity: SeverityHigh,
			expectedCategory: "SCHEDULING",
			requiresDeep:     true,
		},
		{
			name: "Unhealthy",
			event: K8sEvent{
				ID:      "event-4",
				Type:    "Warning",
				Reason:  "Unhealthy",
				Message: "Readiness probe failed",
			},
			expectedSeverity: SeverityHigh,
			expectedCategory: "HEALTH",
			requiresDeep:     false,
		},
		{
			name: "Unknown",
			event: K8sEvent{
				ID:      "event-5",
				Type:    "Normal",
				Reason:  "SomeReason",
				Message: "Some message",
			},
			expectedSeverity: SeverityInfo,
			expectedCategory: "GENERAL",
			requiresDeep:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := activities.TriageEventActivity(ctx, tt.event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.EventID != tt.event.ID {
				t.Errorf("expected event ID %s, got %s", tt.event.ID, result.EventID)
			}
			if result.Severity != tt.expectedSeverity {
				t.Errorf("expected severity %s, got %s", tt.expectedSeverity, result.Severity)
			}
			if result.Category != tt.expectedCategory {
				t.Errorf("expected category %s, got %s", tt.expectedCategory, result.Category)
			}
			if result.RequiresDeepAnalysis != tt.requiresDeep {
				t.Errorf("expected requires deep analysis %v, got %v", tt.requiresDeep, result.RequiresDeepAnalysis)
			}
		})
	}
}

func TestLLMEnhanceAnalysisActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		input         LLMAnalysisInput
		expectSummary bool
	}{
		{
			name: "critical findings",
			input: LLMAnalysisInput{
				Security: &SecurityReport{
					Summary: SecuritySummary{CriticalCount: 3, HighCount: 5},
				},
			},
			expectSummary: true,
		},
		{
			name: "no critical findings",
			input: LLMAnalysisInput{
				Security: &SecurityReport{
					Summary: SecuritySummary{CriticalCount: 0, HighCount: 0},
				},
			},
			expectSummary: true,
		},
		{
			name:          "empty input",
			input:         LLMAnalysisInput{},
			expectSummary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insights, err := activities.LLMEnhanceAnalysisActivity(ctx, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if insights == nil {
				t.Fatal("expected non-nil insights")
			}

			if tt.expectSummary && insights.ExecutiveSummary == "" {
				t.Error("expected non-empty executive summary")
			}

			if insights.ConfidenceLevel <= 0 || insights.ConfidenceLevel > 1 {
				t.Errorf("expected confidence between 0 and 1, got %f", insights.ConfidenceLevel)
			}
		})
	}
}

func TestSendAlertsActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	result := &AnalysisResult{
		ClusterID: "test-cluster",
		Security: &SecurityReport{
			Summary: SecuritySummary{CriticalCount: 1},
		},
	}

	err := activities.SendAlertsActivity(ctx, result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStoreAnalysisResultActivity(t *testing.T) {
	activities := NewActivities(nil)
	ctx := context.Background()

	result := &AnalysisResult{
		ClusterID: "test-cluster",
		Status:    "completed",
	}

	err := activities.StoreAnalysisResultActivity(ctx, result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
