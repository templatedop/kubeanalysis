// Package analyzer provides Kubernetes resource analyzers.
package analyzer

import (
	"context"
	"time"
)

// Analyzer defines the interface for resource analyzers.
type Analyzer interface {
	// Name returns the name of the analyzer.
	Name() string

	// Analyze performs analysis and returns findings.
	Analyze(ctx context.Context, input AnalyzerInput) ([]Finding, error)
}

// AnalyzerInput provides input data for analyzers.
type AnalyzerInput struct {
	ClusterState *ClusterState
	Metrics      *MetricsData
	Events       []Event
	Options      AnalyzerOptions
}

// AnalyzerOptions configures analyzer behavior.
type AnalyzerOptions struct {
	Namespaces     []string
	IncludeInfo    bool
	CustomRules    []Rule
}

// ClusterState represents the state of a Kubernetes cluster.
type ClusterState struct {
	ClusterID   string
	CollectedAt time.Time
	Nodes       []Node
	Pods        []Pod
	Deployments []Deployment
	Services    []Service
	ConfigMaps  []ConfigMap
	Secrets     []Secret
	Namespaces  []Namespace
	NetworkPolicies []NetworkPolicy
	RBAC        *RBACState
}

// Node represents a Kubernetes node.
type Node struct {
	Name           string
	Status         string
	Conditions     []Condition
	Capacity       Resources
	Allocatable    Resources
	Labels         map[string]string
	Taints         []Taint
	Unschedulable  bool
}

// Pod represents a Kubernetes pod.
type Pod struct {
	Name           string
	Namespace      string
	Status         string
	Phase          string
	Conditions     []Condition
	Containers     []Container
	InitContainers []Container
	RestartCount   int32
	NodeName       string
	QoSClass       string
	ServiceAccount string
	Labels         map[string]string
	Annotations    map[string]string
	SecurityContext *PodSecurityContext
	OwnerReferences []OwnerReference
}

// Container represents a container in a pod.
type Container struct {
	Name            string
	Image           string
	ImageID         string
	Ready           bool
	RestartCount    int32
	State           ContainerState
	Resources       ContainerResources
	SecurityContext *ContainerSecurityContext
	Ports           []ContainerPort
	VolumeMounts    []VolumeMount
	EnvVars         []EnvVar
}

// ContainerState represents the state of a container.
type ContainerState struct {
	Running    *ContainerStateRunning
	Waiting    *ContainerStateWaiting
	Terminated *ContainerStateTerminated
}

// ContainerStateRunning represents a running container.
type ContainerStateRunning struct {
	StartedAt time.Time
}

// ContainerStateWaiting represents a waiting container.
type ContainerStateWaiting struct {
	Reason  string
	Message string
}

// ContainerStateTerminated represents a terminated container.
type ContainerStateTerminated struct {
	ExitCode   int32
	Reason     string
	Message    string
	StartedAt  time.Time
	FinishedAt time.Time
}

// ContainerResources represents container resource requests and limits.
type ContainerResources struct {
	Requests Resources
	Limits   Resources
}

// Resources represents resource quantities.
type Resources struct {
	CPU    int64 // millicores
	Memory int64 // bytes
}

// ContainerSecurityContext represents container security context.
type ContainerSecurityContext struct {
	Privileged               bool
	RunAsUser                *int64
	RunAsNonRoot             *bool
	ReadOnlyRootFilesystem   bool
	AllowPrivilegeEscalation *bool
	Capabilities             *Capabilities
}

// PodSecurityContext represents pod security context.
type PodSecurityContext struct {
	RunAsUser    *int64
	RunAsGroup   *int64
	RunAsNonRoot *bool
	FSGroup      *int64
	HostNetwork  bool
	HostPID      bool
	HostIPC      bool
}

// Capabilities represents Linux capabilities.
type Capabilities struct {
	Add  []string
	Drop []string
}

// ContainerPort represents a container port.
type ContainerPort struct {
	Name          string
	ContainerPort int32
	Protocol      string
	HostPort      int32
}

// VolumeMount represents a volume mount.
type VolumeMount struct {
	Name      string
	MountPath string
	ReadOnly  bool
	SubPath   string
}

// EnvVar represents an environment variable.
type EnvVar struct {
	Name      string
	Value     string
	ValueFrom *EnvVarSource
}

// EnvVarSource represents the source of an env var value.
type EnvVarSource struct {
	SecretKeyRef    *SecretKeySelector
	ConfigMapKeyRef *ConfigMapKeySelector
}

// SecretKeySelector selects a key from a Secret.
type SecretKeySelector struct {
	Name string
	Key  string
}

// ConfigMapKeySelector selects a key from a ConfigMap.
type ConfigMapKeySelector struct {
	Name string
	Key  string
}

// Deployment represents a Kubernetes deployment.
type Deployment struct {
	Name              string
	Namespace         string
	Replicas          int32
	ReadyReplicas     int32
	AvailableReplicas int32
	UpdatedReplicas   int32
	Labels            map[string]string
	Selector          map[string]string
	Strategy          DeploymentStrategy
}

