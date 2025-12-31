// Package k8s provides Kubernetes client functionality.
package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeanalysis/kubeanalysis/internal/analyzer"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes client with convenience methods.
type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// ClientConfig holds configuration for the Kubernetes client.
type ClientConfig struct {
	KubeconfigPath string
	Context        string
	InCluster      bool
}

// NewClient creates a new Kubernetes client.
func NewClient(cfg ClientConfig) (*Client, error) {
	var config *rest.Config
	var err error

	if cfg.InCluster {
		// In-cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		// Out-of-cluster configuration
		kubeconfigPath := cfg.KubeconfigPath
		if kubeconfigPath == "" {
			// Try default locations
			if home := os.Getenv("HOME"); home != "" {
				kubeconfigPath = filepath.Join(home, ".kube", "config")
			}
		}

		// Build config from kubeconfig file
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		loadingRules.ExplicitPath = kubeconfigPath

		configOverrides := &clientcmd.ConfigOverrides{}
		if cfg.Context != "" {
			configOverrides.CurrentContext = cfg.Context
		}

		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			configOverrides,
		)

		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// CollectClusterState collects the current state of the cluster.
func (c *Client) CollectClusterState(ctx context.Context, namespaces []string) (*analyzer.ClusterState, error) {
	state := &analyzer.ClusterState{
		CollectedAt: time.Now(),
	}

	// If no namespaces specified, collect from all namespaces
	namespace := ""
	if len(namespaces) == 1 {
		namespace = namespaces[0]
	}

	// Collect nodes
	nodes, err := c.collectNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect nodes: %w", err)
	}
	state.Nodes = nodes

	// Collect pods
	pods, err := c.collectPods(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to collect pods: %w", err)
	}
	// Filter by namespaces if multiple specified
	if len(namespaces) > 1 {
		filtered := make([]analyzer.Pod, 0)
		nsSet := make(map[string]bool)
		for _, ns := range namespaces {
			nsSet[ns] = true
		}
		for _, pod := range pods {
			if nsSet[pod.Namespace] {
				filtered = append(filtered, pod)
			}
		}
		pods = filtered
	}
	state.Pods = pods

	// Collect deployments
	deployments, err := c.collectDeployments(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to collect deployments: %w", err)
	}
	state.Deployments = deployments

	// Collect network policies
	netpols, err := c.collectNetworkPolicies(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to collect network policies: %w", err)
	}
	state.NetworkPolicies = netpols

	// Collect RBAC
	rbac, err := c.collectRBAC(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to collect RBAC: %w", err)
	}
	state.RBAC = rbac

	// Collect namespaces
	nsList, err := c.collectNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect namespaces: %w", err)
	}
	state.Namespaces = nsList

	return state, nil
}

