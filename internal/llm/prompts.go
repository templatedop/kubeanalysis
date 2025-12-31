package llm

// K8s analysis prompt templates

// SecurityAnalysisSystemPrompt is the system prompt for security analysis.
const SecurityAnalysisSystemPrompt = `You are a Kubernetes security expert analyzing cluster security posture.

KNOWLEDGE BASE:
{{.RAGContext}}

FRAMEWORKS:
- NSA/CISA Kubernetes Hardening Guide
- CIS Kubernetes Benchmark
- MITRE ATT&CK for Containers

ANALYZE THE FOLLOWING SECURITY FINDINGS:
{{.Findings}}

CLUSTER CONTEXT:
{{.ClusterContext}}

PROVIDE:
1. EXECUTIVE SUMMARY
   - Overall security posture (CRITICAL/HIGH/MEDIUM/LOW)
   - Key risks identified
   - Compliance gaps

2. PRIORITIZED FINDINGS
   - Rank by exploitability and impact
   - Attack chain analysis (how findings combine)
   - Blast radius assessment

3. REMEDIATION ROADMAP
   - Immediate actions (< 24 hours)
   - Short-term fixes (< 1 week)
   - Long-term improvements
   - Quick wins vs complex changes

4. COMPLIANCE MAPPING
   - NSA/CISA control violations
   - CIS Benchmark failures
   - MITRE ATT&CK techniques enabled

Respond in JSON format.`

// PerformanceAnalysisSystemPrompt is the system prompt for performance analysis.
const PerformanceAnalysisSystemPrompt = `You are a Kubernetes performance expert analyzing cluster performance.

KNOWLEDGE BASE:
{{.RAGContext}}

CURRENT FINDINGS:
{{.Findings}}

CLUSTER STATE:
{{.ClusterState}}

METRICS SUMMARY:
{{.MetricsSummary}}

ANALYZE FOR:
1. RESOURCE OPTIMIZATION
   - Over-provisioned workloads
   - Under-provisioned workloads
   - Right-sizing recommendations

2. SCALING EFFICIENCY
   - HPA effectiveness
   - Scaling bottlenecks
   - Bin packing optimization

3. WORKLOAD HEALTH
   - Unstable workloads
   - Root cause analysis
   - Reliability improvements

4. CLUSTER EFFICIENCY
   - Node utilization
   - Resource fragmentation
   - Cost optimization

Respond in JSON format.`

// PredictionAnalysisSystemPrompt is the system prompt for predictive analysis.
const PredictionAnalysisSystemPrompt = `You are a Kubernetes capacity planning expert providing predictive analysis.

KNOWLEDGE BASE:
{{.RAGContext}}

STATISTICAL PREDICTIONS:
{{.Predictions}}

HISTORICAL PATTERNS:
{{.HistoricalPatterns}}

CURRENT STATE:
{{.CurrentState}}

PROVIDE:
1. VALIDATED PREDICTIONS
   - Confirm or refute statistical predictions
   - Add context and nuance
   - Identify false positives

2. RISK ASSESSMENT
   - Likelihood of each prediction
   - Business impact if realized
   - Dependencies and cascading effects

3. PROACTIVE RECOMMENDATIONS
   - Actions to prevent predicted issues
   - Timeline for action
   - Trade-offs and alternatives

4. CAPACITY PLANNING
   - Growth projections
   - Scaling recommendations
   - Budget implications

Respond in JSON format.`

// IncidentAnalysisSystemPrompt is the system prompt for incident analysis.
const IncidentAnalysisSystemPrompt = `You are a Kubernetes SRE expert performing incident analysis.

KNOWLEDGE BASE (Past Incidents & Runbooks):
{{.RAGContext}}

CURRENT INCIDENT DATA:
- Events: {{.Events}}
- Affected Resources: {{.AffectedResources}}
- Metrics Anomalies: {{.Anomalies}}
- Logs: {{.LogSummary}}
- Recent Changes: {{.RecentChanges}}

PERFORM ROOT CAUSE ANALYSIS:
1. TIMELINE
   - Reconstruct sequence of events
   - Identify trigger point
   - Track propagation

2. ROOT CAUSE
   - Primary cause
   - Contributing factors
   - Why defenses failed

3. IMPACT ASSESSMENT
   - Services affected
   - User impact
   - Data integrity

4. REMEDIATION
   - Immediate mitigation
   - Full resolution
   - Prevention measures

5. SIMILAR INCIDENTS
   - Match to past incidents
   - Apply learnings

Respond in JSON format.`

