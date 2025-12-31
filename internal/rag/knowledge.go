package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// K8sKnowledgeBase provides Kubernetes-specific knowledge management.
type K8sKnowledgeBase struct {
	engine *Engine
}

// NewK8sKnowledgeBase creates a new Kubernetes knowledge base.
func NewK8sKnowledgeBase(engine *Engine) *K8sKnowledgeBase {
	return &K8sKnowledgeBase{engine: engine}
}

// IngestSecurityGuidelines ingests security guidelines (NSA/CISA, CIS Benchmarks).
func (kb *K8sKnowledgeBase) IngestSecurityGuidelines(ctx context.Context, content string, source string) error {
	doc := Document{
		ID:      uuid.New().String(),
		Content: content,
		Metadata: map[string]string{
			"category":  string(CategorySecurity),
			"source":    source,
			"type":      "security_guideline",
			"namespace": "security",
		},
		Source:    source,
		CreatedAt: time.Now(),
	}

	return kb.engine.IngestDocument(ctx, doc)
}

// IngestBestPractices ingests Kubernetes best practices documentation.
func (kb *K8sKnowledgeBase) IngestBestPractices(ctx context.Context, content string, source string) error {
	doc := Document{
		ID:      uuid.New().String(),
		Content: content,
		Metadata: map[string]string{
			"category":  string(CategoryBestPractice),
			"source":    source,
			"type":      "best_practice",
			"namespace": "best_practices",
		},
		Source:    source,
		CreatedAt: time.Now(),
	}

	return kb.engine.IngestDocument(ctx, doc)
}

// IngestIncident ingests incident reports and post-mortems.
func (kb *K8sKnowledgeBase) IngestIncident(ctx context.Context, incident IncidentReport) error {
	content := fmt.Sprintf(`
Incident ID: %s
Severity: %s
Date: %s
Title: %s

Summary:
%s

Root Cause:
%s

Resolution:
%s

Lessons Learned:
%s

Related Resources: %v
Tags: %v
`,
		incident.ID,
		incident.Severity,
		incident.Date.Format(time.RFC3339),
		incident.Title,
		incident.Summary,
		incident.RootCause,
		incident.Resolution,
		incident.LessonsLearned,
		incident.RelatedResources,
		incident.Tags,
	)

	doc := Document{
		ID:      incident.ID,
		Content: content,
		Metadata: map[string]string{
			"category":  string(CategoryIncident),
			"source":    "incident_report",
			"type":      "incident",
			"severity":  incident.Severity,
			"namespace": "incidents",
		},
		Source:    "incident_report",
		CreatedAt: incident.Date,
	}

	return kb.engine.IngestDocument(ctx, doc)
}

// IngestRunbook ingests operational runbooks.
func (kb *K8sKnowledgeBase) IngestRunbook(ctx context.Context, runbook Runbook) error {
	stepsContent := ""
	for i, step := range runbook.Steps {
		stepsContent += fmt.Sprintf("\nStep %d: %s\n%s\n", i+1, step.Title, step.Description)
		if step.Command != "" {
			stepsContent += fmt.Sprintf("Command: %s\n", step.Command)
		}
		if step.ExpectedResult != "" {
			stepsContent += fmt.Sprintf("Expected Result: %s\n", step.ExpectedResult)
		}
	}

	content := fmt.Sprintf(`
Runbook: %s

Description:
%s

Triggers:
%v

Steps:
%s

Escalation:
Contact: %s
Conditions: %v
`,
		runbook.Name,
		runbook.Description,
		runbook.Triggers,
		stepsContent,
		runbook.Escalation.Contact,
		runbook.Escalation.Conditions,
	)

	doc := Document{
		ID:      runbook.ID,
		Content: content,
		Metadata: map[string]string{
			"category":  string(CategoryRunbook),
			"source":    "runbook",
			"type":      "runbook",
			"name":      runbook.Name,
			"namespace": "runbooks",
		},
		Source:    "runbook",
		CreatedAt: time.Now(),
	}

	return kb.engine.IngestDocument(ctx, doc)
}

// QuerySecurity queries for security-related knowledge.
func (kb *K8sKnowledgeBase) QuerySecurity(ctx context.Context, query string, topK int) (*RAGContext, error) {
	filters := map[string]string{
		"namespace": "security",
	}
	return kb.engine.QueryWithFilters(ctx, query, filters, topK)
}

