// Package main provides the CLI entry point for kubeanalysis.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kubeanalysis/kubeanalysis/internal/config"
	"github.com/kubeanalysis/kubeanalysis/internal/rag"
	"github.com/kubeanalysis/kubeanalysis/internal/temporal"
)

const (
	version = "0.1.0"
	banner  = `
╔═══════════════════════════════════════════════════════════════════╗
║  KubeAnalysis - Kubernetes Analysis with RAG + Temporal           ║
║  Intelligent cluster analysis powered by LLM and RAG              ║
╚═══════════════════════════════════════════════════════════════════╝
`
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "analyze":
		runAnalyze()
	case "worker":
		runWorker()
	case "ingest":
		runIngest()
	case "init":
		runInit()
	case "version":
		fmt.Printf("kubeanalysis version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(banner)
	fmt.Print(`
Usage: kubeanalysis <command> [options]

Commands:
  analyze     Run a one-time cluster analysis
  worker      Start the Temporal worker for processing workflows
  ingest      Ingest knowledge documents into the RAG system
  init        Initialize a new configuration file
  version     Show version information
  help        Show this help message

Options:
  -c, --config     Path to configuration file (default: ./config.yaml)
  -n, --namespace  Kubernetes namespace(s) to analyze
  -t, --type       Analysis type: security, performance, prediction, full
  -o, --output     Output format: text, json, yaml
  -v, --verbose    Enable verbose output

Examples:
  # Run full analysis on default namespace
  kubeanalysis analyze -t full

  # Run security analysis on specific namespace
  kubeanalysis analyze -t security -n production

  # Start the Temporal worker
  kubeanalysis worker -c config.yaml

  # Ingest knowledge documents
  kubeanalysis ingest -c config.yaml --path ./knowledge

  # Initialize configuration
  kubeanalysis init -c config.yaml

For more information, visit: https://github.com/kubeanalysis/kubeanalysis
`)
}

func runAnalyze() {
	cfg := parseConfig()

	// Parse analysis options
	analysisType := "full"
	namespaces := []string{}
	outputFormat := "text"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--type":
			if i+1 < len(args) {
				analysisType = args[i+1]
				i++
			}
		case "-n", "--namespace":
			if i+1 < len(args) {
				namespaces = append(namespaces, args[i+1])
				i++
			}
		case "-o", "--output":
			if i+1 < len(args) {
				outputFormat = args[i+1]
				i++
			}
		}
	}

	fmt.Print(banner)
	fmt.Printf("Running %s analysis...\n\n", analysisType)

	// Create RAG engine
	ragEngine := createRAGEngine(cfg)

	// Create Temporal client
	client, err := temporal.NewClient(temporal.WorkerConfig{
		TemporalHost: cfg.Temporal.Host,
		Namespace:    cfg.Temporal.Namespace,
		TaskQueue:    cfg.Temporal.TaskQueue,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Temporal client: %v\n", err)
		fmt.Println("\nNote: Make sure Temporal server is running.")
		fmt.Println("You can start it with: temporal server start-dev")
		os.Exit(1)
	}
	defer client.Close()

	// Build analysis input
	input := temporal.AnalysisInput{
		ClusterID:    "default-cluster",
		AnalysisType: temporal.AnalysisType(analysisType),
		Namespaces:   namespaces,
	}

	// Start workflow
	ctx := context.Background()
	workflowID, err := client.StartAnalysis(ctx, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start analysis: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Started analysis workflow: %s\n", workflowID)
	fmt.Println("Waiting for results...")

	// Wait for result
	result, err := client.GetWorkflowResult(ctx, workflowID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		os.Exit(1)
	}

	// Print results
	printResults(result, outputFormat, ragEngine)
}

func runWorker() {
	cfg := parseConfig()

	fmt.Print(banner)
	fmt.Println("Starting Temporal worker...")
	fmt.Printf("  Host:       %s\n", cfg.Temporal.Host)
	fmt.Printf("  Namespace:  %s\n", cfg.Temporal.Namespace)
	fmt.Printf("  Task Queue: %s\n", cfg.Temporal.TaskQueue)
	fmt.Println()

	// Create RAG engine
	ragEngine := createRAGEngine(cfg)

	// Ingest prebuilt knowledge
	fmt.Println("Loading knowledge base...")
	loadPrebuiltKnowledge(ragEngine)

	// Create worker
	workerCfg := temporal.WorkerConfig{
		TemporalHost:            cfg.Temporal.Host,
		Namespace:               cfg.Temporal.Namespace,
		TaskQueue:               cfg.Temporal.TaskQueue,
		MaxConcurrentActivities: cfg.Temporal.MaxConcurrentActivities,
		MaxConcurrentWorkflows:  cfg.Temporal.MaxConcurrentWorkflows,
	}

	worker, err := temporal.NewWorker(workerCfg, ragEngine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create worker: %v\n", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down worker...")
		cancel()
		worker.Stop()
	}()

	fmt.Println("Worker started. Press Ctrl+C to stop.")
	if err := worker.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Worker error: %v\n", err)
		os.Exit(1)
	}
}

