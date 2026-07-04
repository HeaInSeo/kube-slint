// Package dataplane provides a shared, kind-agnostic model of Kubernetes
// manifests (Deployment/StatefulSet/DaemonSet as a single Workload shape,
// Service, ServiceMonitor) and a directory loader, for use by dataplane
// static-analysis profiles (pkg/dataplane/service today; a future
// pkg/dataplane/job profile can reuse this same model).
//
// Deliberately no k8s.io/api or k8s.io/apimachinery dependency: only the
// fields a profile's checks actually need are declared here, decoded via the
// project's existing gopkg.in/yaml.v3 dependency. Unknown/extra manifest
// fields are silently ignored by yaml.v3's default Decode (no
// KnownFields(true)) — the same behavior pkg/kubeutil/rbac.go already
// relies on for its own hand-rolled RBAC structs.
package dataplane

// ObjectMeta mirrors the metadata fields these checks need.
type ObjectMeta struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

// ContainerPort mirrors corev1.ContainerPort's relevant fields.
type ContainerPort struct {
	Name          string `yaml:"name"`
	ContainerPort int    `yaml:"containerPort"`
}

// HTTPGetAction mirrors corev1.HTTPGetAction's relevant fields.
type HTTPGetAction struct {
	Path string `yaml:"path"`
}

// Probe mirrors corev1.Probe. Only HTTPGet is decoded; a tcpSocket/exec/grpc
// probe still decodes successfully (those fields are simply undeclared),
// leaving HTTPGet nil. Checks treat "probe present but HTTPGet nil" as "not
// an HTTP contract to validate the path of" rather than "missing".
type Probe struct {
	HTTPGet *HTTPGetAction `yaml:"httpGet"`
}

// ResourceList is a map of resource name (cpu/memory/...) to quantity
// string. Values are never parsed or compared for v1.5.0 — only key
// presence matters.
type ResourceList map[string]string

// ResourceRequirements mirrors corev1.ResourceRequirements.
type ResourceRequirements struct {
	Requests ResourceList `yaml:"requests"`
	Limits   ResourceList `yaml:"limits"`
}

// Container mirrors corev1.Container's relevant fields.
type Container struct {
	Name           string               `yaml:"name"`
	Ports          []ContainerPort      `yaml:"ports"`
	ReadinessProbe *Probe               `yaml:"readinessProbe"`
	LivenessProbe  *Probe               `yaml:"livenessProbe"`
	Resources      ResourceRequirements `yaml:"resources"`
}

// PodSpec mirrors corev1.PodSpec's relevant fields.
//
// initContainers is deliberately not declared: init containers don't serve
// traffic/probes in the sense these checks care about, and leaving the field
// undeclared means yaml.v3 silently ignores it rather than the loader
// mistaking init containers for workload containers.
type PodSpec struct {
	TerminationGracePeriodSeconds *int64      `yaml:"terminationGracePeriodSeconds"`
	Containers                    []Container `yaml:"containers"`
}

// PodTemplateSpec mirrors corev1.PodTemplateSpec.
type PodTemplateSpec struct {
	Metadata ObjectMeta `yaml:"metadata"`
	Spec     PodSpec    `yaml:"spec"`
}

// LabelSelector mirrors metav1.LabelSelector's relevant fields (matchLabels only).
type LabelSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

// WorkloadSpec mirrors the shared spec shape of Deployment/StatefulSet/DaemonSet.
type WorkloadSpec struct {
	Selector *LabelSelector  `yaml:"selector"`
	Template PodTemplateSpec `yaml:"template"`
}

// Workload represents a Deployment, StatefulSet, or DaemonSet — all three
// share the exact same spec.selector + spec.template.spec shape, so checks
// operate on this single type instead of switching on Kind.
type Workload struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   ObjectMeta   `yaml:"metadata"`
	Spec       WorkloadSpec `yaml:"spec"`
	SourceFile string       `yaml:"-"` // set by the loader
}

// ServicePort mirrors corev1.ServicePort's relevant fields. TargetPort may
// be an int (numeric port) or a string (named container port) in real
// manifests, so it's decoded as `any`.
type ServicePort struct {
	Name       string `yaml:"name"`
	Port       int    `yaml:"port"`
	TargetPort any    `yaml:"targetPort"`
}

// ServiceSpec mirrors corev1.ServiceSpec's relevant fields.
type ServiceSpec struct {
	Selector map[string]string `yaml:"selector"`
	Ports    []ServicePort     `yaml:"ports"`
}

// Service represents a v1 Service manifest.
type Service struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   ObjectMeta  `yaml:"metadata"`
	Spec       ServiceSpec `yaml:"spec"`
	SourceFile string      `yaml:"-"`
}

// ServiceMonitorEndpoint mirrors the Prometheus Operator ServiceMonitor
// CRD's spec.endpoints[] relevant fields (a named-port reference into the
// target Service).
type ServiceMonitorEndpoint struct {
	Port string `yaml:"port"`
}

// ServiceMonitorSpec mirrors the ServiceMonitor CRD's relevant spec fields.
// Note: spec.namespaceSelector is not modeled — matching is same-namespace
// only (documented limitation).
type ServiceMonitorSpec struct {
	Selector  LabelSelector            `yaml:"selector"`
	Endpoints []ServiceMonitorEndpoint `yaml:"endpoints"`
}

// ServiceMonitor represents a monitoring.coreos.com/v1 ServiceMonitor
// manifest. This is a hand-rolled, minimal shape — not the full
// prometheus-operator API type — to avoid adding that module as a dependency
// just to read two fields.
type ServiceMonitor struct {
	APIVersion string             `yaml:"apiVersion"`
	Kind       string             `yaml:"kind"`
	Metadata   ObjectMeta         `yaml:"metadata"`
	Spec       ServiceMonitorSpec `yaml:"spec"`
	SourceFile string             `yaml:"-"`
}

// UnknownObject records a decoded document whose Kind isn't recognized by
// this model (ConfigMap, Namespace, RBAC objects, other CRDs, ...). Not an
// error — just not relevant to workload/Service/ServiceMonitor checks.
type UnknownObject struct {
	APIVersion string
	Kind       string
	Name       string
	SourceFile string
}

// Bundle is the full set of manifest objects loaded from a directory.
type Bundle struct {
	Workloads       []Workload
	Services        []Service
	ServiceMonitors []ServiceMonitor
	Unknown         []UnknownObject
}