// QueryPerformance queries for performance-related knowledge.
func (kb *K8sKnowledgeBase) QueryPerformance(ctx context.Context, query string, topK int) (*RAGContext, error) {
	filters := map[string]string{
		"category": string(CategoryPerformance),
	}
	return kb.engine.QueryWithFilters(ctx, query, filters, topK)
}

// QueryIncidents queries for incident-related knowledge.
func (kb *K8sKnowledgeBase) QueryIncidents(ctx context.Context, query string, topK int) (*RAGContext, error) {
	filters := map[string]string{
		"namespace": "incidents",
	}
	return kb.engine.QueryWithFilters(ctx, query, filters, topK)
}

// QueryRunbooks queries for runbooks.
func (kb *K8sKnowledgeBase) QueryRunbooks(ctx context.Context, query string, topK int) (*RAGContext, error) {
	filters := map[string]string{
		"namespace": "runbooks",
	}
	return kb.engine.QueryWithFilters(ctx, query, filters, topK)
}

// QueryAll queries across all knowledge categories.
func (kb *K8sKnowledgeBase) QueryAll(ctx context.Context, query string, topK int) (*RAGContext, error) {
	return kb.engine.Query(ctx, query, topK)
}

// IncidentReport represents an incident report for the knowledge base.
type IncidentReport struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Severity         string    `json:"severity"`
	Date             time.Time `json:"date"`
	Summary          string    `json:"summary"`
	RootCause        string    `json:"root_cause"`
	Resolution       string    `json:"resolution"`
	LessonsLearned   string    `json:"lessons_learned"`
	RelatedResources []string  `json:"related_resources"`
	Tags             []string  `json:"tags"`
}

// Runbook represents an operational runbook.
type Runbook struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Triggers    []string      `json:"triggers"`
	Steps       []RunbookStep `json:"steps"`
	Escalation  Escalation    `json:"escalation"`
}

// RunbookStep represents a step in a runbook.
type RunbookStep struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Command        string `json:"command,omitempty"`
	ExpectedResult string `json:"expected_result,omitempty"`
}

// Escalation represents escalation information.
type Escalation struct {
	Contact    string   `json:"contact"`
	Conditions []string `json:"conditions"`
}

