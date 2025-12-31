package temporal

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// TaskQueueName is the default task queue for K8s analysis workflows.
	TaskQueueName = "k8s-analysis-queue"
)

// K8sAnalysisWorkflow orchestrates a complete Kubernetes cluster analysis.
func K8sAnalysisWorkflow(ctx workflow.Context, input AnalysisInput) (*AnalysisResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting K8s analysis workflow", "clusterID", input.ClusterID, "type", input.AnalysisType)

	result := &AnalysisResult{
		ClusterID:    input.ClusterID,
		AnalysisType: input.AnalysisType,
		StartTime:    workflow.Now(ctx),
		Status:       "running",
	}

	// Configure activity options with retry policy
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2.0,
			MaximumInterval:        time.Minute,
			MaximumAttempts:        3,
			NonRetryableErrorTypes: []string{"InvalidClusterError"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 1: Collect Cluster State
	var clusterState *ClusterState
	err := workflow.ExecuteActivity(ctx, "CollectClusterStateActivity", input).Get(ctx, &clusterState)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to collect cluster state: %v", err)
		return result, err
	}

	// Step 2: Run Analysis Pipelines in Parallel
	var securityReport *SecurityReport
	var performanceReport *PerformanceReport
	var predictionReport *PredictionReport

	// Create futures for parallel execution
	var securityFuture workflow.Future
	var performanceFuture workflow.Future
	var predictionFuture workflow.Future

	if input.AnalysisType == AnalysisTypeFull || input.AnalysisType == AnalysisTypeSecurity {
		securityFuture = workflow.ExecuteActivity(ctx, "SecurityAnalysisActivity", clusterState)
	}

	if input.AnalysisType == AnalysisTypeFull || input.AnalysisType == AnalysisTypePerformance {
		performanceFuture = workflow.ExecuteActivity(ctx, "PerformanceAnalysisActivity", clusterState)
	}

	if input.AnalysisType == AnalysisTypeFull || input.AnalysisType == AnalysisTypePrediction {
		predictionFuture = workflow.ExecuteActivity(ctx, "PredictionAnalysisActivity", clusterState)
	}

	// Wait for all analysis activities to complete
	if securityFuture != nil {
		if err := securityFuture.Get(ctx, &securityReport); err != nil {
			logger.Warn("Security analysis failed", "error", err)
		}
	}

	if performanceFuture != nil {
		if err := performanceFuture.Get(ctx, &performanceReport); err != nil {
			logger.Warn("Performance analysis failed", "error", err)
		}
	}

	if predictionFuture != nil {
		if err := predictionFuture.Get(ctx, &predictionReport); err != nil {
			logger.Warn("Prediction analysis failed", "error", err)
		}
	}

	// Step 3: Fetch RAG Context for LLM enhancement
	ragQuery := buildRAGQuery(securityReport, performanceReport, predictionReport)
	var ragContext interface{} // Using interface{} to avoid circular imports
	if ragQuery != "" {
		ragInput := RAGQueryInput{
			Query: ragQuery,
			TopK:  10,
		}
		err = workflow.ExecuteActivity(ctx, "FetchRAGContextActivity", ragInput).Get(ctx, &ragContext)
		if err != nil {
			logger.Warn("RAG context fetch failed", "error", err)
		}
	}

	// Step 4: LLM-Enhanced Analysis
	llmInput := LLMAnalysisInput{
		Security:    securityReport,
		Performance: performanceReport,
		Predictions: predictionReport,
	}

	var llmInsights *LLMInsights
	err = workflow.ExecuteActivity(ctx, "LLMEnhanceAnalysisActivity", llmInput).Get(ctx, &llmInsights)
	if err != nil {
		logger.Warn("LLM analysis failed", "error", err)
	}

	// Step 5: Compile results
	result.Security = securityReport
	result.Performance = performanceReport
	result.Predictions = predictionReport
	result.LLMInsights = llmInsights
	result.EndTime = workflow.Now(ctx)
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = "completed"

	// Aggregate findings
	result.Findings = aggregateFindings(securityReport, performanceReport, predictionReport)

	// Step 6: Send Alerts if Critical
	if hasCriticalFindings(result) {
		err = workflow.ExecuteActivity(ctx, "SendAlertsActivity", result).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to send alerts", "error", err)
		}
	}

	// Step 7: Store Results
	err = workflow.ExecuteActivity(ctx, "StoreAnalysisResultActivity", result).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to store results", "error", err)
	}

	return result, nil
}