func (c *Client) collectNodes(ctx context.Context) ([]analyzer.Node, error) {
	nodeList, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodes := make([]analyzer.Node, 0, len(nodeList.Items))
	for _, n := range nodeList.Items {
		node := analyzer.Node{
			Name:          n.Name,
			Labels:        n.Labels,
			Unschedulable: n.Spec.Unschedulable,
		}

		// Convert conditions
		for _, cond := range n.Status.Conditions {
			node.Conditions = append(node.Conditions, analyzer.Condition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
				Reason:  cond.Reason,
				Message: cond.Message,
			})

			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				node.Status = "Ready"
			}
		}

		// Convert taints
		for _, t := range n.Spec.Taints {
			node.Taints = append(node.Taints, analyzer.Taint{
				Key:    t.Key,
				Value:  t.Value,
				Effect: string(t.Effect),
			})
		}

		// Convert capacity
		cpu := n.Status.Capacity.Cpu()
		memory := n.Status.Capacity.Memory()
		node.Capacity = analyzer.Resources{
			CPU:    cpu.MilliValue(),
			Memory: memory.Value(),
		}

		// Convert allocatable
		allocCPU := n.Status.Allocatable.Cpu()
		allocMemory := n.Status.Allocatable.Memory()
		node.Allocatable = analyzer.Resources{
			CPU:    allocCPU.MilliValue(),
			Memory: allocMemory.Value(),
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (c *Client) collectPods(ctx context.Context, namespace string) ([]analyzer.Pod, error) {
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods := make([]analyzer.Pod, 0, len(podList.Items))
	for _, p := range podList.Items {
		pod := analyzer.Pod{
			Name:           p.Name,
			Namespace:      p.Namespace,
			Phase:          string(p.Status.Phase),
			NodeName:       p.Spec.NodeName,
			ServiceAccount: p.Spec.ServiceAccountName,
			Labels:         p.Labels,
			Annotations:    p.Annotations,
			QoSClass:       string(p.Status.QOSClass),
		}

		// Calculate total restart count
		for _, cs := range p.Status.ContainerStatuses {
			pod.RestartCount += cs.RestartCount
		}

		// Convert pod security context
		if p.Spec.SecurityContext != nil {
			psc := p.Spec.SecurityContext
			pod.SecurityContext = &analyzer.PodSecurityContext{
				RunAsUser:    psc.RunAsUser,
				RunAsGroup:   psc.RunAsGroup,
				RunAsNonRoot: psc.RunAsNonRoot,
				FSGroup:      psc.FSGroup,
				HostNetwork:  p.Spec.HostNetwork,
				HostPID:      p.Spec.HostPID,
				HostIPC:      p.Spec.HostIPC,
			}
		} else if p.Spec.HostNetwork || p.Spec.HostPID || p.Spec.HostIPC {
			pod.SecurityContext = &analyzer.PodSecurityContext{
				HostNetwork: p.Spec.HostNetwork,
				HostPID:     p.Spec.HostPID,
				HostIPC:     p.Spec.HostIPC,
			}
		}

		// Convert conditions
		for _, cond := range p.Status.Conditions {
			pod.Conditions = append(pod.Conditions, analyzer.Condition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
				Reason:  cond.Reason,
				Message: cond.Message,
			})
		}

		// Convert owner references
		for _, ref := range p.OwnerReferences {
			pod.OwnerReferences = append(pod.OwnerReferences, analyzer.OwnerReference{
				Kind: ref.Kind,
				Name: ref.Name,
				UID:  string(ref.UID),
			})
		}

		// Convert containers
		for i, container := range p.Spec.Containers {
			c := analyzer.Container{
				Name:  container.Name,
				Image: container.Image,
			}

			// Get container status
			if i < len(p.Status.ContainerStatuses) {
				cs := p.Status.ContainerStatuses[i]
				c.ImageID = cs.ImageID
				c.Ready = cs.Ready
				c.RestartCount = cs.RestartCount

				// Convert container state
				if cs.State.Running != nil {
					c.State.Running = &analyzer.ContainerStateRunning{
						StartedAt: cs.State.Running.StartedAt.Time,
					}
				}
				if cs.State.Waiting != nil {
					c.State.Waiting = &analyzer.ContainerStateWaiting{
						Reason:  cs.State.Waiting.Reason,
						Message: cs.State.Waiting.Message,
					}
				}
				if cs.State.Terminated != nil {
					c.State.Terminated = &analyzer.ContainerStateTerminated{
						ExitCode:   cs.State.Terminated.ExitCode,
						Reason:     cs.State.Terminated.Reason,
						Message:    cs.State.Terminated.Message,
						StartedAt:  cs.State.Terminated.StartedAt.Time,
						FinishedAt: cs.State.Terminated.FinishedAt.Time,
					}
				}
			}

			// Convert resources
			if container.Resources.Requests != nil {
				cpu := container.Resources.Requests.Cpu()
				memory := container.Resources.Requests.Memory()
				c.Resources.Requests = analyzer.Resources{
					CPU:    cpu.MilliValue(),
					Memory: memory.Value(),
				}
			}
			if container.Resources.Limits != nil {
				cpu := container.Resources.Limits.Cpu()
				memory := container.Resources.Limits.Memory()
				c.Resources.Limits = analyzer.Resources{
					CPU:    cpu.MilliValue(),
					Memory: memory.Value(),
				}
			}

			// Convert security context
			if container.SecurityContext != nil {
				sc := container.SecurityContext
				c.SecurityContext = &analyzer.ContainerSecurityContext{
					Privileged:               sc.Privileged != nil && *sc.Privileged,
					RunAsUser:                sc.RunAsUser,
					RunAsNonRoot:             sc.RunAsNonRoot,
					ReadOnlyRootFilesystem:   sc.ReadOnlyRootFilesystem != nil && *sc.ReadOnlyRootFilesystem,
					AllowPrivilegeEscalation: sc.AllowPrivilegeEscalation,
				}

				if sc.Capabilities != nil {
					caps := &analyzer.Capabilities{}
					for _, cap := range sc.Capabilities.Add {
						caps.Add = append(caps.Add, string(cap))
					}
					for _, cap := range sc.Capabilities.Drop {
						caps.Drop = append(caps.Drop, string(cap))
					}
					c.SecurityContext.Capabilities = caps
				}
			}

			// Convert ports
			for _, port := range container.Ports {
				c.Ports = append(c.Ports, analyzer.ContainerPort{
					Name:          port.Name,
					ContainerPort: port.ContainerPort,
					Protocol:      string(port.Protocol),
					HostPort:      port.HostPort,
				})
			}

			// Convert volume mounts
			for _, vm := range container.VolumeMounts {
				c.VolumeMounts = append(c.VolumeMounts, analyzer.VolumeMount{
					Name:      vm.Name,
					MountPath: vm.MountPath,
					ReadOnly:  vm.ReadOnly,
					SubPath:   vm.SubPath,
				})
			}

			// Convert env vars (check for secret refs)
			for _, env := range container.Env {
				ev := analyzer.EnvVar{
					Name:  env.Name,
					Value: env.Value,
				}
				if env.ValueFrom != nil {
					ev.ValueFrom = &analyzer.EnvVarSource{}
					if env.ValueFrom.SecretKeyRef != nil {
						ev.ValueFrom.SecretKeyRef = &analyzer.SecretKeySelector{
							Name: env.ValueFrom.SecretKeyRef.Name,
							Key:  env.ValueFrom.SecretKeyRef.Key,
						}
					}
					if env.ValueFrom.ConfigMapKeyRef != nil {
						ev.ValueFrom.ConfigMapKeyRef = &analyzer.ConfigMapKeySelector{
							Name: env.ValueFrom.ConfigMapKeyRef.Name,
							Key:  env.ValueFrom.ConfigMapKeyRef.Key,
						}
					}
				}
				c.EnvVars = append(c.EnvVars, ev)
			}

			pod.Containers = append(pod.Containers, c)
		}

		// Convert init containers
		for _, container := range p.Spec.InitContainers {
			c := analyzer.Container{
				Name:  container.Name,
				Image: container.Image,
			}
			pod.InitContainers = append(pod.InitContainers, c)
		}

		pods = append(pods, pod)
	}

	return pods, nil
}