// PrebuiltKnowledge contains prebuilt knowledge for common Kubernetes scenarios.
var PrebuiltKnowledge = map[string]string{
	// NSA/CISA Kubernetes Hardening Guide
	"nsa_cisa_pod_security": `
NSA/CISA Kubernetes Hardening Guide - Pod Security

1. Non-root Containers (Critical)
   - All containers should run as non-root users
   - Use runAsUser with non-zero UID
   - Enable runAsNonRoot: true to enforce at runtime
   - NSA Control: Prevents privilege escalation attacks

2. Immutable Container Filesystems (High)
   - Use readOnlyRootFilesystem: true
   - Use emptyDir for temporary storage
   - Prevents attackers from modifying binaries
   - NSA Control: Reduces attack surface

3. Container Privileges (Critical)
   - Never use privileged: true
   - Drop ALL capabilities, add only what's needed
   - Disable allowPrivilegeEscalation
   - NSA Control: Prevents container escape

4. Host Namespace Isolation (Critical)
   - Avoid hostNetwork, hostPID, hostIPC
   - These break container isolation
   - NSA Control: Maintains security boundaries

5. Seccomp and AppArmor (High)
   - Apply RuntimeDefault or custom seccomp profiles
   - Use AppArmor profiles for additional protection
   - NSA Control: Restricts syscall attack surface
`,

	"nsa_cisa_network_security": `
NSA/CISA Kubernetes Hardening Guide - Network Security

1. Network Policies (Critical)
   - Implement default-deny ingress policies
   - Implement default-deny egress policies
   - Allow only required communication paths
   - Use namespace and pod selectors carefully
   - NSA Control: Zero trust network architecture

2. Control Plane Protection (Critical)
   - Restrict API server access with network policies
   - Use private endpoints when possible
   - Enable API server audit logging
   - NSA Control: Protects cluster management

3. Encryption in Transit (High)
   - Enable TLS for all cluster communications
   - Use mutual TLS (mTLS) with service mesh
   - Encrypt etcd communications
   - NSA Control: Prevents data interception

4. External Network Access (High)
   - Control egress to external services
   - Use network policies for DNS access
   - Consider egress gateways
   - NSA Control: Prevents data exfiltration

5. Ingress Security (Medium)
   - Use TLS termination at ingress
   - Implement rate limiting
   - Consider WAF integration
   - NSA Control: Protects external endpoints
`,

	"nsa_cisa_rbac": `
NSA/CISA Kubernetes Hardening Guide - RBAC

1. Least Privilege Principle (Critical)
   - Grant minimum required permissions
   - Use namespaced Roles over ClusterRoles
   - Avoid wildcards (*) in rules
   - Regular permission audits
   - NSA Control: Limits blast radius

2. Service Account Security (Critical)
   - Create dedicated service accounts per app
   - Disable automountServiceAccountToken when not needed
   - Never use default service account for apps
   - NSA Control: Prevents token abuse

3. Cluster Admin Restriction (Critical)
   - Minimize cluster-admin bindings
   - Never bind cluster-admin to service accounts
   - Use time-limited admin access (OIDC)
   - NSA Control: Protects cluster integrity

4. User Authentication (High)
   - Use external identity providers (OIDC)
   - Disable anonymous authentication
   - Enable authentication audit logging
   - NSA Control: Ensures accountability

5. API Server Access (High)
   - Restrict API access by source IP
   - Use RBAC for all operations
   - Monitor and alert on privilege escalation
   - NSA Control: Prevents unauthorized access
`,

	"nsa_cisa_secrets": `
NSA/CISA Kubernetes Hardening Guide - Secrets Management

1. Encryption at Rest (Critical)
   - Enable etcd encryption for secrets
   - Use KMS provider for key management
   - Rotate encryption keys periodically
   - NSA Control: Protects stored secrets

2. Secret Access Control (Critical)
   - Use RBAC to control secret access
   - Limit secret access to specific namespaces
   - Audit secret access regularly
   - NSA Control: Prevents unauthorized access

3. External Secret Management (High)
   - Consider HashiCorp Vault or AWS Secrets Manager
   - Use operators for secret injection
   - Avoid mounting secrets as environment variables
   - NSA Control: Centralizes secret management

4. Secret Rotation (High)
   - Implement automatic secret rotation
   - Use short-lived credentials when possible
   - Monitor for stale secrets
   - NSA Control: Limits exposure window

5. Image Registry Credentials (Medium)
   - Use imagePullSecrets for private registries
   - Scope credentials to specific namespaces
   - Use short-lived tokens where possible
   - NSA Control: Protects container supply chain
`,

	// CIS Kubernetes Benchmark
	"cis_control_plane": `
CIS Kubernetes Benchmark - Control Plane Security

1.1 API Server Configuration
   - Enable --anonymous-auth=false (1.1.1)
   - Use --authorization-mode with RBAC (1.1.2)
   - Enable --audit-log-path (1.1.3)
   - Set --kubelet-certificate-authority (1.1.4)

1.2 etcd Configuration
   - Use --cert-file and --key-file (1.2.1)
   - Enable --client-cert-auth=true (1.2.2)
   - Set --peer-cert-file and --peer-key-file (1.2.3)
   - Enable --peer-client-cert-auth=true (1.2.4)

1.3 Controller Manager
   - Disable --profiling (1.3.1)
   - Set --use-service-account-credentials=true (1.3.2)
   - Configure --service-account-private-key-file (1.3.3)

1.4 Scheduler
   - Disable --profiling (1.4.1)
   - Bind to localhost only (1.4.2)
`,

	"cis_worker_nodes": `
CIS Kubernetes Benchmark - Worker Node Security

2.1 Kubelet Configuration
   - Disable --anonymous-auth (2.1.1)
   - Set --authorization-mode to Webhook (2.1.2)
   - Enable --client-ca-file (2.1.3)
   - Set --read-only-port=0 (2.1.4)
   - Enable --streaming-connection-idle-timeout (2.1.5)
   - Enable --protect-kernel-defaults=true (2.1.6)
   - Set --make-iptables-util-chains=true (2.1.7)
   - Configure --tls-cert-file and --tls-private-key-file (2.1.8)
   - Set --rotate-certificates=true (2.1.9)

2.2 Kubelet Service Files
   - Verify kubelet.service ownership (2.2.1)
   - Check kubelet.conf permissions (2.2.2)
   - Validate kubeconfig file permissions (2.2.3)

2.3 Container Runtime
   - Use supported container runtime (2.3.1)
   - Configure seccomp profiles (2.3.2)
   - Enable AppArmor/SELinux (2.3.3)
`,

	"cis_pod_security_policies": `
CIS Kubernetes Benchmark - Pod Security Standards

5.1 Pod Security Admission
   - Enable PodSecurity admission controller (5.1.1)
   - Set baseline or restricted security level (5.1.2)
   - Apply policies to all namespaces (5.1.3)

5.2 Pod Security Controls
   - Do not admit privileged containers (5.2.1)
   - Do not admit containers with hostPID (5.2.2)
   - Do not admit containers with hostIPC (5.2.3)
   - Do not admit containers with hostNetwork (5.2.4)
   - Minimize allowedCapabilities (5.2.5)
   - Do not admit containers wanting to share host process ID (5.2.6)
   - Do not admit containers wanting to share host IPC (5.2.7)
   - Do not admit containers wanting to share host network (5.2.8)
   - Restrict allowed volume types (5.2.9)
   - Require read-only root filesystem (5.2.10)
   - Restrict privilege escalation (5.2.11)
   - Restrict root user (5.2.12)

5.3 Network Policies
   - Use default deny ingress policy (5.3.1)
   - Use default deny egress policy (5.3.2)
`,

	"cis_rbac_service_accounts": `
CIS Kubernetes Benchmark - RBAC and Service Accounts

5.1 RBAC Configuration
   - Limit cluster-admin usage (5.1.1)
   - Minimize access to secrets (5.1.2)
   - Minimize wildcard use in Roles (5.1.3)
   - Minimize access to create pods (5.1.4)
   - Ensure default service account not used (5.1.5)
   - Ensure service account tokens only mounted when necessary (5.1.6)
   - Avoid use of system:masters group (5.1.7)
   - Limit use of bind/escalate/impersonate (5.1.8)
   - Minimize PersistentVolume creation access (5.1.9)
   - Minimize storage object creation access (5.1.10)

5.2 Service Accounts
   - Prefer dedicated service accounts (5.2.1)
   - Set automountServiceAccountToken: false where not needed (5.2.2)
   - Use projected service account tokens (5.2.3)
`,

	"pod_security_basics": `
Pod Security Best Practices:

1. Run containers as non-root
   - Set runAsNonRoot: true
   - Set runAsUser to a non-zero value

2. Use read-only root filesystem
   - Set readOnlyRootFilesystem: true

3. Drop all capabilities
   - Set capabilities.drop: ["ALL"]
   - Only add specific capabilities needed

4. Disable privilege escalation
   - Set allowPrivilegeEscalation: false

5. Avoid privileged containers
   - Never set privileged: true unless absolutely necessary

6. Use seccomp profiles
   - Apply seccomp profiles to restrict syscalls
`,

	"rbac_best_practices": `
RBAC Best Practices:

1. Principle of Least Privilege
   - Grant only the minimum permissions required
   - Use Roles instead of ClusterRoles when possible

2. Avoid cluster-admin
   - Never use cluster-admin for applications
   - Audit cluster-admin bindings regularly

3. Namespace isolation
   - Use namespaced Roles and RoleBindings
   - Create separate service accounts per application

4. Service Account tokens
   - Disable automounting when not needed
   - Set automountServiceAccountToken: false

5. Regular audits
   - Review permissions periodically
   - Remove unused bindings
`,

	"network_policy_basics": `
Network Policy Best Practices:

1. Default Deny
   - Start with a default deny-all ingress policy
   - Add explicit allow rules as needed

2. Namespace isolation
   - Use namespace selectors to isolate workloads

3. Label-based selection
   - Use consistent labeling for pod selection

4. Egress policies
   - Control outbound traffic to external services
   - Allow only necessary DNS and external endpoints

5. Audit and test
   - Regularly audit network policies
   - Test policies in staging before production
`,

	"resource_management": `
Resource Management Best Practices:

1. Always set requests and limits
   - CPU and memory requests for scheduling
   - Limits to prevent resource exhaustion

2. Right-sizing
   - Base requests on actual usage
   - Set limits 2-3x of typical usage

3. Quality of Service
   - Guaranteed: requests == limits
   - Burstable: requests < limits
   - BestEffort: no requests or limits (avoid)

4. Vertical Pod Autoscaler
   - Use VPA recommendations for sizing

5. Resource quotas
   - Set namespace quotas to prevent runaway usage
`,

	"health_checks": `
Health Check Best Practices:

1. Readiness Probes
   - Check if pod can accept traffic
   - Use for gradual rollouts

2. Liveness Probes
   - Detect and restart hung processes
   - Avoid aggressive settings (can cause loops)

3. Startup Probes
   - For slow-starting applications
   - Prevents premature liveness failures

4. Probe configuration
   - initialDelaySeconds: allow time for startup
   - periodSeconds: 10-30 seconds typical
   - failureThreshold: 3 for liveness, 1 for readiness

5. Endpoint design
   - Dedicated health endpoints
   - Check real dependencies
`,

	// Troubleshooting Runbooks
	"runbook_pod_crash_loop": `
Runbook: Pod CrashLoopBackOff

Symptoms:
- Pod repeatedly crashes and restarts
- Status shows CrashLoopBackOff
- Restart count keeps increasing

Diagnostic Steps:
1. Check pod events:
   kubectl describe pod <pod-name> -n <namespace>

2. Check container logs:
   kubectl logs <pod-name> -c <container> -n <namespace> --previous

3. Check resource limits:
   kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A5 resources

4. Check for OOMKilled:
   kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A2 lastState

Common Causes:
- Application startup failure
- Missing configuration or secrets
- Insufficient memory (OOMKilled)
- Failed health checks
- Missing dependencies

Remediation:
1. Fix application errors in code
2. Add missing ConfigMaps/Secrets
3. Increase memory limits if OOMKilled
4. Adjust probe settings if health checks fail
5. Verify all dependencies are available
`,

	"runbook_oom_killed": `
Runbook: OOMKilled Containers

Symptoms:
- Container terminated with reason: OOMKilled
- Exit code 137
- Application killed without proper shutdown

Diagnostic Steps:
1. Check container state:
   kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[*].lastState}'

2. Check memory usage over time:
   kubectl top pod <pod-name> -n <namespace>

3. Review memory limits:
   kubectl get pod <pod-name> -o yaml | grep -A5 resources

4. Check for memory leaks:
   Review application memory patterns

Common Causes:
- Memory limit too low
- Memory leak in application
- Sudden spike in workload
- Large data processing without streaming

Remediation:
1. Increase memory limits (consider 1.5-2x current limit)
2. Investigate and fix memory leaks
3. Implement memory-efficient data processing
4. Add memory monitoring alerts
5. Consider using Vertical Pod Autoscaler
`,

	"runbook_pending_pods": `
Runbook: Pods Stuck in Pending State

Symptoms:
- Pod status shows Pending
- No node assigned to pod
- Pod events show FailedScheduling

Diagnostic Steps:
1. Check pod events:
   kubectl describe pod <pod-name> -n <namespace>

2. Check node resources:
   kubectl describe nodes | grep -A5 "Allocated resources"

3. Check for taints:
   kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints

4. Check PVC status if using volumes:
   kubectl get pvc -n <namespace>

Common Causes:
- Insufficient cluster resources (CPU, memory)
- Node selector/affinity doesn't match
- Taints without tolerations
- PVC not bound
- Pod quota exceeded

Remediation:
1. Scale cluster if resources insufficient
2. Fix nodeSelector/affinity rules
3. Add appropriate tolerations
4. Ensure PVC is bound and available
5. Increase namespace resource quota
`,

	"runbook_image_pull": `
Runbook: Image Pull Failures

Symptoms:
- Pod status shows ImagePullBackOff or ErrImagePull
- Container cannot start
- Events show pull errors

Diagnostic Steps:
1. Check pod events:
   kubectl describe pod <pod-name> -n <namespace>

2. Verify image name and tag:
   kubectl get pod <pod-name> -o jsonpath='{.spec.containers[*].image}'

3. Check image pull secrets:
   kubectl get pod <pod-name> -o jsonpath='{.spec.imagePullSecrets}'

4. Try pulling manually:
   docker pull <image-name>

Common Causes:
- Image name typo
- Image tag doesn't exist
- Private registry without credentials
- Network connectivity issues
- Registry rate limiting

Remediation:
1. Verify correct image name and tag
2. Create and attach imagePullSecret
3. Check network connectivity to registry
4. Use image pull-through cache
5. Authenticate with registry (DockerHub rate limits)
`,

	"runbook_high_cpu": `
Runbook: High CPU Usage

Symptoms:
- Node CPU utilization > 85%
- Pod performance degradation
- CPU throttling observed

Diagnostic Steps:
1. Check node CPU usage:
   kubectl top nodes

2. Check pod CPU usage:
   kubectl top pods -n <namespace> --sort-by=cpu

3. Check for CPU throttling:
   kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.status.containerStatuses[*].ready}'

4. Review CPU requests/limits:
   kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].resources}{"\n"}{end}'

Common Causes:
- CPU limits too low (throttling)
- Application inefficiency
- Sudden traffic spike
- Runaway process or infinite loop
- Too many pods on single node

Remediation:
1. Increase CPU limits if throttling
2. Optimize application code
3. Scale horizontally (more replicas)
4. Implement Horizontal Pod Autoscaler
5. Add more nodes to cluster
`,

	"runbook_high_memory": `
Runbook: High Memory Usage

Symptoms:
- Node memory utilization > 80%
- MemoryPressure condition on node
- Pod evictions occurring

Diagnostic Steps:
1. Check node memory:
   kubectl top nodes

2. Check pod memory usage:
   kubectl top pods -n <namespace> --sort-by=memory

3. Check node conditions:
   kubectl describe node <node-name> | grep -A5 Conditions

4. Check eviction events:
   kubectl get events --field-selector reason=Evicted

Common Causes:
- Memory requests under-provisioned
- Memory leaks in applications
- Too many pods scheduled
- Large cache without limits
- Improper resource quota

Remediation:
1. Right-size memory requests based on actual usage
2. Fix memory leaks in applications
3. Implement memory limits on all containers
4. Add more nodes or larger nodes
5. Review and optimize caching strategies
`,

	// Incident Response Patterns
	"incident_cluster_outage": `
Incident Pattern: Cluster Outage

Indicators:
- Multiple nodes NotReady
- API server unavailable
- etcd cluster unhealthy
- Control plane pods crashing

Immediate Actions:
1. Check control plane components
2. Verify etcd cluster health
3. Check node status and kubelet logs
4. Review recent changes (deployments, config)

Investigation:
- kubectl get nodes
- kubectl get pods -n kube-system
- Check cloud provider status
- Review audit logs

Recovery:
1. Restore control plane components
2. Repair etcd if needed
3. Drain and repair unhealthy nodes
4. Verify workload health after recovery

Prevention:
- High availability control plane
- Regular etcd backups
- Node health monitoring
- Staged rollouts for changes
`,

	"incident_security_breach": `
Incident Pattern: Security Breach

Indicators:
- Unexpected pods or containers
- Unusual network traffic patterns
- Modified RBAC permissions
- Cryptocurrency mining detected
- Unusual API access patterns

Immediate Actions:
1. Isolate affected workloads
2. Capture forensic evidence (logs, configs)
3. Block malicious network traffic
4. Rotate compromised credentials

Investigation:
- kubectl get pods --all-namespaces (look for unexpected)
- Review audit logs for unauthorized access
- Check for privileged containers
- Analyze network policy violations
- Review service account usage

Recovery:
1. Remove malicious workloads
2. Patch vulnerable components
3. Rotate all potentially compromised secrets
4. Apply network policies
5. Enable/enhance audit logging

Prevention:
- Implement Pod Security Standards
- Enable network policies
- Regular security scanning
- Principle of least privilege RBAC
- Image scanning in CI/CD
`,

	"incident_data_loss": `
Incident Pattern: Data Loss

Indicators:
- PersistentVolume unavailable
- Database corruption errors
- Backup failures
- Accidental deletion events

Immediate Actions:
1. Stop writes to affected storage
2. Preserve any remaining data
3. Identify last known good backup
4. Assess scope of data loss

Investigation:
- Check PV/PVC status
- Review deletion events
- Check storage backend logs
- Verify backup integrity
- Review change history

Recovery:
1. Restore from backup if available
2. Rebuild from replica if applicable
3. Recover using storage snapshots
4. Document data that cannot be recovered

Prevention:
- Regular automated backups
- Backup verification testing
- Storage redundancy (replicas)
- PVC protection policies
- Deletion protection policies
`,
}
