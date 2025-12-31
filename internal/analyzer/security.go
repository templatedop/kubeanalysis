package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SecurityAnalyzer analyzes security configuration of Kubernetes resources.
type SecurityAnalyzer struct {
	name string
}

// NewSecurityAnalyzer creates a new security analyzer.
func NewSecurityAnalyzer() *SecurityAnalyzer {
	return &SecurityAnalyzer{
		name: "security",
	}
}

// Name returns the analyzer name.
func (a *SecurityAnalyzer) Name() string {
	return a.name
}

// Analyze performs security analysis.
func (a *SecurityAnalyzer) Analyze(ctx context.Context, input AnalyzerInput) ([]Finding, error) {
	var findings []Finding

	if input.ClusterState == nil {
		return findings, nil
	}

	// Analyze pods
	podFindings := a.analyzePods(input.ClusterState.Pods)
	findings = append(findings, podFindings...)

	// Analyze RBAC
	if input.ClusterState.RBAC != nil {
		rbacFindings := a.analyzeRBAC(input.ClusterState.RBAC)
		findings = append(findings, rbacFindings...)
	}

	// Analyze network policies
	netpolFindings := a.analyzeNetworkPolicies(input.ClusterState.Pods, input.ClusterState.NetworkPolicies)
	findings = append(findings, netpolFindings...)

	// Analyze secrets
	secretFindings := a.analyzeSecrets(input.ClusterState.Secrets, input.ClusterState.Pods)
	findings = append(findings, secretFindings...)

	return findings, nil
}

func (a *SecurityAnalyzer) analyzePods(pods []Pod) []Finding {
	var findings []Finding

	for _, pod := range pods {
		// Skip system namespaces
		if pod.Namespace == "kube-system" || pod.Namespace == "kube-public" {
			continue
		}

		for _, container := range pod.Containers {
			// Check for privileged containers
			if container.SecurityContext != nil && container.SecurityContext.Privileged {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Privileged container detected",
					Description: fmt.Sprintf("Container %s in pod %s is running in privileged mode, "+
						"which grants full access to the host system", container.Name, pod.Name),
					Evidence:    []string{"securityContext.privileged: true"},
					Remediation: "Remove privileged flag unless absolutely necessary. Use specific capabilities instead.",
					Framework:   "NSA",
					ControlID:   "POD-SEC-001",
					Timestamp:   time.Now(),
				})
			}

			// Check for root user
			if container.SecurityContext == nil ||
				container.SecurityContext.RunAsNonRoot == nil ||
				!*container.SecurityContext.RunAsNonRoot {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Container may run as root",
					Description: fmt.Sprintf("Container %s does not enforce non-root execution, "+
						"which increases the risk of container escape", container.Name),
					Remediation: "Set securityContext.runAsNonRoot: true and specify a non-root runAsUser",
					Framework:   "CIS",
					ControlID:   "5.2.6",
					Timestamp:   time.Now(),
				})
			}

			// Check for read-only root filesystem
			if container.SecurityContext == nil || !container.SecurityContext.ReadOnlyRootFilesystem {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityMedium,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Container without read-only root filesystem",
					Description: fmt.Sprintf("Container %s does not have a read-only root filesystem, "+
						"which could allow attackers to modify files", container.Name),
					Remediation: "Set securityContext.readOnlyRootFilesystem: true",
					Framework:   "CIS",
					ControlID:   "5.2.4",
					Timestamp:   time.Now(),
				})
			}

			// Check for privilege escalation
			if container.SecurityContext != nil &&
				container.SecurityContext.AllowPrivilegeEscalation != nil &&
				*container.SecurityContext.AllowPrivilegeEscalation {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Privilege escalation allowed",
					Description: fmt.Sprintf("Container %s allows privilege escalation, "+
						"which could be exploited for container escape", container.Name),
					Remediation: "Set securityContext.allowPrivilegeEscalation: false",
					Framework:   "CIS",
					ControlID:   "5.2.5",
					Timestamp:   time.Now(),
				})
			}

			// Check for dangerous capabilities
			if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil {
				for _, cap := range container.SecurityContext.Capabilities.Add {
					if isDangerousCapability(cap) {
						findings = append(findings, Finding{
							ID:       uuid.New().String(),
							Severity: SeverityHigh,
							Category: "POD_SECURITY",
							Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
							Title:    "Dangerous capability added",
							Description: fmt.Sprintf("Container %s has dangerous capability %s, "+
								"which could be exploited for privilege escalation", container.Name, cap),
							Evidence:    []string{fmt.Sprintf("capabilities.add: %s", cap)},
							Remediation: "Remove unnecessary capabilities and use specific ones only",
							Framework:   "MITRE",
							ControlID:   "T1611",
							Timestamp:   time.Now(),
						})
					}
				}
			}
		}

		// Check host namespaces
		if pod.SecurityContext != nil {
			if pod.SecurityContext.HostNetwork {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Pod uses host network namespace",
					Description: fmt.Sprintf("Pod %s uses the host network namespace, "+
						"which bypasses network isolation", pod.Name),
					Remediation: "Disable hostNetwork unless absolutely required",
					Framework:   "CIS",
					ControlID:   "5.2.2",
					Timestamp:   time.Now(),
				})
			}

			if pod.SecurityContext.HostPID {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Pod uses host PID namespace",
					Description: fmt.Sprintf("Pod %s uses the host PID namespace, "+
						"which can be used for container escape", pod.Name),
					Remediation: "Disable hostPID - this allows container escape",
					Framework:   "CIS",
					ControlID:   "5.2.3",
					Timestamp:   time.Now(),
				})
			}

			if pod.SecurityContext.HostIPC {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "POD_SECURITY",
					Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
					Title:    "Pod uses host IPC namespace",
					Description: fmt.Sprintf("Pod %s uses the host IPC namespace, "+
						"which can be used for inter-process attacks", pod.Name),
					Remediation: "Disable hostIPC unless absolutely required",
					Framework:   "CIS",
					ControlID:   "5.2.3",
					Timestamp:   time.Now(),
				})
			}
		}
	}

	return findings
}