func (c *Client) collectDeployments(ctx context.Context, namespace string) ([]analyzer.Deployment, error) {
	deployList, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	deployments := make([]analyzer.Deployment, 0, len(deployList.Items))
	for _, d := range deployList.Items {
		deployment := analyzer.Deployment{
			Name:              d.Name,
			Namespace:         d.Namespace,
			Replicas:          *d.Spec.Replicas,
			ReadyReplicas:     d.Status.ReadyReplicas,
			AvailableReplicas: d.Status.AvailableReplicas,
			UpdatedReplicas:   d.Status.UpdatedReplicas,
			Labels:            d.Labels,
		}

		if d.Spec.Selector != nil {
			deployment.Selector = d.Spec.Selector.MatchLabels
		}

		// Convert strategy
		deployment.Strategy = analyzer.DeploymentStrategy{
			Type: string(d.Spec.Strategy.Type),
		}
		if d.Spec.Strategy.RollingUpdate != nil {
			if d.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
				deployment.Strategy.MaxUnavailable = d.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
			}
			if d.Spec.Strategy.RollingUpdate.MaxSurge != nil {
				deployment.Strategy.MaxSurge = d.Spec.Strategy.RollingUpdate.MaxSurge.String()
			}
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

func (c *Client) collectNetworkPolicies(ctx context.Context, namespace string) ([]analyzer.NetworkPolicy, error) {
	netpolList, err := c.clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	netpols := make([]analyzer.NetworkPolicy, 0, len(netpolList.Items))
	for _, np := range netpolList.Items {
		netpol := analyzer.NetworkPolicy{
			Name:        np.Name,
			Namespace:   np.Namespace,
			PodSelector: np.Spec.PodSelector.MatchLabels,
		}

		// Convert policy types
		for _, pt := range np.Spec.PolicyTypes {
			netpol.PolicyTypes = append(netpol.PolicyTypes, string(pt))
		}

		// Convert ingress rules
		for _, rule := range np.Spec.Ingress {
			netpolRule := analyzer.NetworkPolicyRule{}
			for _, port := range rule.Ports {
				netpolPort := analyzer.NetworkPolicyPort{
					Protocol: string(*port.Protocol),
				}
				if port.Port != nil {
					netpolPort.Port = port.Port.IntVal
				}
				netpolRule.Ports = append(netpolRule.Ports, netpolPort)
			}
			for _, from := range rule.From {
				peer := convertNetworkPolicyPeer(from)
				netpolRule.From = append(netpolRule.From, peer)
			}
			netpol.IngressRules = append(netpol.IngressRules, netpolRule)
		}

		// Convert egress rules
		for _, rule := range np.Spec.Egress {
			netpolRule := analyzer.NetworkPolicyRule{}
			for _, port := range rule.Ports {
				netpolPort := analyzer.NetworkPolicyPort{
					Protocol: string(*port.Protocol),
				}
				if port.Port != nil {
					netpolPort.Port = port.Port.IntVal
				}
				netpolRule.Ports = append(netpolRule.Ports, netpolPort)
			}
			for _, to := range rule.To {
				peer := convertNetworkPolicyPeer(to)
				netpolRule.To = append(netpolRule.To, peer)
			}
			netpol.EgressRules = append(netpol.EgressRules, netpolRule)
		}

		netpols = append(netpols, netpol)
	}

	return netpols, nil
}

func convertNetworkPolicyPeer(peer networkingv1.NetworkPolicyPeer) analyzer.NetworkPolicyPeer {
	result := analyzer.NetworkPolicyPeer{}

	if peer.PodSelector != nil {
		result.PodSelector = peer.PodSelector.MatchLabels
	}
	if peer.NamespaceSelector != nil {
		result.NamespaceSelector = peer.NamespaceSelector.MatchLabels
	}
	if peer.IPBlock != nil {
		result.IPBlock = &analyzer.IPBlock{
			CIDR:   peer.IPBlock.CIDR,
			Except: peer.IPBlock.Except,
		}
	}

	return result
}

func (c *Client) collectRBAC(ctx context.Context, namespace string) (*analyzer.RBACState, error) {
	rbacState := &analyzer.RBACState{}

	// Collect ClusterRoles
	crList, err := c.clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ClusterRoles: %w", err)
	}
	for _, cr := range crList.Items {
		clusterRole := analyzer.ClusterRole{
			Name: cr.Name,
		}
		for _, rule := range cr.Rules {
			clusterRole.Rules = append(clusterRole.Rules, convertPolicyRule(rule))
		}
		rbacState.ClusterRoles = append(rbacState.ClusterRoles, clusterRole)
	}

	// Collect ClusterRoleBindings
	crbList, err := c.clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}
	for _, crb := range crbList.Items {
		binding := analyzer.ClusterRoleBinding{
			Name: crb.Name,
			RoleRef: analyzer.RoleRef{
				Kind: crb.RoleRef.Kind,
				Name: crb.RoleRef.Name,
			},
		}
		for _, subj := range crb.Subjects {
			binding.Subjects = append(binding.Subjects, analyzer.Subject{
				Kind:      subj.Kind,
				Name:      subj.Name,
				Namespace: subj.Namespace,
			})
		}
		rbacState.ClusterRoleBindings = append(rbacState.ClusterRoleBindings, binding)
	}

	// Collect Roles (namespace scoped)
	roleList, err := c.clientset.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Roles: %w", err)
	}
	for _, r := range roleList.Items {
		role := analyzer.Role{
			Name:      r.Name,
			Namespace: r.Namespace,
		}
		for _, rule := range r.Rules {
			role.Rules = append(role.Rules, convertPolicyRule(rule))
		}
		rbacState.Roles = append(rbacState.Roles, role)
	}

	// Collect RoleBindings
	rbList, err := c.clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list RoleBindings: %w", err)
	}
	for _, rb := range rbList.Items {
		binding := analyzer.RoleBinding{
			Name:      rb.Name,
			Namespace: rb.Namespace,
			RoleRef: analyzer.RoleRef{
				Kind: rb.RoleRef.Kind,
				Name: rb.RoleRef.Name,
			},
		}
		for _, subj := range rb.Subjects {
			binding.Subjects = append(binding.Subjects, analyzer.Subject{
				Kind:      subj.Kind,
				Name:      subj.Name,
				Namespace: subj.Namespace,
			})
		}
		rbacState.RoleBindings = append(rbacState.RoleBindings, binding)
	}

	// Collect ServiceAccounts
	saList, err := c.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ServiceAccounts: %w", err)
	}
	for _, sa := range saList.Items {
		rbacState.ServiceAccounts = append(rbacState.ServiceAccounts, analyzer.ServiceAccount{
			Name:                         sa.Name,
			Namespace:                    sa.Namespace,
			AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
		})
	}

	return rbacState, nil
}

