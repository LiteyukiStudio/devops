package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	ServiceDependencyCheckServiceExists         = "service_exists"
	ServiceDependencyCheckServicePortResolved   = "service_port_resolved"
	ServiceDependencyCheckEndpointReady         = "endpoint_ready"
	ServiceDependencyCheckNetworkPolicyDetected = "network_policy_detected"
)

type ServiceDependencyCheckStatus string

const (
	ServiceDependencyCheckPassed  ServiceDependencyCheckStatus = "passed"
	ServiceDependencyCheckWarning ServiceDependencyCheckStatus = "warning"
	ServiceDependencyCheckFailed  ServiceDependencyCheckStatus = "failed"
)

// ServiceDependencyChecker is intentionally read-only so API and service layers can
// depend on diagnostics without gaining access to Kubernetes mutation methods.
type ServiceDependencyChecker interface {
	CheckServiceDependency(ctx context.Context, options ServiceDependencyCheckOptions) (ServiceDependencyDiagnostic, error)
}

type ServiceDependencyCheckOptions struct {
	SourceNamespace string `json:"sourceNamespace,omitempty"`
	TargetNamespace string `json:"targetNamespace"`
	ServiceName     string `json:"serviceName"`
	PortName        string `json:"portName,omitempty"`
	PortNumber      int32  `json:"portNumber,omitempty"`
}

type ServiceDependencyDiagnostic struct {
	SourceNamespace string                                `json:"sourceNamespace,omitempty"`
	TargetNamespace string                                `json:"targetNamespace"`
	ServiceName     string                                `json:"serviceName"`
	Checks          []ServiceDependencyCheckResult        `json:"checks"`
	Service         ServiceDependencyServiceSummary       `json:"service"`
	Port            ServiceDependencyPortResolution       `json:"port"`
	Endpoints       ServiceDependencyEndpointSummary      `json:"endpoints"`
	NetworkPolicies ServiceDependencyNetworkPolicySummary `json:"networkPolicies"`
}

type ServiceDependencyCheckResult struct {
	Code   string                       `json:"code"`
	Status ServiceDependencyCheckStatus `json:"status"`
}