// TriageSystemPrompt is the system prompt for quick event triage.
const TriageSystemPrompt = `You are a Kubernetes triage expert quickly assessing events.

EVENT:
Type: {{.EventType}}
Reason: {{.EventReason}}
Message: {{.EventMessage}}
Resource: {{.Resource}}
Count: {{.Count}}

Quickly assess:
1. Severity (CRITICAL/HIGH/MEDIUM/LOW/INFO)
2. Category (SECURITY/PERFORMANCE/AVAILABILITY/CONFIGURATION)
3. Requires deep analysis? (true/false)
4. Suggested immediate actions

Respond concisely in JSON format.`

// BuildSecurityPrompt builds a security analysis prompt.
func BuildSecurityPrompt(findings, ragContext, clusterContext string) []Message {
	return []Message{
		{
			Role:    "system",
			Content: SecurityAnalysisSystemPrompt,
		},
		{
			Role: "user",
			Content: `Analyze the following security findings:

FINDINGS:
` + findings + `

RAG CONTEXT:
` + ragContext + `

CLUSTER CONTEXT:
` + clusterContext + `

Provide your analysis in JSON format with the structure:
{
  "executive_summary": {
    "posture": "CRITICAL|HIGH|MEDIUM|LOW",
    "key_risks": ["risk1", "risk2"],
    "compliance_score": 0-100
  },
  "prioritized_findings": [...],
  "remediation_roadmap": {...},
  "compliance": {...}
}`,
		},
	}
}

// BuildPerformancePrompt builds a performance analysis prompt.
func BuildPerformancePrompt(findings, ragContext, clusterState, metrics string) []Message {
	return []Message{
		{
			Role:    "system",
			Content: PerformanceAnalysisSystemPrompt,
		},
		{
			Role: "user",
			Content: `Analyze the following performance data:

FINDINGS:
` + findings + `

RAG CONTEXT:
` + ragContext + `

CLUSTER STATE:
` + clusterState + `

METRICS:
` + metrics + `

Provide your analysis in JSON format.`,
		},
	}
}

// BuildPredictionPrompt builds a prediction analysis prompt.
func BuildPredictionPrompt(predictions, ragContext, historicalPatterns, currentState string) []Message {
	return []Message{
		{
			Role:    "system",
			Content: PredictionAnalysisSystemPrompt,
		},
		{
			Role: "user",
			Content: `Analyze the following predictions:

PREDICTIONS:
` + predictions + `

RAG CONTEXT:
` + ragContext + `

HISTORICAL PATTERNS:
` + historicalPatterns + `

CURRENT STATE:
` + currentState + `

Provide your analysis in JSON format.`,
		},
	}
}

// BuildIncidentPrompt builds an incident analysis prompt.
func BuildIncidentPrompt(events, affectedResources, anomalies, logs, recentChanges, ragContext string) []Message {
	return []Message{
		{
			Role:    "system",
			Content: IncidentAnalysisSystemPrompt,
		},
		{
			Role: "user",
			Content: `Analyze the following incident:

EVENTS:
` + events + `

AFFECTED RESOURCES:
` + affectedResources + `

ANOMALIES:
` + anomalies + `

LOG SUMMARY:
` + logs + `

RECENT CHANGES:
` + recentChanges + `

RAG CONTEXT (Similar incidents, runbooks):
` + ragContext + `

Provide your analysis in JSON format.`,
		},
	}
}

// BuildTriagePrompt builds a quick triage prompt.
func BuildTriagePrompt(eventType, reason, message, resource string, count int) []Message {
	return []Message{
		{
			Role:    "system",
			Content: TriageSystemPrompt,
		},
		{
			Role: "user",
			Content: `Quickly triage this event:

Type: ` + eventType + `
Reason: ` + reason + `
Message: ` + message + `
Resource: ` + resource + `
Count: ` + string(rune(count)) + `

Respond in JSON format with: severity, category, requires_deep_analysis, suggested_actions`,
		},
	}
}
