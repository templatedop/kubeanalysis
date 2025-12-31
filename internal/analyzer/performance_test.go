package analyzer

import (
	"context"
	"testing"
	"time"
)

func TestPerformanceAnalyzerName(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	if analyzer.Name() != "performance" {
		t.Errorf("expected name 'performance', got %s", analyzer.Name())
	}
}

func TestPerformanceAnalyzerEmptyInput(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: nil,
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for nil cluster state, got %d", len(findings))
	}
}

func TestPerformanceAnalyzerNodeMemoryPressure(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "memory-pressure-node",
					Conditions: []Condition{
						{
							Type:    "MemoryPressure",
							Status:  "True",
							Message: "Node is under memory pressure",
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Node memory pressure detected" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity, got %s", f.Severity)
			}
			if f.Category != "NODE" {
				t.Errorf("expected NODE category, got %s", f.Category)
			}
			if f.Resource.Kind != "Node" {
				t.Errorf("expected Node resource kind, got %s", f.Resource.Kind)
			}
		}
	}
	if !found {
		t.Error("expected to find memory pressure issue")
	}
}

func TestPerformanceAnalyzerNodeDiskPressure(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "disk-pressure-node",
					Conditions: []Condition{
						{
							Type:    "DiskPressure",
							Status:  "True",
							Message: "Node is under disk pressure",
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Node disk pressure detected" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find disk pressure issue")
	}
}

func TestPerformanceAnalyzerNodePIDPressure(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "pid-pressure-node",
					Conditions: []Condition{
						{
							Type:    "PIDPressure",
							Status:  "True",
							Message: "Too many processes",
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Node PID pressure detected" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find PID pressure issue")
	}
}

func TestPerformanceAnalyzerNodeNotReady(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "not-ready-node",
					Conditions: []Condition{
						{
							Type:    "Ready",
							Status:  "False",
							Reason:  "KubeletNotReady",
							Message: "Kubelet is not ready",
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Node not ready" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find node not ready issue")
	}
}

func TestPerformanceAnalyzerNodeUnschedulable(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name:          "cordoned-node",
					Unschedulable: true,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Node is unschedulable" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("expected MEDIUM severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find unschedulable node issue")
	}
}

func TestPerformanceAnalyzerHighCPUUsage(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "high-cpu-node",
				},
			},
		},
		Metrics: &MetricsData{
			NodeMetrics: map[string]NodeMetrics{
				"high-cpu-node": {
					CPUUsage:    90.5,
					MemoryUsage: 50.0,
					DiskUsage:   30.0,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "High CPU utilization" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
			if f.Category != "RESOURCE" {
				t.Errorf("expected RESOURCE category, got %s", f.Category)
			}
			if f.Metadata["cpu_usage_percent"] != 90.5 {
				t.Errorf("expected cpu_usage_percent 90.5, got %v", f.Metadata["cpu_usage_percent"])
			}
		}
	}
	if !found {
		t.Error("expected to find high CPU usage issue")
	}
}

func TestPerformanceAnalyzerHighMemoryUsage(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "high-memory-node",
				},
			},
		},
		Metrics: &MetricsData{
			NodeMetrics: map[string]NodeMetrics{
				"high-memory-node": {
					CPUUsage:    50.0,
					MemoryUsage: 85.5,
					DiskUsage:   30.0,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "High memory utilization" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find high memory usage issue")
	}
}

func TestPerformanceAnalyzerHighDiskUsage(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "high-disk-node",
				},
			},
		},
		Metrics: &MetricsData{
			NodeMetrics: map[string]NodeMetrics{
				"high-disk-node": {
					CPUUsage:    50.0,
					MemoryUsage: 50.0,
					DiskUsage:   92.0,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "High disk utilization" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find high disk usage issue")
	}
}

func TestPerformanceAnalyzerPodRestartLoop(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:         "restarting-pod",
					Namespace:    "default",
					RestartCount: 10,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Pod restart loop detected" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
			if f.Category != "HEALTH" {
				t.Errorf("expected HEALTH category, got %s", f.Category)
			}
			if f.Metadata["restart_count"] != int32(10) {
				t.Errorf("expected restart_count 10, got %v", f.Metadata["restart_count"])
			}
		}
	}
	if !found {
		t.Error("expected to find pod restart loop issue")
	}
}

func TestPerformanceAnalyzerCrashLoopBackOff(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "crashing-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "crashing-container",
							State: ContainerState{
								Waiting: &ContainerStateWaiting{
									Reason:  "CrashLoopBackOff",
									Message: "Container keeps crashing",
								},
							},
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "CrashLoopBackOff detected" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find CrashLoopBackOff issue")
	}
}

func TestPerformanceAnalyzerImagePullBackOff(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "image-pull-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name:  "container",
							Image: "nonexistent/image:latest",
							State: ContainerState{
								Waiting: &ContainerStateWaiting{
									Reason:  "ImagePullBackOff",
									Message: "Cannot pull image",
								},
							},
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Image pull failure" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find image pull failure issue")
	}
}

