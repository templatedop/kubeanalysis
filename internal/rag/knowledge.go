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
}