// ContinuousMonitoringWorkflow runs continuous analysis at regular intervals.
func ContinuousMonitoringWorkflow(ctx workflow.Context, config MonitoringConfig) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting continuous monitoring", "clusterID", config.ClusterID, "interval", config.Interval)

	for {
		// Check for workflow cancellation
		if ctx.Err() != nil {
			logger.Info("Continuous monitoring cancelled")
			return ctx.Err()
		}

		// Run analysis workflow as a child workflow
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:               fmt.Sprintf("analysis-%s-%d", config.ClusterID, workflow.Now(ctx).Unix()),
			WorkflowExecutionTimeout: 15 * time.Minute,
		})

		input := AnalysisInput{
			ClusterID:    config.ClusterID,
			AnalysisType: config.AnalysisType,
		}

		var result *AnalysisResult
		err := workflow.ExecuteChildWorkflow(childCtx, K8sAnalysisWorkflow, input).Get(ctx, &result)
		if err != nil {
			logger.Warn("Analysis workflow failed", "error", err)
		} else {
			logger.Info("Analysis completed",
				"clusterID", config.ClusterID,
				"status", result.Status,
				"findings", len(result.Findings),
			)
		}

		// Wait for next interval
		if err := workflow.Sleep(ctx, config.Interval); err != nil {
			return err
		}
	}
}

// EventAnalysisWorkflow handles event-driven analysis with quick triage.
func EventAnalysisWorkflow(ctx workflow.Context, input EventAnalysisInput) (*AnalysisResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting event analysis", "eventID", input.Event.ID, "reason", input.Event.Reason)

	// Configure activity options for quick triage
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 1: Quick Triage
	var triage *TriageResult
	err := workflow.ExecuteActivity(ctx, "TriageEventActivity", input.Event).Get(ctx, &triage)
	if err != nil {
		return nil, fmt.Errorf("triage failed: %w", err)
	}

	result := &AnalysisResult{
		ClusterID:    input.Event.Resource.Namespace,
		AnalysisType: AnalysisTypeSecurity,
		StartTime:    workflow.Now(ctx),
		Status:       "completed",
		Findings: []Finding{{
			ID:          triage.EventID,
			Severity:    triage.Severity,
			Category:    triage.Category,
			Resource:    input.Event.Resource,
			Title:       input.Event.Reason,
			Description: triage.Summary,
			Timestamp:   workflow.Now(ctx),
		}},
	}

	// Step 2: If critical, run deep analysis
	if triage.RequiresDeepAnalysis && (triage.Severity == SeverityCritical || triage.Severity == SeverityHigh) {
		logger.Info("Running deep analysis for critical event", "eventID", input.Event.ID)

		// Configure longer timeout for deep analysis
		deepCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute,
			HeartbeatTimeout:    30 * time.Second,
		})

		// Collect related cluster state
		analysisInput := AnalysisInput{
			ClusterID:    input.Event.Resource.Namespace,
			AnalysisType: AnalysisTypeFull,
			Namespaces:   []string{input.Event.Resource.Namespace},
		}

		var clusterState *ClusterState
		err = workflow.ExecuteActivity(deepCtx, "CollectClusterStateActivity", analysisInput).Get(ctx, &clusterState)
		if err != nil {
			logger.Warn("Failed to collect cluster state for deep analysis", "error", err)
		} else {
			// Run security analysis on affected namespace
			var securityReport *SecurityReport
			err = workflow.ExecuteActivity(deepCtx, "SecurityAnalysisActivity", clusterState).Get(ctx, &securityReport)
			if err == nil {
				result.Security = securityReport
			}

			// Run performance analysis
			var performanceReport *PerformanceReport
			err = workflow.ExecuteActivity(deepCtx, "PerformanceAnalysisActivity", clusterState).Get(ctx, &performanceReport)
			if err == nil {
				result.Performance = performanceReport
			}
		}

		// Send alert for critical events
		err = workflow.ExecuteActivity(ctx, "SendAlertsActivity", result).Get(ctx, nil)
		if err != nil {
			logger.Warn("Failed to send alert", "error", err)
		}
	}

	result.EndTime = workflow.Now(ctx)
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// SecurityScanWorkflow performs focused security scanning.
func SecurityScanWorkflow(ctx workflow.Context, input AnalysisInput) (*SecurityReport, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting security scan", "clusterID", input.ClusterID)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Collect cluster state
	var clusterState *ClusterState
	err := workflow.ExecuteActivity(ctx, "CollectClusterStateActivity", input).Get(ctx, &clusterState)
	if err != nil {
		return nil, fmt.Errorf("failed to collect cluster state: %w", err)
	}

	// Run security analysis
	var securityReport *SecurityReport
	err = workflow.ExecuteActivity(ctx, "SecurityAnalysisActivity", clusterState).Get(ctx, &securityReport)
	if err != nil {
		return nil, fmt.Errorf("security analysis failed: %w", err)
	}

	// Get RAG context for security best practices
	ragInput := RAGQueryInput{
		Query: "kubernetes security best practices pod security network policies RBAC",
		TopK:  5,
	}
	var ragContext interface{}
	_ = workflow.ExecuteActivity(ctx, "FetchRAGContextActivity", ragInput).Get(ctx, &ragContext)

	// Enhance with LLM
	llmInput := LLMAnalysisInput{
		Security: securityReport,
	}
	var llmInsights *LLMInsights
	err = workflow.ExecuteActivity(ctx, "LLMEnhanceAnalysisActivity", llmInput).Get(ctx, &llmInsights)
	if err == nil && llmInsights != nil {
		securityReport.Recommendations = llmInsights.PrioritizedActions
	}

	return securityReport, nil
}