func TestPerformanceAnalyzerErrImagePull(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "image-err-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name:  "container",
							Image: "invalid/image",
							State: ContainerState{
								Waiting: &ContainerStateWaiting{
									Reason:  "ErrImagePull",
									Message: "Error pulling image",
								},
							},
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Image pull failure" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find ErrImagePull issue")
	}
}

func TestPerformanceAnalyzerPendingPod(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "pending-pod",
					Namespace: "default",
					Phase:     "Pending",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Pod stuck in Pending state" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find pending pod issue")
	}
}

func TestPerformanceAnalyzerMissingResourceRequests(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "no-resources-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "container",
							Resources: ContainerResources{
								Requests: Resources{CPU: 0, Memory: 0},
								Limits:   Resources{CPU: 100, Memory: 256 * 1024 * 1024},
							},
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Missing resource requests" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("expected MEDIUM severity, got %s", f.Severity)
			}
			if f.Category != "RESOURCE" {
				t.Errorf("expected RESOURCE category, got %s", f.Category)
			}
		}
	}
	if !found {
		t.Error("expected to find missing resource requests issue")
	}
}

func TestPerformanceAnalyzerMissingResourceLimits(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "no-limits-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "container",
							Resources: ContainerResources{
								Requests: Resources{CPU: 100, Memory: 256 * 1024 * 1024},
								Limits:   Resources{CPU: 0, Memory: 0},
							},
						},
					},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Missing resource limits" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("expected MEDIUM severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find missing resource limits issue")
	}
}