// DeploymentStrategy represents deployment update strategy.
type DeploymentStrategy struct {
	Type          string
	MaxUnavailable string
	MaxSurge      string
}

// Service represents a Kubernetes service.
type Service struct {
	Name       string
	Namespace  string
	Type       string
	ClusterIP  string
	ExternalIP string
	Ports      []ServicePort
	Selector   map[string]string
}

// ServicePort represents a service port.
type ServicePort struct {
	Name       string
	Port       int32
	TargetPort int32
	Protocol   string
	NodePort   int32
}

// ConfigMap represents a Kubernetes ConfigMap.
type ConfigMap struct {
	Name      string
	Namespace string
	Data      map[string]string
}

// Secret represents a Kubernetes Secret.
type Secret struct {
	Name      string
	Namespace string
	Type      string
	Keys      []string // Only keys, not values
}

// Namespace represents a Kubernetes namespace.
type Namespace struct {
	Name   string
	Status string
	Labels map[string]string
}

// NetworkPolicy represents a Kubernetes NetworkPolicy.
type NetworkPolicy struct {
	Name      string
	Namespace string
	PodSelector map[string]string
	PolicyTypes []string
	IngressRules []NetworkPolicyRule
	EgressRules  []NetworkPolicyRule
}

// NetworkPolicyRule represents a network policy rule.
type NetworkPolicyRule struct {
	Ports []NetworkPolicyPort
	From  []NetworkPolicyPeer
	To    []NetworkPolicyPeer
}

// NetworkPolicyPort represents a port in a network policy.
type NetworkPolicyPort struct {
	Protocol string
	Port     int32
}

// NetworkPolicyPeer represents a peer in a network policy.
type NetworkPolicyPeer struct {
	PodSelector       map[string]string
	NamespaceSelector map[string]string
	IPBlock           *IPBlock
}

// IPBlock represents an IP block for network policies.
type IPBlock struct {
	CIDR   string
	Except []string
}

// RBACState represents RBAC state in the cluster.
type RBACState struct {
	Roles               []Role
	ClusterRoles        []ClusterRole
	RoleBindings        []RoleBinding
	ClusterRoleBindings []ClusterRoleBinding
	ServiceAccounts     []ServiceAccount
}

// Role represents a Kubernetes Role.
type Role struct {
	Name      string
	Namespace string
	Rules     []PolicyRule
}

// ClusterRole represents a Kubernetes ClusterRole.
type ClusterRole struct {
	Name  string
	Rules []PolicyRule
}

// PolicyRule represents a rule in a Role or ClusterRole.
type PolicyRule struct {
	Verbs     []string
	APIGroups []string
	Resources []string
}

// RoleBinding represents a Kubernetes RoleBinding.
type RoleBinding struct {
	Name      string
	Namespace string
	RoleRef   RoleRef
	Subjects  []Subject
}

// ClusterRoleBinding represents a Kubernetes ClusterRoleBinding.
type ClusterRoleBinding struct {
	Name     string
	RoleRef  RoleRef
	Subjects []Subject
}

// RoleRef references a Role or ClusterRole.
type RoleRef struct {
	Kind string
	Name string
}

// Subject represents a subject in a binding.
type Subject struct {
	Kind      string
	Name      string
	Namespace string
}

// ServiceAccount represents a Kubernetes ServiceAccount.
type ServiceAccount struct {
	Name                         string
	Namespace                    string
	AutomountServiceAccountToken *bool
}

// Condition represents a resource condition.
type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// Taint represents a node taint.
type Taint struct {
	Key    string
	Value  string
	Effect string
}

// OwnerReference represents an owner reference.
type OwnerReference struct {
	Kind string
	Name string
	UID  string
}

// Event represents a Kubernetes event.
type Event struct {
	ID           string
	Type         string
	Reason       string
	Message      string
	Count        int32
	FirstSeen    time.Time
	LastSeen     time.Time
	InvolvedObject ResourceRef
}

// ResourceRef references a Kubernetes resource.
type ResourceRef struct {
	Kind      string
	Name      string
	Namespace string
}

// MetricsData represents metrics data from Prometheus.
type MetricsData struct {
	NodeMetrics      map[string]NodeMetrics
	PodMetrics       map[string]PodMetrics
	ContainerMetrics map[string]ContainerMetrics
}

// NodeMetrics represents metrics for a node.
type NodeMetrics struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	NetworkIn   float64
	NetworkOut  float64
}

// PodMetrics represents metrics for a pod.
type PodMetrics struct {
	CPUUsage    float64
	MemoryUsage float64
	RestartRate float64
}

// ContainerMetrics represents metrics for a container.
type ContainerMetrics struct {
	CPUUsage     float64
	MemoryUsage  float64
	CPUThrottled float64
}

// Finding represents an analysis finding.
type Finding struct {
	ID          string
	Severity    Severity
	Category    string
	Resource    ResourceRef
	Title       string
	Description string
	Evidence    []string
	Remediation string
	Framework   string
	ControlID   string
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// Severity represents finding severity.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

// Rule represents a custom analysis rule.
type Rule struct {
	ID          string
	Name        string
	Description string
	Severity    Severity
	Category    string
	Condition   string
}
