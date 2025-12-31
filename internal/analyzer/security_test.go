package analyzer

import (
	"context"
	"testing"
)

func TestSecurityAnalyzerName(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	if analyzer.Name() != "security" {
		t.Errorf("expected name 'security', got %s", analyzer.Name())
	}
}

func TestSecurityAnalyzerPrivilegedContainers(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	privileged := true
	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "privileged-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "privileged-container",
							SecurityContext: &ContainerSecurityContext{
								Privileged: privileged,
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

	// Should find privileged container
	found := false
	for _, f := range findings {
		if f.Title == "Privileged container detected" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity, got %s", f.Severity)
			}
			if f.Category != "POD_SECURITY" {
				t.Errorf("expected POD_SECURITY category, got %s", f.Category)
			}
			if f.Framework != "NSA" {
				t.Errorf("expected NSA framework, got %s", f.Framework)
			}
		}
	}
	if !found {
		t.Error("expected to find privileged container issue")
	}
}

func TestSecurityAnalyzerRootUser(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "root-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "root-container",
							// No security context - will run as root
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

	// Should find root user issue
	found := false
	for _, f := range findings {
		if f.Title == "Container may run as root" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find root user issue")
	}
}

func TestSecurityAnalyzerHostNamespaces(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "host-ns-pod",
					Namespace: "default",
					SecurityContext: &PodSecurityContext{
						HostNetwork: true,
						HostPID:     true,
						HostIPC:     true,
					},
					Containers: []Container{{Name: "container"}},
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find host namespace issues
	hostNetworkFound := false
	hostPIDFound := false
	hostIPCFound := false

	for _, f := range findings {
		switch f.Title {
		case "Pod uses host network namespace":
			hostNetworkFound = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity for host network, got %s", f.Severity)
			}
		case "Pod uses host PID namespace":
			hostPIDFound = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity for host PID, got %s", f.Severity)
			}
		case "Pod uses host IPC namespace":
			hostIPCFound = true
		}
	}

	if !hostNetworkFound {
		t.Error("expected to find host network issue")
	}
	if !hostPIDFound {
		t.Error("expected to find host PID issue")
	}
	if !hostIPCFound {
		t.Error("expected to find host IPC issue")
	}
}

func TestSecurityAnalyzerDangerousCapabilities(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "cap-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "cap-container",
							SecurityContext: &ContainerSecurityContext{
								Capabilities: &Capabilities{
									Add: []string{"SYS_ADMIN", "NET_ADMIN"},
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

	// Should find dangerous capabilities
	capCount := 0
	for _, f := range findings {
		if f.Title == "Dangerous capability added" {
			capCount++
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}

	if capCount < 2 {
		t.Errorf("expected at least 2 capability findings, got %d", capCount)
	}
}

func TestSecurityAnalyzerRBACClusterAdmin(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			RBAC: &RBACState{
				ClusterRoleBindings: []ClusterRoleBinding{
					{
						Name: "admin-binding",
						RoleRef: RoleRef{
							Kind: "ClusterRole",
							Name: "cluster-admin",
						},
						Subjects: []Subject{
							{
								Kind:      "ServiceAccount",
								Name:      "my-sa",
								Namespace: "default",
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

	// Should find cluster-admin binding
	found := false
	for _, f := range findings {
		if f.Title == "Cluster-admin binding detected" {
			found = true
			if f.Severity != SeverityCritical {
				t.Errorf("expected CRITICAL severity, got %s", f.Severity)
			}
			if f.Category != "RBAC" {
				t.Errorf("expected RBAC category, got %s", f.Category)
			}
		}
	}
	if !found {
		t.Error("expected to find cluster-admin binding issue")
	}
}

func TestSecurityAnalyzerRBACWildcard(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			RBAC: &RBACState{
				ClusterRoles: []ClusterRole{
					{
						Name: "wildcard-role",
						Rules: []PolicyRule{
							{
								Verbs:     []string{"*"},
								Resources: []string{"pods"},
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

	// Should find wildcard permission
	found := false
	for _, f := range findings {
		if f.Title == "Wildcard permissions in ClusterRole" {
			found = true
			if f.Severity != SeverityHigh {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected to find wildcard permission issue")
	}
}

func TestSecurityAnalyzerNetworkPolicies(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "unprotected-pod",
					Namespace: "default",
					Containers: []Container{
						{Name: "container"},
					},
				},
			},
			NetworkPolicies: []NetworkPolicy{
				// No policies for "default" namespace
				{
					Name:      "policy",
					Namespace: "other-ns",
				},
			},
		},
	}

	findings, err := analyzer.Analyze(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find missing network policy
	found := false
	for _, f := range findings {
		if f.Title == "No network policy in namespace" {
			found = true
			if f.Severity != SeverityMedium {
				t.Errorf("expected MEDIUM severity, got %s", f.Severity)
			}
			if f.Category != "NETWORK" {
				t.Errorf("expected NETWORK category, got %s", f.Category)
			}
		}
	}
	if !found {
		t.Error("expected to find missing network policy issue")
	}
}

func TestSecurityAnalyzerSecretsInEnv(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "secret-pod",
					Namespace: "default",
					Containers: []Container{
						{
							Name: "container",
							EnvVars: []EnvVar{
								{
									Name: "SECRET_KEY",
									ValueFrom: &EnvVarSource{
										SecretKeyRef: &SecretKeySelector{
											Name: "my-secret",
											Key:  "password",
										},
									},
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

	// Should find secret in env var
	found := false
	for _, f := range findings {
		if f.Title == "Secret exposed as environment variable" {
			found = true
			if f.Category != "SECRETS" {
				t.Errorf("expected SECRETS category, got %s", f.Category)
			}
		}
	}
	if !found {
		t.Error("expected to find secret in env var issue")
	}
}

func TestSecurityAnalyzerSkipsSystemNamespaces(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
	ctx := context.Background()

	privileged := true
	input := AnalyzerInput{
		ClusterState: &ClusterState{
			Pods: []Pod{
				{
					Name:      "system-pod",
					Namespace: "kube-system",
					Containers: []Container{
						{
							Name: "container",
							SecurityContext: &ContainerSecurityContext{
								Privileged: privileged,
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

	// Should not find issues in kube-system
	for _, f := range findings {
		if f.Resource.Namespace == "kube-system" {
			t.Error("should skip kube-system namespace")
		}
	}
}

func TestSecurityAnalyzerEmptyInput(t *testing.T) {
	analyzer := NewSecurityAnalyzer()
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

func TestIsDangerousCapability(t *testing.T) {
	dangerous := []string{
		"SYS_ADMIN", "NET_ADMIN", "SYS_PTRACE", "SYS_MODULE",
		"DAC_OVERRIDE", "SETUID", "SETGID", "NET_RAW",
	}

	for _, cap := range dangerous {
		if !isDangerousCapability(cap) {
			t.Errorf("expected %s to be dangerous", cap)
		}
	}

	safe := []string{"CHOWN", "FOWNER", "SYS_NICE"}
	for _, cap := range safe {
		if isDangerousCapability(cap) {
			t.Errorf("expected %s to not be dangerous", cap)
		}
	}
}

func TestContainsWildcard(t *testing.T) {
	tests := []struct {
		items    []string
		expected bool
	}{
		{[]string{"*"}, true},
		{[]string{"get", "list", "*"}, true},
		{[]string{"get", "list", "watch"}, false},
		{[]string{}, false},
	}

	for _, tt := range tests {
		result := containsWildcard(tt.items)
		if result != tt.expected {
			t.Errorf("containsWildcard(%v) = %v, expected %v", tt.items, result, tt.expected)
		}
	}
}