func runIngest() {
	cfg := parseConfig()

	// Parse ingest options
	knowledgePath := cfg.RAG.KnowledgePath

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--path":
			if i+1 < len(args) {
				knowledgePath = args[i+1]
				i++
			}
		}
	}

	fmt.Print(banner)
	fmt.Printf("Ingesting knowledge from: %s\n\n", knowledgePath)

	// Create RAG engine
	ragEngine := createRAGEngine(cfg)

	// Load prebuilt knowledge
	loadPrebuiltKnowledge(ragEngine)

	// Ingest from path if it exists
	if _, err := os.Stat(knowledgePath); err == nil {
		if err := ingestFromPath(ragEngine, knowledgePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to ingest knowledge: %v\n", err)
			os.Exit(1)
		}
	}

	// Show stats
	ctx := context.Background()
	count, _ := ragEngine.Count(ctx, "")
	fmt.Printf("\nKnowledge base ready: %d chunks indexed\n", count)
}

func runInit() {
	configPath := "./config.yaml"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		}
	}

	fmt.Print(banner)

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists: %s\n", configPath)
		fmt.Print("Overwrite? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return
		}
	}

	// Create default config
	cfg := config.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration file created: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit the configuration file to match your environment")
	fmt.Println("  2. Start the Temporal worker: kubeanalysis worker")
	fmt.Println("  3. Run an analysis: kubeanalysis analyze")
}

func parseConfig() *config.Config {
	configPath := ""

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		}
	}

	return config.LoadOrDefault(configPath)
}

func createRAGEngine(cfg *config.Config) *rag.Engine {
	ragCfg := rag.DefaultEngineConfig()
	ragCfg.DefaultTopK = cfg.RAG.TopK
	ragCfg.DefaultMinScore = cfg.RAG.MinScore
	ragCfg.MaxContextLength = cfg.RAG.MaxContextLength
	ragCfg.SplitterConfig.ChunkSize = cfg.RAG.ChunkSize
	ragCfg.SplitterConfig.ChunkOverlap = cfg.RAG.ChunkOverlap

	ragCfg.EmbedderConfig.Provider = cfg.Embedder.Provider
	ragCfg.EmbedderConfig.BaseURL = cfg.Embedder.BaseURL
	ragCfg.EmbedderConfig.APIKey = cfg.Embedder.APIKey
	ragCfg.EmbedderConfig.Model = cfg.Embedder.Model
	ragCfg.EmbedderConfig.EmbeddingDimension = cfg.Embedder.Dimension

	return rag.NewEngine(ragCfg, nil, nil, nil)
}

func loadPrebuiltKnowledge(engine *rag.Engine) {
	ctx := context.Background()
	kb := rag.NewK8sKnowledgeBase(engine)

	// Load prebuilt knowledge
	for name, content := range rag.PrebuiltKnowledge {
		doc := rag.Document{
			ID:      "prebuilt-" + name,
			Content: content,
			Metadata: map[string]string{
				"source":   "prebuilt",
				"category": "best_practice",
				"name":     name,
			},
		}
		if err := engine.IngestDocument(ctx, doc); err != nil {
			fmt.Printf("Warning: Failed to ingest %s: %v\n", name, err)
		}
	}

	// Register source
	engine.RegisterSource(&rag.KnowledgeSource{
		ID:          "prebuilt",
		Name:        "Prebuilt Knowledge",
		Description: "Built-in Kubernetes best practices",
		Category:    rag.CategoryBestPractice,
		Type:        "prebuilt",
	})

	_ = kb // Keep reference
}

