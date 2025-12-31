package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PerformanceAnalyzer analyzes performance of Kubernetes resources.
type PerformanceAnalyzer struct {
	name string
}

// NewPerformanceAnalyzer creates a new performance analyzer.
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		name: "performance",
	}
}

// Name returns the analyzer name.
func (a *PerformanceAnalyzer) Name() string {
	return a.name
}

// Analyze performs performance analysis.
func (a *PerformanceAnalyzer) Analyze(ctx context.Context, input AnalyzerInput) ([]Finding, error) {
	var findings []Finding

	if input.ClusterState == nil {
		return findings, nil
	}

	// Analyze nodes
	nodeFindings := a.analyzeNodes(input.ClusterState.Nodes, input.Metrics)
	findings = append(findings, nodeFindings...)

	// Analyze pods
	podFindings := a.analyzePods(input.ClusterState.Pods, input.Metrics)
	findings = append(findings, podFindings...)

	// Analyze deployments
	deploymentFindings := a.analyzeDeployments(input.ClusterState.Deployments)
	findings = append(findings, deploymentFindings...)

	// Analyze events for issues
	eventFindings := a.analyzeEvents(input.Events)
	findings = append(findings, eventFindings...)

	return findings, nil
}

func (a *PerformanceAnalyzer) analyzeNodes(nodes []Node, metrics *MetricsData) []Finding {
	var findings []Finding

	for _, node := range nodes {
		// Check node conditions
		for _, condition := range node.Conditions {
			// Check for memory pressure
			if condition.Type == "MemoryPressure" && condition.Status == "True" {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "NODE",
					Resource: ResourceRef{Kind: "Node", Name: node.Name},
					Title:    "Node memory pressure detected",
					Description: fmt.Sprintf("Node %s is under memory pressure. "+
						"Pods may be evicted and new pods may not schedule", node.Name),
					Evidence:    []string{condition.Message},
					Remediation: "Add nodes or reduce memory usage on existing workloads",
					Timestamp:   time.Now(),
				})
			}

			// Check for disk pressure
			if condition.Type == "DiskPressure" && condition.Status == "True" {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "NODE",
					Resource: ResourceRef{Kind: "Node", Name: node.Name},
					Title:    "Node disk pressure detected",
					Description: fmt.Sprintf("Node %s is under disk pressure. "+
						"Pods may be evicted", node.Name),
					Evidence:    []string{condition.Message},
					Remediation: "Clean up disk space or expand storage",
					Timestamp:   time.Now(),
				})
			}

			// Check for PID pressure
			if condition.Type == "PIDPressure" && condition.Status == "True" {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "NODE",
					Resource: ResourceRef{Kind: "Node", Name: node.Name},
					Title:    "Node PID pressure detected",
					Description: fmt.Sprintf("Node %s has too many processes. "+
						"New pods may fail to start", node.Name),
					Evidence:    []string{condition.Message},
					Remediation: "Investigate processes on the node and increase PID limits if needed",
					Timestamp:   time.Now(),
				})
			}

			// Check for not ready
			if condition.Type == "Ready" && condition.Status != "True" {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "NODE",
					Resource: ResourceRef{Kind: "Node", Name: node.Name},
					Title:    "Node not ready",
					Description: fmt.Sprintf("Node %s is not ready. "+
						"Reason: %s", node.Name, condition.Reason),
					Evidence:    []string{condition.Message},
					Remediation: "Investigate node health and kubelet status",
					Timestamp:   time.Now(),
				})
			}
		}

		// Check if node is unschedulable
		if node.Unschedulable {
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityMedium,
				Category: "NODE",
				Resource: ResourceRef{Kind: "Node", Name: node.Name},
				Title:    "Node is unschedulable",
				Description: fmt.Sprintf("Node %s is marked as unschedulable (cordoned)", node.Name),
				Remediation: "Uncordon the node if maintenance is complete: kubectl uncordon " + node.Name,
				Timestamp:   time.Now(),
			})
		}

		// Check metrics if available
		if metrics != nil && metrics.NodeMetrics != nil {
			if nodeMetrics, ok := metrics.NodeMetrics[node.Name]; ok {
				if nodeMetrics.CPUUsage > 85 {
					findings = append(findings, Finding{
						ID:       uuid.New().String(),
						Severity: SeverityHigh,
						Category: "RESOURCE",
						Resource: ResourceRef{Kind: "Node", Name: node.Name},
						Title:    "High CPU utilization",
						Description: fmt.Sprintf("Node %s CPU utilization is %.1f%%",
							node.Name, nodeMetrics.CPUUsage),
						Metadata: map[string]interface{}{
							"cpu_usage_percent": nodeMetrics.CPUUsage,
						},
						Remediation: "Scale cluster horizontally or optimize CPU-intensive workloads",
						Timestamp:   time.Now(),
					})
				}

				if nodeMetrics.MemoryUsage > 80 {
					findings = append(findings, Finding{
						ID:       uuid.New().String(),
						Severity: SeverityHigh,
						Category: "RESOURCE",
						Resource: ResourceRef{Kind: "Node", Name: node.Name},
						Title:    "High memory utilization",
						Description: fmt.Sprintf("Node %s memory utilization is %.1f%%",
							node.Name, nodeMetrics.MemoryUsage),
						Metadata: map[string]interface{}{
							"memory_usage_percent": nodeMetrics.MemoryUsage,
						},
						Remediation: "Add nodes or reduce memory usage on workloads",
						Timestamp:   time.Now(),
					})
				}

				if nodeMetrics.DiskUsage > 85 {
					findings = append(findings, Finding{
						ID:       uuid.New().String(),
						Severity: SeverityHigh,
						Category: "RESOURCE",
						Resource: ResourceRef{Kind: "Node", Name: node.Name},
						Title:    "High disk utilization",
						Description: fmt.Sprintf("Node %s disk utilization is %.1f%%",
							node.Name, nodeMetrics.DiskUsage),
						Metadata: map[string]interface{}{
							"disk_usage_percent": nodeMetrics.DiskUsage,
						},
						Remediation: "Clean up unused images and data, or expand storage",
						Timestamp:   time.Now(),
					})
				}
			}
		}
	}

	return findings
}