func (a *SecurityAnalyzer) analyzeRBAC(rbac *RBACState) []Finding {
	var findings []Finding

	// Check for cluster-admin bindings
	for _, crb := range rbac.ClusterRoleBindings {
		if crb.RoleRef.Name == "cluster-admin" {
			for _, subject := range crb.Subjects {
				// Skip system subjects
				if subject.Namespace == "kube-system" {
					continue
				}

				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityCritical,
					Category: "RBAC",
					Resource: ResourceRef{Kind: "ClusterRoleBinding", Name: crb.Name},
					Title:    "Cluster-admin binding detected",
					Description: fmt.Sprintf("%s %s/%s has cluster-admin privileges",
						subject.Kind, subject.Namespace, subject.Name),
					Evidence: []string{
						fmt.Sprintf("ClusterRoleBinding: %s", crb.Name),
						fmt.Sprintf("Subject: %s/%s", subject.Namespace, subject.Name),
					},
					Remediation: "Replace cluster-admin with a more restrictive role",
					Framework:   "CIS",
					ControlID:   "5.1.1",
					Timestamp:   time.Now(),
				})
			}
		}
	}

	// Check for wildcard permissions
	for _, cr := range rbac.ClusterRoles {
		for _, rule := range cr.Rules {
			if containsWildcard(rule.Verbs) || containsWildcard(rule.Resources) {
				findings = append(findings, Finding{
					ID:       uuid.New().String(),
					Severity: SeverityHigh,
					Category: "RBAC",
					Resource: ResourceRef{Kind: "ClusterRole", Name: cr.Name},
					Title:    "Wildcard permissions in ClusterRole",
					Description: fmt.Sprintf("ClusterRole %s contains wildcard permissions "+
						"which may grant excessive access", cr.Name),
					Evidence: []string{
						fmt.Sprintf("Verbs: %v", rule.Verbs),
						fmt.Sprintf("Resources: %v", rule.Resources),
					},
					Remediation: "Replace wildcards with specific verbs and resources",
					Framework:   "CIS",
					ControlID:   "5.1.3",
					Timestamp:   time.Now(),
				})
			}
		}
	}

	// Check service account token automounting
	for _, sa := range rbac.ServiceAccounts {
		if sa.AutomountServiceAccountToken == nil || *sa.AutomountServiceAccountToken {
			// Skip default service accounts
			if sa.Name == "default" {
				continue
			}

			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityMedium,
				Category: "RBAC",
				Resource: ResourceRef{Kind: "ServiceAccount", Name: sa.Name, Namespace: sa.Namespace},
				Title:    "Service account token auto-mounted",
				Description: fmt.Sprintf("ServiceAccount %s/%s has token auto-mounting enabled",
					sa.Namespace, sa.Name),
				Remediation: "Set automountServiceAccountToken: false if not needed",
				Framework:   "CIS",
				ControlID:   "5.1.6",
				Timestamp:   time.Now(),
			})
		}
	}

	return findings
}