func convertPolicyRule(rule rbacv1.PolicyRule) analyzer.PolicyRule {
	return analyzer.PolicyRule{
		Verbs:     rule.Verbs,
		APIGroups: rule.APIGroups,
		Resources: rule.Resources,
	}
}

func (c *Client) collectNamespaces(ctx context.Context) ([]analyzer.Namespace, error) {
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaces := make([]analyzer.Namespace, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, analyzer.Namespace{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
			Labels: ns.Labels,
		})
	}

	return namespaces, nil
}

// CollectEvents collects events from the cluster.
func (c *Client) CollectEvents(ctx context.Context, namespace string, since time.Duration) ([]analyzer.Event, error) {
	eventList, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-since)
	events := make([]analyzer.Event, 0)

	for _, e := range eventList.Items {
		// Filter by time
		if e.LastTimestamp.Time.Before(cutoff) && e.EventTime.Time.Before(cutoff) {
			continue
		}

		event := analyzer.Event{
			Type:    e.Type,
			Reason:  e.Reason,
			Message: e.Message,
			Count:   e.Count,
			InvolvedObject: analyzer.ResourceRef{
				Kind:      e.InvolvedObject.Kind,
				Name:      e.InvolvedObject.Name,
				Namespace: e.InvolvedObject.Namespace,
			},
		}

		if !e.FirstTimestamp.IsZero() {
			event.FirstSeen = e.FirstTimestamp.Time
		}
		if !e.LastTimestamp.IsZero() {
			event.LastSeen = e.LastTimestamp.Time
		}

		events = append(events, event)
	}

	return events, nil
}

// GetVersion returns the Kubernetes server version.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.GitVersion, nil
}

// Ping checks connectivity to the cluster.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}