func (a *PerformanceAnalyzer) analyzePods(pods []Pod, metrics *MetricsData) []Finding {
	var findings []Finding

	for _, pod := range pods {
		// Check for high restart count
		if pod.RestartCount > 5 {
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityHigh,
				Category: "HEALTH",
				Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
				Title:    "Pod restart loop detected",
				Description: fmt.Sprintf("Pod %s has restarted %d times, indicating instability",
					pod.Name, pod.RestartCount),
				Metadata: map[string]interface{}{
					"restart_count": pod.RestartCount,
				},
				Remediation: "Check pod logs and events for root cause",
				Timestamp:   time.Now(),
			})
		}

		// Check for CrashLoopBackOff
		for _, container := range pod.Containers {
			if container.State.Waiting != nil && container.State.Waiting.Reason == "CrashLoopBackOff" {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "HEALTH",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "CrashLoopBackOff detected",
					Description: fmt.Sprintf("Container %s in pod %s is in CrashLoopBackOff",
						container.Name, pod.Name),
					Evidence: []string{container.State.Waiting.Message},
					Remediation: "Check container logs: kubectl logs " + pod.Name + " -c " + container.Name,
					Timestamp:   time.Now(),
				})
			}

			// Check for ImagePullBackOff
			if container.State.Waiting != nil &&
				(container.State.Waiting.Reason == "ImagePullBackOff" ||
					container.State.Waiting.Reason == "ErrImagePull") {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "HEALTH",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Image pull failure",
					Description: fmt.Sprintf("Container %s in pod %s cannot pull image %s",
						container.Name, pod.Name, container.Image),
					Evidence: []string{container.State.Waiting.Message},
					Remediation: "Check image name, tag, and registry credentials",
					Timestamp:   time.Now(),
				})
			}
		}

		// Check for pending pods
		if pod.Phase == "Pending" {
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityHigh,
				Category: "HEALTH",
				Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
				Title:    "Pod stuck in Pending state",
				Description: fmt.Sprintf("Pod %s cannot be scheduled", pod.Name),
				Remediation: "Check node resources, taints/tolerations, and affinity rules",
				Timestamp:   time.Now(),
			})
		}

		// Check for missing resource requests/limits
		for _, container := range pod.Containers {
			if container.Resources.Requests.CPU == 0 || container.Resources.Requests.Memory == 0 {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityMedium,
					Category: "RESOURCE",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Missing resource requests",
					Description: fmt.Sprintf("Container %s has no resource requests defined",
						container.Name),
					Remediation: "Define CPU and memory requests for better scheduling",
					Timestamp:   time.Now(),
				})
			}

			if container.Resources.Limits.CPU == 0 || container.Resources.Limits.Memory == 0 {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityMedium,
					Category: "RESOURCE",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Missing resource limits",
					Description: fmt.Sprintf("Container %s has no resource limits defined",
						container.Name),
					Remediation: "Define CPU and memory limits to prevent resource exhaustion",
					Timestamp:   time.Now(),
				})
			}
		}

		// Check QoS class
		if pod.QoSClass == "BestEffort" {
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityLow,
				Category: "RESOURCE",
				Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
				Title:    "BestEffort QoS class",
				Description: fmt.Sprintf("Pod %s has BestEffort QoS class and will be "+
					"first to be evicted under resource pressure", pod.Name),
				Remediation: "Define resource requests and limits for Guaranteed or Burstable QoS",
				Timestamp:   time.Now(),
			})
		}
	}

	return findings
}