func (a *SecurityAnalyzer) analyzeNetworkPolicies(pods []Pod, policies []NetworkPolicy) []Finding {
	var findings []Finding

	// Build a map of namespaces with network policies
	namespacesWithPolicies := make(map[string]bool)
	for _, policy := range policies {
		namespacesWithPolicies[policy.Namespace] = true
	}

	// Check for pods without network policies
	for _, pod := range pods {
		// Skip system namespaces
		if pod.Namespace == "kube-system" || pod.Namespace == "kube-public" {
			continue
		}

		if !namespacesWithPolicies[pod.Namespace] {
			findings = append(findings, Finding{
				ID:       uuid.New().String(),
				Severity: SeverityMedium,
				Category: "NETWORK",
				Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
				Title:    "No network policy in namespace",
				Description: fmt.Sprintf("Namespace %s has no network policies, "+
					"allowing unrestricted pod communication", pod.Namespace),
				Remediation: "Apply a default deny network policy and add specific allow rules",
				Framework:   "NSA",
				ControlID:   "NET-001",
				Timestamp:   time.Now(),
			})
			// Only report once per namespace
			namespacesWithPolicies[pod.Namespace] = true
		}
	}

	return findings
}

func (a *SecurityAnalyzer) analyzeSecrets(secrets []Secret, pods []Pod) []Finding {
	var findings []Finding

	// Check for secrets mounted as environment variables
	for _, pod := range pods {
		for _, container := range pod.Containers {
			for _, env := range container.EnvVars {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					findings = append(findings, Finding{
						ID:       uuid.New().String(),
						Severity: SeverityLow,
						Category: "SECRETS",
						Resource: ResourceRef{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace},
						Title:    "Secret exposed as environment variable",
						Description: fmt.Sprintf("Container %s exposes secret %s as env var %s. "+
							"Environment variables may be logged or exposed",
							container.Name, env.ValueFrom.SecretKeyRef.Name, env.Name),
						Remediation: "Mount secrets as files instead of environment variables",
						Framework:   "CIS",
						ControlID:   "5.4.1",
						Timestamp:   time.Now(),
					})
				}
			}
		}
	}

	return findings
}

func isDangerousCapability(cap string) bool {
	dangerous := map[string]bool{
		"SYS_ADMIN":     true,
		"NET_ADMIN":     true,
		"SYS_PTRACE":    true,
		"SYS_MODULE":    true,
		"DAC_OVERRIDE":  true,
		"SETUID":        true,
		"SETGID":        true,
		"NET_RAW":       true,
		"SYS_RAWIO":     true,
		"MKNOD":         true,
		"SYS_CHROOT":    true,
		"AUDIT_CONTROL": true,
		"AUDIT_WRITE":   true,
		"BLOCK_SUSPEND": true,
		"MAC_ADMIN":     true,
		"MAC_OVERRIDE":  true,
		"SETFCAP":       true,
		"SYSLOG":        true,
		"WAKE_ALARM":    true,
	}
	return dangerous[cap]
}

func containsWildcard(items []string) bool {
	for _, item := range items {
		if item == "*" {
			return true
		}
	}
	return false
}
