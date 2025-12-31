package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/kubeanalysis/kubeanalysis/internal/rag"
)

// WorkerConfig holds configuration for the Temporal worker.
type WorkerConfig struct {
	// TemporalHost is the Temporal server host.
	TemporalHost string `json:"temporal_host"`

	// Namespace is the Temporal namespace.
	Namespace string `json:"namespace"`

	// TaskQueue is the task queue name.
	TaskQueue string `json:"task_queue"`

	// MaxConcurrentActivities limits concurrent activity executions.
	MaxConcurrentActivities int `json:"max_concurrent_activities"`

	// MaxConcurrentWorkflows limits concurrent workflow executions.
	MaxConcurrentWorkflows int `json:"max_concurrent_workflows"`
}

// DefaultWorkerConfig returns a default worker configuration.
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		TemporalHost:            "localhost:7233",
		Namespace:               "default",
		TaskQueue:               TaskQueueName,
		MaxConcurrentActivities: 10,
		MaxConcurrentWorkflows:  10,
	}
}

// Worker manages the Temporal worker lifecycle.
type Worker struct {
	config     WorkerConfig
	client     client.Client
	worker     worker.Worker
	activities *Activities
	ragEngine  *rag.Engine
}

// NewWorker creates a new Temporal worker.
func NewWorker(config WorkerConfig, ragEngine *rag.Engine) (*Worker, error) {
	return &Worker{
		config:     config,
		ragEngine:  ragEngine,
		activities: NewActivities(ragEngine),
	}, nil
}

// Start starts the Temporal worker.
func (w *Worker) Start(ctx context.Context) error {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  w.config.TemporalHost,
		Namespace: w.config.Namespace,
	})
	if err != nil {
		return fmt.Errorf("failed to create Temporal client: %w", err)
	}
	w.client = c

	// Create worker
	workerOpts := worker.Options{
		MaxConcurrentActivityExecutionSize:     w.config.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: w.config.MaxConcurrentWorkflows,
	}

	w.worker = worker.New(c, w.config.TaskQueue, workerOpts)

	// Register workflows
	w.worker.RegisterWorkflow(K8sAnalysisWorkflow)
	w.worker.RegisterWorkflow(ContinuousMonitoringWorkflow)
	w.worker.RegisterWorkflow(EventAnalysisWorkflow)
	w.worker.RegisterWorkflow(SecurityScanWorkflow)
	w.worker.RegisterWorkflow(PerformanceScanWorkflow)

	// Register activities
	w.worker.RegisterActivity(w.activities.CollectClusterStateActivity)
	w.worker.RegisterActivity(w.activities.SecurityAnalysisActivity)
	w.worker.RegisterActivity(w.activities.PerformanceAnalysisActivity)
	w.worker.RegisterActivity(w.activities.PredictionAnalysisActivity)
	w.worker.RegisterActivity(w.activities.FetchRAGContextActivity)
	w.worker.RegisterActivity(w.activities.LLMEnhanceAnalysisActivity)
	w.worker.RegisterActivity(w.activities.SendAlertsActivity)
	w.worker.RegisterActivity(w.activities.StoreAnalysisResultActivity)
	w.worker.RegisterActivity(w.activities.TriageEventActivity)

	// Start worker
	return w.worker.Run(worker.InterruptCh())
}

// Stop stops the Temporal worker.
func (w *Worker) Stop() {
	if w.worker != nil {
		w.worker.Stop()
	}
	if w.client != nil {
		w.client.Close()
	}
}

// GetClient returns the Temporal client.
func (w *Worker) GetClient() client.Client {
	return w.client
}

// Client provides a client interface for starting workflows.
type Client struct {
	client    client.Client
	taskQueue string
}

// NewClient creates a new workflow client.
func NewClient(config WorkerConfig) (*Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  config.TemporalHost,
		Namespace: config.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return &Client{
		client:    c,
		taskQueue: config.TaskQueue,
	}, nil
}

// Close closes the client connection.
func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// StartAnalysis starts a K8s analysis workflow.
func (c *Client) StartAnalysis(ctx context.Context, input AnalysisInput) (string, error) {
	workflowID := fmt.Sprintf("k8s-analysis-%s-%d", input.ClusterID, time.Now().Unix())

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: c.taskQueue,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, K8sAnalysisWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	return run.GetID(), nil
}

// StartContinuousMonitoring starts continuous monitoring.
func (c *Client) StartContinuousMonitoring(ctx context.Context, config MonitoringConfig) (string, error) {
	workflowID := fmt.Sprintf("k8s-monitoring-%s", config.ClusterID)

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: c.taskQueue,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, ContinuousMonitoringWorkflow, config)
	if err != nil {
		return "", fmt.Errorf("failed to start monitoring workflow: %w", err)
	}

	return run.GetID(), nil
}

// StartEventAnalysis starts event-driven analysis.
func (c *Client) StartEventAnalysis(ctx context.Context, input EventAnalysisInput) (string, error) {
	workflowID := fmt.Sprintf("k8s-event-%s-%d", input.Event.ID, time.Now().Unix())

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: c.taskQueue,
	}

	run, err := c.client.ExecuteWorkflow(ctx, options, EventAnalysisWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start event analysis workflow: %w", err)
	}

	return run.GetID(), nil
}

// GetWorkflowResult retrieves the result of a completed workflow.
func (c *Client) GetWorkflowResult(ctx context.Context, workflowID string) (*AnalysisResult, error) {
	run := c.client.GetWorkflow(ctx, workflowID, "")

	var result AnalysisResult
	if err := run.Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("failed to get workflow result: %w", err)
	}

	return &result, nil
}

// CancelWorkflow cancels a running workflow.
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	return c.client.CancelWorkflow(ctx, workflowID, "")
}

// GetRawClient returns the underlying Temporal client.
func (c *Client) GetRawClient() client.Client {
	return c.client
}