func (a *PerformanceAnalyzer) analyzeDeployments(deployments []Deployment) []Finding {
	var findings []Finding

	for _, deployment := range deployments {
		// Check for unavailable replicas
		if deployment.ReadyReplicas < deployment.Replicas {
			unavailable := deployment.Replicas - deployment.ReadyReplicas
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityHigh,
				Category: "AVAILABILITY",
				Resource: ResourceRef{Kind: "Deployment", Name: deployment.Name, Namespace: deployment.Namespace},
				Title:    "Deployment has unavailable replicas",
				Description: fmt.Sprintf("Deployment %s has %d/%d replicas ready",
					deployment.Name, deployment.ReadyReplicas, deployment.Replicas),
				Metadata: map[string]interface{}{
					"replicas":       deployment.Replicas,
					"ready_replicas": deployment.ReadyReplicas,
					"unavailable":    unavailable,
				},
				Remediation: "Check pod status and events for the deployment",
				Timestamp:   time.Now(),
			})
		}

		// Check for single replica deployments
		if deployment.Replicas == 1 {
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityMedium,
				Category: "AVAILABILITY",
				Resource: ResourceRef{Kind: "Deployment", Name: deployment.Name, Namespace: deployment.Namespace},
				Title:    "Single replica deployment",
				Description: fmt.Sprintf("Deployment %s has only 1 replica, no high availability",
					deployment.Name),
				Remediation: "Consider increasing replica count for high availability",
				Timestamp:   time.Now(),
			})
		}
	}

	return findings
}

func (a *PerformanceAnalyzer) analyzeEvents(events []Event) []Finding {
	var findings []Finding

	for _, event := range events {
		// Check for warning events
		if event.Type == "Warning" {
			severity := SeverityMedium
			switch event.Reason {
			case "OOMKilled", "OOMKilling":
				severity = SeverityCritical
			case "FailedScheduling", "BackOff", "Unhealthy":
				severity = SeverityHigh
			case "FailedMount", "FailedAttachVolume":
				severity = SeverityHigh
			}

			if event.Count > 3 {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: severity,
					Category: "EVENT",
					Resource: event.InvolvedObject,
					Title:    fmt.Sprintf("Repeated warning event: %s", event.Reason),
					Description: fmt.Sprintf("Event occurred %d times: %s",
						event.Count, event.Message),
					Metadata: map[string]interface{}{
						"count":      event.Count,
						"first_seen": event.FirstSeen,
						"last_seen":  event.LastSeen,
					},
					Remediation: "Investigate the root cause of the repeated warning",
					Timestamp:   time.Now(),
				})
			}
		}
	}

	return findings
}