func TestPerformanceAnalyzerBestEffortQoS(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "besteffort-pod",
					Namespace: "default",
					QoSClass:  "BestEffort",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "BestEffort QoS class" {
			found = true
			if f.Severity != SeverityLow {
				t.Errorf("expected LOW severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find BestEffort QoS issue")
	}
}

func TestPerformanceAnalyzerDeploymentUnavailable(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Deployments: []Deployment{
				{
					Name:          "partial-deployment",
					Namespace:     "default",
					Replicas:      3,
					ReadyReplicas: 1,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Deployment has unavailable replicas" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
			if f.Category != "AVAILABILITY" {
				t.Errorf("expected AVAILABILITY category, got %s", f.Category)
			}
			if f.Metadata["replicas"] != int32(3) {
				t.Errorf("expected replicas 3, got %v", f.Metadata["replicas"])
			}
			if f.Metadata["ready_replicas"] != int32(1) {
				t.Errorf("expected ready_replicas 1, got %v", f.Metadata["ready_replicas"])
			}
		}
	}
	if !found {
		t.Error("expected to find unavailable replicas issue")
	}
}

func TestPerformanceAnalyzerSingleReplicaDeployment(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Deployments: []Deployment{
				{
					Name:          "single-replica-deployment",
					Namespace:     "default",
					Replicas:      1,
					ReadyReplicas: 1,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Single replica deployment" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("expected MEDIUM severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find single replica deployment issue")
	}
}

func TestPerformanceAnalyzerRepeatedWarningEvents(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	now := time.Now()
	input := AnalyzerInput{
		ClusterState: &ClusterState{},
		Events: []Event{
			{
				Type:      "Warning",
				Reason:    "BackOff",
				Message:   "Back-off restarting failed container",
				Count:     5,
				FirstSeen: now.Add(-1 * time.Hour),
				LastSeen:  now,
				InvolvedObject: ResourceRef{
					Kind:      "Pod",
					Name:      "problematic-pod",
					Namespace: "default",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Repeated warning event: BackOff" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
			if f.Category != "EVENT" {
				t.Errorf("expected EVENT category, got %s", f.Category)
			}
		}
	}
	if !found {
		t.Error("expected to find repeated warning event issue")
	}
}

func TestPerformanceAnalyzerOOMKilledEvent(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	now := time.Now()
	input := AnalyzerInput{
		ClusterState: &ClusterState{},
		Events: []Event{
			{
				Type:      "Warning",
				Reason:    "OOMKilled",
				Message:   "Container was killed due to OOM",
				Count:     4,
				FirstSeen: now.Add(-30 * time.Minute),
				LastSeen:  now,
				InvolvedObject: ResourceRef{
					Kind:      "Pod",
					Name:      "oom-pod",
					Namespace: "default",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Repeated warning event: OOMKilled" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity for OOMKilled, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find OOMKilled event issue")
	}
}

func TestPerformanceAnalyzerFailedSchedulingEvent(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	now := time.Now()
	input := AnalyzerInput{
		ClusterState: &ClusterState{},
		Events: []Event{
			{
				Type:      "Warning",
				Reason:    "FailedScheduling",
				Message:   "No nodes available",
				Count:     5,
				FirstSeen: now.Add(-15 * time.Minute),
				LastSeen:  now,
				InvolvedObject: ResourceRef{
					Kind:      "Pod",
					Name:      "unscheduled-pod",
					Namespace: "default",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Repeated warning event: FailedScheduling" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity for FailedScheduling, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find FailedScheduling event issue")
	}
}

func TestPerformanceAnalyzerFailedMountEvent(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	now := time.Now()
	input := AnalyzerInput{
		ClusterState: &ClusterState{},
		Events: []Event{
			{
				Type:      "Warning",
				Reason:    "FailedMount",
				Message:   "Unable to mount volume",
				Count:     4,
				FirstSeen: now.Add(-10 * time.Minute),
				LastSeen:  now,
				InvolvedObject: ResourceRef{
					Kind:      "Pod",
					Name:      "volume-pod",
					Namespace: "default",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Title == "Repeated warning event: FailedMount" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity for FailedMount, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find FailedMount event issue")
	}
}

func TestPerformanceAnalyzerLowEventCount(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	now := time.Now()
	input := AnalyzerInput{
		ClusterState: &ClusterState{},
		Events: []Event{
			{
				Type:      "Warning",
				Reason:    "BackOff",
				Message:   "Back-off restarting",
				Count:     2, // Below threshold of 3
				FirstSeen: now.Add(-5 * time.Minute),
				LastSeen:  now,
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not find any issue for event count <= 3
	for _, f := range findings {
		if f.Category == "EVENT" {
			t.Error("should not report events with count <= 3")
		}
	}
}

func TestPerformanceAnalyzerNormalEventIgnored(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	now := time.Now()
	input := AnalyzerInput{
		ClusterState: &ClusterState{},
		Events: []Event{
			{
				Type:      "Normal",
				Reason:    "Scheduled",
				Message:   "Successfully scheduled",
				Count:     10,
				FirstSeen: now.Add(-1 * time.Hour),
				LastSeen:  now,
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not find any issue for normal events
	for _, f := range findings {
		if f.Category == "EVENT" {
			t.Error("should not report normal events")
		}
	}
}

func TestPerformanceAnalyzerMultipleIssues(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name:          "problem-node",
					Unschedulable: true,
					Conditions: []Condition{
						{
							Type:   "MemoryPressure",
							Status: "True",
						},
					},
				},
			},
			Pods: []Pod{
				{
					Name:         "unstable-pod",
					Namespace:    "default",
					RestartCount: 20,
					QoSClass:     "BestEffort",
				},
			},
			Deployments: []Deployment{
				{
					Name:          "failing-deployment",
					Namespace:     "default",
					Replicas:      5,
					ReadyReplicas: 2,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find multiple issues
	if len(findings) < 4 {
		t.Errorf("expected at least 4 findings, got %d", len(findings))
	}

	// Verify we have findings from different categories
	categories := make(map[string]bool)
	for _, f := range findings {
		categories[f.Category] = true
	}

	expectedCategories := []string{"NODE", "HEALTH", "RESOURCE", "AVAILABILITY"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("expected findings in category %s", cat)
		}
	}
}

func TestPerformanceAnalyzerHealthyCluster(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "healthy-node",
					Conditions: []Condition{
						{Type: "Ready", Status: "True"},
						{Type: "MemoryPressure", Status: "False"},
						{Type: "DiskPressure", Status: "False"},
						{Type: "PIDPressure", Status: "False"},
					},
				},
			},
			Pods: []Pod{
				{
					Name:         "healthy-pod",
					Namespace:    "default",
					Phase:        "Running",
					RestartCount: 0,
					QoSClass:     "Guaranteed",
					Containers: []Container{
						{
							Name: "container",
							Resources: ContainerResources{
								Requests: Resources{CPU: 100, Memory: 256 * 1024 * 1024},
								Limits:   Resources{CPU: 200, Memory: 512 * 1024 * 1024},
							},
							State: ContainerState{
								Running: &ContainerStateRunning{
									StartedAt: time.Now().Add(-24 * time.Hour),
								},
							},
						},
					},
				},
			},
			Deployments: []Deployment{
				{
					Name:          "healthy-deployment",
					Namespace:     "default",
					Replicas:      3,
					ReadyReplicas: 3,
				},
			},
		},
		Metrics: &MetricsData{
			NodeMetrics: map[string]NodeMetrics{
				"healthy-node": {
					CPUUsage:    50.0,
					MemoryUsage: 60.0,
					DiskUsage:   40.0,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have no critical or high severity findings
	for _, f := range findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
			t.Errorf("healthy cluster should not have critical/high findings, got: %s - %s",
				f.Severity, f.Title)
		}
	}
}

func TestPerformanceAnalyzerResourceRefPopulated(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Nodes: []Node{
				{
					Name: "test-node",
					Conditions: []Condition{
						{Type: "MemoryPressure", Status: "True"},
					},
				},
			},
			Pods: []Pod{
				{
					Name:         "test-pod",
					Namespace:    "test-ns",
					RestartCount: 10,
				},
			},
			Deployments: []Deployment{
				{
					Name:          "test-deployment",
					Namespace:     "test-ns",
					Replicas:      2,
					ReadyReplicas: 1,
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range findings {
		if f.Resource.Kind == "" {
			t.Error("finding should have resource kind populated")
		}
		if f.Resource.Name == "" {
			t.Error("finding should have resource name populated")
		}
		if f.Resource.Kind == "Pod" || f.Resource.Kind == "Deployment" {
			if f.Resource.Namespace == "" {
				t.Errorf("namespaced resource %s should have namespace populated", f.Resource.Kind)
			}
		}
	}
}