func ingestFromPath(engine *rag.Engine, path string) error {
	ctx := context.Background()

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := path + "/" + entry.Name()
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s: %v\n", filePath, err)
			continue
		}

		doc := rag.Document{
			ID:      "file-" + entry.Name(),
			Content: string(content),
			Source:  filePath,
			Metadata: map[string]string{
				"source": filePath,
				"type":   "file",
			},
		}

		if err := engine.IngestDocument(ctx, doc); err != nil {
			fmt.Printf("Warning: Failed to ingest %s: %v\n", filePath, err)
			continue
		}

		fmt.Printf("  Ingested: %s\n", entry.Name())
	}

	return nil
}

func printResults(result *temporal.AnalysisResult, format string, ragEngine *rag.Engine) {
	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Println("                    ANALYSIS RESULTS")
	fmt.Println(strings.Repeat("═", 60) + "\n")

	fmt.Printf("Cluster:    %s\n", result.ClusterID)
	fmt.Printf("Type:       %s\n", result.AnalysisType)
	fmt.Printf("Duration:   %s\n", result.Duration)
	fmt.Printf("Status:     %s\n", result.Status)
	fmt.Println()

	// Security Results
	if result.Security != nil {
		fmt.Println("┌─────────────────────────────────────────────────────────┐")
		fmt.Println("│                    SECURITY ANALYSIS                    │")
		fmt.Println("└─────────────────────────────────────────────────────────┘")
		fmt.Printf("  Posture Score: %.1f/100 (%s)\n",
			result.Security.Summary.PostureScore,
			result.Security.Summary.RiskLevel)
		fmt.Printf("  Total Findings: %d\n", result.Security.Summary.TotalFindings)
		fmt.Printf("    Critical: %d\n", result.Security.Summary.CriticalCount)
		fmt.Printf("    High:     %d\n", result.Security.Summary.HighCount)
		fmt.Println()

		if len(result.Security.Findings) > 0 {
			fmt.Println("  Top Findings:")
			for i, f := range result.Security.Findings {
				if i >= 5 {
					break
				}
				fmt.Printf("  [%s] %s\n", f.Severity, f.Title)
				fmt.Printf("         %s/%s\n", f.Resource.Namespace, f.Resource.Name)
			}
		}
		fmt.Println()
	}

	// Performance Results
	if result.Performance != nil {
		fmt.Println("┌─────────────────────────────────────────────────────────┐")
		fmt.Println("│                   PERFORMANCE ANALYSIS                  │")
		fmt.Println("└─────────────────────────────────────────────────────────┘")
		fmt.Printf("  Health Score: %.1f/100\n", result.Performance.Summary.HealthScore)
		fmt.Printf("  Total Findings: %d\n", result.Performance.Summary.TotalFindings)
		fmt.Println()

		if len(result.Performance.Findings) > 0 {
			fmt.Println("  Top Findings:")
			for i, f := range result.Performance.Findings {
				if i >= 5 {
					break
				}
				fmt.Printf("  [%s] %s\n", f.Severity, f.Title)
			}
		}
		fmt.Println()
	}

	// Predictions
	if result.Predictions != nil && len(result.Predictions.Predictions) > 0 {
		fmt.Println("┌─────────────────────────────────────────────────────────┐")
		fmt.Println("│                      PREDICTIONS                        │")
		fmt.Println("└─────────────────────────────────────────────────────────┘")
		fmt.Printf("  Total Predictions: %d\n", result.Predictions.Summary.TotalPredictions)
		fmt.Printf("  High Risk: %d\n", result.Predictions.Summary.HighRiskCount)
		fmt.Println()

		for i, p := range result.Predictions.Predictions {
			if i >= 3 {
				break
			}
			fmt.Printf("  [%s] %s\n", p.Severity, p.Title)
			fmt.Printf("         Confidence: %.0f%%\n", p.Confidence*100)
		}
		fmt.Println()
	}

	// LLM Insights
	if result.LLMInsights != nil {
		fmt.Println("┌─────────────────────────────────────────────────────────┐")
		fmt.Println("│                    LLM INSIGHTS                         │")
		fmt.Println("└─────────────────────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("  Executive Summary:")
		fmt.Printf("  %s\n", result.LLMInsights.ExecutiveSummary)
		fmt.Println()

		if len(result.LLMInsights.PrioritizedActions) > 0 {
			fmt.Println("  Recommended Actions:")
			for _, action := range result.LLMInsights.PrioritizedActions {
				fmt.Printf("  • %s\n", action)
			}
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("═", 60) + "\n")
}