type ServiceDependencyServiceSummary struct {
	Exists    bool   `json:"exists"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	ClusterIP string `json:"clusterIP,omitempty"`
}

type ServiceDependencyPortResolution struct {
	Resolved        bool                           `json:"resolved"`
	RequestedName   string                         `json:"requestedName,omitempty"`
	RequestedNumber int32                          `json:"requestedNumber,omitempty"`
	Port            *ServiceDependencyPortSummary  `json:"port,omitempty"`
	Available       []ServiceDependencyPortSummary `json:"available"`
}

type ServiceDependencyPortSummary struct {
	Name        string `json:"name,omitempty"`
	Port        int32  `json:"port"`
	TargetPort  string `json:"targetPort,omitempty"`
	Protocol    string `json:"protocol"`
	AppProtocol string `json:"appProtocol,omitempty"`
}

type ServiceDependencyEndpointSummary struct {
	SliceCount         int `json:"sliceCount"`
	MatchingSliceCount int `json:"matchingSliceCount"`
	EndpointCount      int `json:"endpointCount"`
	ReadyEndpointCount int `json:"readyEndpointCount"`
	ReadyAddressCount  int `json:"readyAddressCount"`
}

type ServiceDependencyNetworkPolicySummary struct {
	Detected      bool                                     `json:"detected"`
	PolicyCount   int                                      `json:"policyCount"`
	SourceEgress  []ServiceDependencyNetworkPolicyResource `json:"sourceEgress"`
	TargetIngress []ServiceDependencyNetworkPolicyResource `json:"targetIngress"`
}

type ServiceDependencyNetworkPolicyResource struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Direction   string `json:"direction"`
	PodSelector string `json:"podSelector,omitempty"`
	RuleCount   int    `json:"ruleCount"`
}

var _ ServiceDependencyChecker = (*Client)(nil)

func (c *Client) CheckServiceDependency(ctx context.Context, options ServiceDependencyCheckOptions) (ServiceDependencyDiagnostic, error) {
	options = normalizeServiceDependencyCheckOptions(options)
	if err := validateServiceDependencyCheckOptions(options); err != nil {
		return ServiceDependencyDiagnostic{}, err
	}

	diagnostic := ServiceDependencyDiagnostic{
		SourceNamespace: options.SourceNamespace,
		TargetNamespace: options.TargetNamespace,
		ServiceName:     options.ServiceName,
		Checks:          make([]ServiceDependencyCheckResult, 0, 4),
		Service: ServiceDependencyServiceSummary{
			Namespace: options.TargetNamespace,
			Name:      options.ServiceName,
		},
		Port: ServiceDependencyPortResolution{
			RequestedName:   options.PortName,
			RequestedNumber: options.PortNumber,
			Available:       []ServiceDependencyPortSummary{},
		},
		NetworkPolicies: ServiceDependencyNetworkPolicySummary{
			SourceEgress:  []ServiceDependencyNetworkPolicyResource{},
			TargetIngress: []ServiceDependencyNetworkPolicyResource{},
		},
	}

	service, err := c.client.CoreV1().Services(options.TargetNamespace).Get(ctx, options.ServiceName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		diagnostic.Checks = append(diagnostic.Checks,
			dependencyCheck(ServiceDependencyCheckServiceExists, ServiceDependencyCheckFailed),
			dependencyCheck(ServiceDependencyCheckServicePortResolved, ServiceDependencyCheckFailed),
			dependencyCheck(ServiceDependencyCheckEndpointReady, ServiceDependencyCheckFailed),
		)
	} else if err != nil {
		return diagnostic, fmt.Errorf("get service %s/%s: %w", options.TargetNamespace, options.ServiceName, err)
	} else {
		diagnostic.Service.Exists = true
		diagnostic.Service.Type = string(service.Spec.Type)
		diagnostic.Service.ClusterIP = service.Spec.ClusterIP
		diagnostic.Checks = append(diagnostic.Checks, dependencyCheck(ServiceDependencyCheckServiceExists, ServiceDependencyCheckPassed))

		resolved := resolveServiceDependencyPort(service.Spec.Ports, options)
		diagnostic.Port = resolved
		if resolved.Resolved {
			diagnostic.Checks = append(diagnostic.Checks, dependencyCheck(ServiceDependencyCheckServicePortResolved, ServiceDependencyCheckPassed))
			endpoints, endpointErr := c.serviceDependencyEndpoints(ctx, options.TargetNamespace, service.Name, *resolved.Port)
			if endpointErr != nil {
				return diagnostic, endpointErr
			}
			diagnostic.Endpoints = endpoints
			endpointStatus := ServiceDependencyCheckFailed
			if endpoints.ReadyEndpointCount > 0 {
				endpointStatus = ServiceDependencyCheckPassed
			}
			diagnostic.Checks = append(diagnostic.Checks, dependencyCheck(ServiceDependencyCheckEndpointReady, endpointStatus))
		} else {
			diagnostic.Checks = append(diagnostic.Checks,
				dependencyCheck(ServiceDependencyCheckServicePortResolved, ServiceDependencyCheckFailed),
				dependencyCheck(ServiceDependencyCheckEndpointReady, ServiceDependencyCheckFailed),
			)
		}
	}

	policies, err := c.serviceDependencyNetworkPolicies(ctx, options.SourceNamespace, options.TargetNamespace)
	if err != nil {
		return diagnostic, err
	}
	diagnostic.NetworkPolicies = policies
	policyStatus := ServiceDependencyCheckPassed
	if policies.Detected {
		policyStatus = ServiceDependencyCheckWarning
	}
	diagnostic.Checks = append(diagnostic.Checks, dependencyCheck(ServiceDependencyCheckNetworkPolicyDetected, policyStatus))
	return diagnostic, nil
}

func normalizeServiceDependencyCheckOptions(options ServiceDependencyCheckOptions) ServiceDependencyCheckOptions {
	options.SourceNamespace = strings.TrimSpace(options.SourceNamespace)
	options.TargetNamespace = strings.TrimSpace(options.TargetNamespace)
	options.ServiceName = strings.TrimSpace(options.ServiceName)
	options.PortName = strings.TrimSpace(options.PortName)
	return options
}

func validateServiceDependencyCheckOptions(options ServiceDependencyCheckOptions) error {
	if errs := validation.IsDNS1123Label(options.TargetNamespace); len(errs) > 0 {
		return fmt.Errorf("invalid target namespace %q: %s", options.TargetNamespace, strings.Join(errs, "; "))
	}
	if options.SourceNamespace != "" {
		if errs := validation.IsDNS1123Label(options.SourceNamespace); len(errs) > 0 {
			return fmt.Errorf("invalid source namespace %q: %s", options.SourceNamespace, strings.Join(errs, "; "))
		}
	}
	if errs := validation.IsDNS1123Label(options.ServiceName); len(errs) > 0 {
		return fmt.Errorf("invalid service name %q: %s", options.ServiceName, strings.Join(errs, "; "))
	}
	if options.PortName == "" && options.PortNumber == 0 {
		return fmt.Errorf("service port name or number is required")
	}
	if options.PortName != "" {
		if errs := validation.IsValidPortName(options.PortName); len(errs) > 0 {
			return fmt.Errorf("invalid service port name %q: %s", options.PortName, strings.Join(errs, "; "))
		}
	}
	if options.PortNumber < 0 || options.PortNumber > 65535 {
		return fmt.Errorf("service port number must be between 1 and 65535")
	}
	return nil
}

func resolveServiceDependencyPort(ports []corev1.ServicePort, options ServiceDependencyCheckOptions) ServiceDependencyPortResolution {
	resolution := ServiceDependencyPortResolution{
		RequestedName:   options.PortName,
		RequestedNumber: options.PortNumber,
		Available:       make([]ServiceDependencyPortSummary, 0, len(ports)),
	}
	for _, servicePort := range ports {
		summary := summarizeServiceDependencyPort(servicePort)
		resolution.Available = append(resolution.Available, summary)
		nameMatches := options.PortName == "" || servicePort.Name == options.PortName
		numberMatches := options.PortNumber == 0 || servicePort.Port == options.PortNumber
		if resolution.Resolved || !nameMatches || !numberMatches {
			continue
		}
		resolved := summary
		resolution.Resolved = true
		resolution.Port = &resolved
	}
	return resolution
}

func summarizeServiceDependencyPort(port corev1.ServicePort) ServiceDependencyPortSummary {
	targetPort := port.TargetPort.String()
	if port.TargetPort.IntVal == 0 && port.TargetPort.StrVal == "" {
		targetPort = strconv.FormatInt(int64(port.Port), 10)
	}
	protocol := port.Protocol
	if protocol == "" {
		protocol = corev1.ProtocolTCP
	}
	summary := ServiceDependencyPortSummary{
		Name:       port.Name,
		Port:       port.Port,
		TargetPort: targetPort,
		Protocol:   string(protocol),
	}
	if port.AppProtocol != nil {
		summary.AppProtocol = *port.AppProtocol
	}
	return summary
}

func (c *Client) serviceDependencyEndpoints(ctx context.Context, namespace, serviceName string, port ServiceDependencyPortSummary) (ServiceDependencyEndpointSummary, error) {
	slices, err := c.client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{discoveryv1.LabelServiceName: serviceName}.AsSelector().String(),
	})
	if err != nil {
		return ServiceDependencyEndpointSummary{}, fmt.Errorf("list endpoint slices for service %s/%s: %w", namespace, serviceName, err)
	}

	summary := ServiceDependencyEndpointSummary{SliceCount: len(slices.Items)}
	portNames := map[string]bool{}
	if port.Name != "" {
		portNames[port.Name] = true
	}
	targetPorts := map[int32]bool{}
	if targetPort, parseErr := strconv.ParseInt(port.TargetPort, 10, 32); parseErr == nil && targetPort > 0 {
		targetPorts[int32(targetPort)] = true
	}
	for _, slice := range slices.Items {
		if !endpointSliceMatchesPort(slice.Ports, port.Port, portNames, targetPorts) {
			continue
		}
		summary.MatchingSliceCount++
		for _, endpoint := range slice.Endpoints {
			summary.EndpointCount++
			if endpoint.Conditions.Ready != nil && !*endpoint.Conditions.Ready {
				continue
			}
			summary.ReadyEndpointCount++
			summary.ReadyAddressCount += len(endpoint.Addresses)
		}
	}
	return summary, nil
}

func (c *Client) serviceDependencyNetworkPolicies(ctx context.Context, sourceNamespace, targetNamespace string) (ServiceDependencyNetworkPolicySummary, error) {
	summary := ServiceDependencyNetworkPolicySummary{
		SourceEgress:  []ServiceDependencyNetworkPolicyResource{},
		TargetIngress: []ServiceDependencyNetworkPolicyResource{},
	}
	byNamespace := map[string][]networkingv1.NetworkPolicy{}
	for _, namespace := range uniqueNonEmptyStrings(targetNamespace, sourceNamespace) {
		policies, err := c.client.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return summary, fmt.Errorf("list network policies in namespace %s: %w", namespace, err)
		}
		byNamespace[namespace] = policies.Items
	}

	policyKeys := map[string]struct{}{}
	for _, policy := range byNamespace[targetNamespace] {
		if !networkPolicyIncludesType(policy, networkingv1.PolicyTypeIngress) {
			continue
		}
		summary.TargetIngress = append(summary.TargetIngress, summarizeNetworkPolicy(policy, "ingress", len(policy.Spec.Ingress)))
		policyKeys[policy.Namespace+"/"+policy.Name] = struct{}{}
	}
	if sourceNamespace != "" {
		for _, policy := range byNamespace[sourceNamespace] {
			if !networkPolicyIncludesType(policy, networkingv1.PolicyTypeEgress) {
				continue
			}
			summary.SourceEgress = append(summary.SourceEgress, summarizeNetworkPolicy(policy, "egress", len(policy.Spec.Egress)))
			policyKeys[policy.Namespace+"/"+policy.Name] = struct{}{}
		}
	}
	sortNetworkPolicyResources(summary.TargetIngress)
	sortNetworkPolicyResources(summary.SourceEgress)
	summary.PolicyCount = len(policyKeys)
	summary.Detected = summary.PolicyCount > 0
	return summary, nil
}

func networkPolicyIncludesType(policy networkingv1.NetworkPolicy, policyType networkingv1.PolicyType) bool {
	if len(policy.Spec.PolicyTypes) == 0 {
		if policyType == networkingv1.PolicyTypeIngress {
			return true
		}
		return policyType == networkingv1.PolicyTypeEgress && len(policy.Spec.Egress) > 0
	}
	for _, candidate := range policy.Spec.PolicyTypes {
		if candidate == policyType {
			return true
		}
	}
	return false
}

func summarizeNetworkPolicy(policy networkingv1.NetworkPolicy, direction string, ruleCount int) ServiceDependencyNetworkPolicyResource {
	selector := ""
	if parsed, err := metav1.LabelSelectorAsSelector(&policy.Spec.PodSelector); err == nil {
		selector = parsed.String()
	}
	return ServiceDependencyNetworkPolicyResource{
		Namespace:   policy.Namespace,
		Name:        policy.Name,
		Direction:   direction,
		PodSelector: selector,
		RuleCount:   ruleCount,
	}
}

func sortNetworkPolicyResources(resources []ServiceDependencyNetworkPolicyResource) {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Namespace == resources[j].Namespace {
			return resources[i].Name < resources[j].Name
		}
		return resources[i].Namespace < resources[j].Namespace
	})
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func dependencyCheck(code string, status ServiceDependencyCheckStatus) ServiceDependencyCheckResult {
	return ServiceDependencyCheckResult{Code: code, Status: status}
}