// PerformanceScanWorkflow performs focused performance scanning.
func PerformanceScanWorkflow(ctx workflow.Context, input AnalysisInput) (*PerformanceReport, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting performance scan", "clusterID", input.ClusterID)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Collect cluster state
	var clusterState *ClusterState
	err := workflow.ExecuteActivity(ctx, "CollectClusterStateActivity", input).Get(ctx, &clusterState)
	if err != nil {
		return nil, fmt.Errorf("failed to collect cluster state: %w", err)
	}

	// Run performance analysis
	var performanceReport *PerformanceReport
	err = workflow.ExecuteActivity(ctx, "PerformanceAnalysisActivity", clusterState).Get(ctx, &performanceReport)
	if err != nil {
		return nil, fmt.Errorf("performance analysis failed: %w", err)
	}

	return performanceReport, nil
}

// Helper functions

func buildRAGQuery(security *SecurityReport, performance *PerformanceReport, predictions *PredictionReport) string {
	var query string

	if security != nil && security.Summary.CriticalCount > 0 {
		query += "kubernetes security vulnerabilities privileged containers network policies "
	}

	if performance != nil && performance.Summary.BySeverity["CRITICAL"] > 0 {
		query += "kubernetes performance optimization resource limits OOM memory "
	}

	if predictions != nil && predictions.Summary.HighRiskCount > 0 {
		query += "kubernetes capacity planning failure prediction "
	}

	if query == "" {
		query = "kubernetes best practices"
	}

	return query
}

func hasCriticalFindings(result *AnalysisResult) bool {
	if result.Security != nil && result.Security.Summary.CriticalCount > 0 {
		return true
	}
	if result.Performance != nil && result.Performance.Summary.BySeverity["CRITICAL"] > 0 {
		return true
	}
	if result.Predictions != nil && result.Predictions.Summary.BySeverity["CRITICAL"] > 0 {
		return true
	}
	return false
}

func aggregateFindings(security *SecurityReport, performance *PerformanceReport, predictions *PredictionReport) []Finding {
	var findings []Finding

	if security != nil {
		for _, f := range security.Findings {
			findings = append(findings, f.Finding)
		}
	}

	if performance != nil {
		for _, f := range performance.Findings {
			findings = append(findings, f.Finding)
		}
	}

	if predictions != nil {
		for _, p := range predictions.Predictions {
			findings = append(findings, Finding{
				ID:          p.ID,
				Severity:    p.Severity,
				Category:    "PREDICTION",
				Resource:    p.Resource,
				Title:       p.Title,
				Description: p.Description,
				Remediation: p.Recommendation,
			})
		}
	}

	return findings
}
