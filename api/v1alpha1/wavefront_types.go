/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WavefrontSpec defines the desired state of Wavefront
type WavefrontSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterName is a unique name for the Kubernetes cluster to be identified via a metric tag on Wavefront (Required).
	// +kubebuilder:validation:MinLength:=3
	ClusterName string `json:"clusterName,required"`

	// Wavefront URL for your cluster
	// +kubebuilder:validation:Pattern:=`^https:\/\/.*.wavefront.com`
	WavefrontUrl string `json:"wavefrontUrl,required"`

	// WavefrontTokenSecret is the name of the secret that contains a wavefront API Token.
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([a-z0-9\.\-]*[a-z0-9])?$`
	// +kubebuilder:default:=wavefront-secret
	WavefrontTokenSecret string `json:"wavefrontTokenSecret,omitempty"`

	// DataExport options
	DataExport DataExport `json:"dataExport,omitempty"`

	// DataCollection options
	DataCollection DataCollection `json:"dataCollection,omitempty"`

	//  Allows the operator based Wavefront installation to be run in parallel with a legacy Wavefront (helm or manual) installation. Defaults to false.
	AllowLegacyInstall bool `json:"allowLegacyInstall,omitempty"`

	// ImageRegistry for internal use
	ImageRegistry string `json:"-"`

	// ControllerManagerUID is for internal use of deletion delegation
	ControllerManagerUID string `json:"-"`

	// CanExportData is for internal use
	CanExportData bool `json:"-"`

	// Namespace is for internal use
	Namespace string `json:"-"`
}

type Metrics struct {
	// Enable is whether to enable the metrics. Defaults to true.
	// +kubebuilder:default:=true
	Enable bool `json:"enable,omitempty"`

	// CustomConfig is the custom ConfigMap name for the collector. Leave blank to use defaults.
	CustomConfig string `json:"customConfig,omitempty"`

	// Filters to apply towards all metrics collected by the collector.
	// +kubebuilder:default:={denyList: {kubernetes.sys_container.*, kubernetes.collector.runtime.*, kubernetes.*.network.rx_rate, kubernetes.*.network.rx_errors_rate, kubernetes.*.network.tx_rate, kubernetes.*.network.tx_errors_rate, kubernetes.*.memory.page_faults, kubernetes.*.memory.page_faults_rate, kubernetes.*.memory.major_page_faults, kubernetes.*.memory.major_page_faults_rate, kubernetes.*.filesystem.inodes, kubernetes.*.filesystem.inodes_free, kubernetes.*.ephemeral_storage.request, kubernetes.*.ephemeral_storage.limit}}
	Filters Filters `json:"filters,omitempty"`

	// Tags are a map of key value pairs that are added as point tags on all metrics emitted.
	Tags map[string]string `json:"tags,omitempty"`

	// Default metrics collection interval. Defaults to 60s.
	// +kubebuilder:default:="60s"
	DefaultCollectionInterval string `json:"defaultCollectionInterval,omitempty"`

	// Rules based and Prometheus endpoints auto-discovery. Defaults to true.
	// +kubebuilder:default:=true
	EnableDiscovery bool `json:"enableDiscovery,omitempty"`

	// ClusterCollector is for resource configuration for the cluster collector.
	// +kubebuilder:default:={resources: {requests: {cpu: "200m", memory: "10Mi", ephemeral-storage: "20Mi"}, limits: {cpu: "400m", memory: "512Mi", ephemeral-storage: "1Gi"}}}
	ClusterCollector Collector `json:"clusterCollector,omitempty"`

	// NodeCollector is for resource configuration for the node collector.
	// +kubebuilder:default:={resources: {requests: {cpu: "200m", memory: "10Mi", ephemeral-storage: "20Mi"}, limits: {cpu: "200m", memory: "256Mi", ephemeral-storage: "512Mi"}}}
	NodeCollector Collector `json:"nodeCollector,omitempty"`

	// CollectorConfigName ConfigMap name that is used internally
	CollectorConfigName string `json:"-"`

	// ProxyAddress is for internal use only
	ProxyAddress string `json:"-"`
}

type DataExport struct {
	// External Wavefront WavefrontProxy configuration
	ExternalWavefrontProxy ExternalWavefrontProxy `json:"externalWavefrontProxy,omitempty"`

	// WavefrontProxy configuration options
	WavefrontProxy WavefrontProxy `json:"wavefrontProxy,omitempty"`
}

type ExternalWavefrontProxy struct {
	// Url is the proxy URL that the collector sends metrics to.
	// +kubebuilder:validation:MinLength:=10
	Url string `json:"url,required"`
}

type DataCollection struct {
	// Metrics has resource configuration for node- and cluster-deployed collectors
	Metrics Metrics `json:"metrics,omitempty"`

	//Enable and configure wavefront logging
	Logging Logging `json:"logging,omitempty"`

	// Configure for DaemonSet metric and logging data collectors.  It appends to the default tolerations already set.'
	// +optional
	// +listType=atomic
	Tolerations []Toleration `json:"tolerations,omitempty"`
}

type WavefrontProxy struct {
	// Enable is whether to enable the wavefront proxy. Defaults to true.
	// +kubebuilder:default:=true
	Enable bool `json:"enable,omitempty"`

	// MetricPort is the primary port for Wavefront data format metrics. Defaults to 2878.
	// +kubebuilder:default:=2878
	MetricPort int `json:"metricPort,omitempty"`

	// DeltaCounterPort accumulates 1-minute delta counters on Wavefront data format (usually 50000)
	DeltaCounterPort int `json:"deltaCounterPort,omitempty"`

	// Args is additional Wavefront proxy properties to be passed as command line arguments in the
	// --<property_name> <value> format. Multiple properties can be specified.
	// +kubebuilder:validation:Pattern:=`--.* .*`
	Args string `json:"args,omitempty"`

	// Distributed tracing configuration
	Tracing Tracing `json:"tracing,omitempty"`

	// OpenTelemetry Protocol configuration
	OLTP OLTP `json:"oltp,omitempty"`

	// Histogram distribution configuration
	Histogram Histogram `json:"histogram,omitempty"`

	// Preprocessor is the name of the configmap containing a rules.yaml key with proxy preprocessing rules
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([a-z0-9\.\-]*[a-z0-9])?$`
	Preprocessor string `json:"preprocessor,omitempty"`

	// Resources Compute resources required by the Proxy containers.
	// +kubebuilder:default:={requests: {cpu: "100m", memory: "1Gi", ephemeral-storage: "2Gi"}, limits: {cpu: "1000m", memory: "4Gi", ephemeral-storage: "8Gi"}}
	Resources Resources `json:"resources,omitempty"`

	// Replicas number of replicas
	// +kubebuilder:default:=1
	Replicas int `json:"replicas,omitempty"`

	// HttpProxy configuration
	HttpProxy HttpProxy `json:"httpProxy,omitempty"`

	// ConfigHash is for internal use only
	ConfigHash string `json:"-"`

	// AvailableReplicas is for internal use only
	AvailableReplicas int `json:"-"`
}

type Tracing struct {

	// Wavefront distributed tracing configurations
	Wavefront WavefrontTracing `json:"wavefront,omitempty"`

	// Jaeger distributed tracing configurations
	Jaeger JaegerTracing `json:"jaeger,omitempty"`

	// Zipkin distributed tracing configurations
	Zipkin ZipkinTracing `json:"zipkin,omitempty"`
}

type WavefrontTracing struct {
	// Port for distributed tracing data (usually 30000)
	// +kubebuilder:default:=30000
	Port int `json:"port,omitempty"`

	// SamplingRate Distributed tracing data sampling rate (0 to 1)
	// +kubebuilder:validation:Pattern:=`^(0+\.?|0*\.\d+|0*1(\.0*)?)$`
	SamplingRate string `json:"samplingRate,omitempty"`

	// SamplingDuration When set to greater than 0, spans that exceed this duration will force trace to be sampled (ms)
	SamplingDuration int `json:"samplingDuration,omitempty"`
}

type JaegerTracing struct {
	// Port for Jaeger format distributed tracing data (usually 30001)
	Port int `json:"port,omitempty"`

	// HttpPort for Jaeger Thrift format data (usually 30080)
	HttpPort int `json:"httpPort,omitempty"`

	// GrpcPort for Jaeger GRPC format data (usually 14250)
	GrpcPort int `json:"grpcPort,omitempty"`

	// ApplicationName Custom application name for traces received on Jaeger's Http or Gprc port.
	// +kubebuilder:validation:MinLength:=3
	ApplicationName string `json:"applicationName,omitempty"`
}

type ZipkinTracing struct {
	// Port for Zipkin format distributed tracing data (usually 9411)
	// +kubebuilder:default:=9411
	Port int `json:"port,omitempty"`

	// ApplicationName Custom application name for traces received on Zipkin's port.
	// +kubebuilder:validation:MinLength:=3
	ApplicationName string `json:"applicationName,omitempty"`
}

type Histogram struct {
	// Port for histogram distribution format data (usually 40000)
	Port int `json:"port,omitempty"`

	// MinutePort to accumulate 1-minute based histograms on Wavefront data format (usually 40001)
	MinutePort int `json:"minutePort,omitempty"`

	// HourPort to accumulate 1-hour based histograms on Wavefront data format (usually 40002)
	HourPort int `json:"hourPort,omitempty"`

	// DayPort to accumulate 1-day based histograms on Wavefront data format (usually 40003)
	DayPort int `json:"dayPort,omitempty"`
}

type HttpProxy struct {
	// Name of the secret containing the HttpProxy configuration.
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([a-z0-9\.\-]*[a-z0-9])?$`
	// +kubebuilder:default:=http-proxy-secret
	Secret string `json:"secret,omitempty"`

	// Used internally. Read in from HttpProxy Secret.
	HttpProxyHost      string `json:"-"`
	HttpProxyPort      string `json:"-"`
	HttpProxyUser      string `json:"-"`
	HttpProxyPassword  string `json:"-"`
	UseHttpProxyCAcert bool   `json:"-"`
}

type OLTP struct {
	// GrpcPort for OLTP GRPC format data (usually 4317)
	GrpcPort int `json:"grpcPort,omitempty"`

	// HttpPort for OLTP format data (usually 4318)
	HttpPort int `json:"httpPort,omitempty"`

	// Enable resource attributes on metrics to be included. Defaults to false.
	ResourceAttrsOnMetricsIncluded bool `json:"resourceAttrsOnMetricsIncluded,omitempty"`
}

type Resource struct {
	// CPU is for specifying CPU requirements
	// +kubebuilder:validation:Pattern:=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	// Memory is for specifying Memory requirements
	// +kubebuilder:validation:Pattern:=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	// Memory is for specifying Memory requirements
	// +kubebuilder:validation:Pattern:=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	EphemeralStorage string `json:"ephemeral-storage,omitempty" yaml:"ephemeral-storage,omitempty"`
}

type Toleration struct {
	//
	Key string `json:"key,omitempty" yaml:"key,omitempty"`

	Value string `json:"value,omitempty" yaml:"value,omitempty"`

	// +kubebuilder:validation:Enum=Equal;Exists
	Operator string `json:"operator,omitempty" yaml:"operator,omitempty"`

	// +kubebuilder:validation:Enum=NoSchedule;NoExecute
	Effect string `json:"effect,omitempty" yaml:"effect,omitempty"`
}

type Resources struct {
	// Requests CPU and Memory requirements
	Requests Resource `json:"requests,omitempty" yaml:"requests,omitempty"`

	// Limits CPU and Memory requirements
	Limits Resource `json:"limits,omitempty" yaml:"limits,omitempty"`
}

type Filters struct {
	// List of metric patterns to deny
	// +kubebuilder:default:={kubernetes.sys_container.*, kubernetes.collector.runtime.*, kubernetes.*.network.rx_rate, kubernetes.*.network.rx_errors_rate, kubernetes.*.network.tx_rate, kubernetes.*.network.tx_errors_rate, kubernetes.*.memory.page_faults, kubernetes.*.memory.page_faults_rate, kubernetes.*.memory.major_page_faults, kubernetes.*.memory.major_page_faults_rate, kubernetes.*.filesystem.inodes, kubernetes.*.filesystem.inodes_free, kubernetes.*.ephemeral_storage.request, kubernetes.*.ephemeral_storage.limit}
	DenyList []string `json:"denyList,omitempty"`

	// List of metric patterns to allow
	AllowList []string `json:"allowList,omitempty"`
}

type LogFilters struct {
	// List of log tag patterns to deny
	TagDenyList map[string][]string `json:"tagDenyList,omitempty"`

	// List of log tag patterns to allow
	TagAllowList map[string][]string `json:"tagAllowList,omitempty"`
}

type Collector struct {
	// Resources Compute resources required by the Collector containers.
	Resources Resources `json:"resources,omitempty"`
}

type Logging struct {
	// Enable is whether to enable the wavefront logging. Defaults to false.
	// +kubebuilder:default:=false
	Enable bool `json:"enable,omitempty"`

	// Filters to apply towards all logs collected by wavefront-logging.
	Filters LogFilters `json:"filters,omitempty"`

	// Resources Compute resources required by the logging containers.
	// +kubebuilder:default:={requests: {cpu: "100m", memory: "200Mi", ephemeral-storage: "1Gi"}, limits: {cpu: "1000m", memory: "500Mi", ephemeral-storage: "2Gi"}}
	Resources Resources `json:"resources,omitempty"`

	// Tags are a map of key value pairs that are added to all logging emitted.
	Tags map[string]string `json:"tags,omitempty"`

	// ConfigHash is for internal use only
	ConfigHash string `json:"-"`

	// ProxyAddress is for internal use only
	ProxyAddress string `json:"-"`
}

// WavefrontStatus defines the observed state of Wavefront
type WavefrontStatus struct {
	// Message is a human-readable message indicating details about all the deployment statuses.
	Message string `json:"message,omitempty"`

	// Status is a quick view of all the deployment statuses.
	Status string `json:"status,omitempty"`

	ResourceStatuses []ResourceStatus `json:"resourceStatuses,omitempty"`
}

type ResourceStatus struct {

	// Computed running status. (available / desired )
	Status string `json:"status,omitempty"`

	// Human readable message indicating details of the component.
	Message string `json:"message,omitempty"`

	// Name of the resource
	Name string `json:"name,omitempty"`

	// Resource type (daemonSet or deployment) internal use only
	Type string `json:"-"`

	// Health status internal use only
	Healthy bool `json:"-"`

	// Installing internal use only
	Installing bool `json:"-"`
}

type DaemonSetStatus struct {
	// The total number of nodes that should be running the daemon
	// pod (including nodes correctly running the daemon pod).
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled,omitempty"`

	// numberReady is the number of nodes that should be running the daemon pod and have one
	// or more of the daemon pod running with a Ready Condition.
	NumberReady int32 `json:"numberReady,omitempty"`

	// Computed daemonset status. (available replicas / desired replicas)
	Status string `json:"status,omitempty"`

	// Human readable message indicating details about the daemonset status.
	Message string `json:"message,omitempty"`

	// Name of the daemonset
	DaemonSetName string `json:"daemonSetName,omitempty"`

	// Health status of the daemonset, internal use only
	Healthy bool `json:"-"`
}

type DeploymentStatus struct {
	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	Replicas int32 `json:"replicas,omitempty"`

	// Total number of available pods (ready for at least minReadySeconds) targeted by this deployment.
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Computed deployment status. (available replicas / desired replicas)
	Status string `json:"status,omitempty"`

	// Human readable message indicating details about the deployment status.
	Message string `json:"message,omitempty"`

	// Name of the deployment
	DeploymentName string `json:"deploymentName,omitempty"`

	// Health status of the deployment, internal use only
	Healthy bool `json:"-"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="proxy",type="string",JSONPath=".status.resourceStatuses[?(@.name=='wavefront-proxy')].status"
// +kubebuilder:printcolumn:name="cluster-collector",type="string",JSONPath=".status.resourceStatuses[?(@.name=='wavefront-cluster-collector')].status"
// +kubebuilder:printcolumn:name="node-collector",type="string",JSONPath=".status.resourceStatuses[?(@.name=='wavefront-node-collector')].status"
// +kubebuilder:printcolumn:name="logging",type="string",JSONPath=".status.resourceStatuses[?(@.name=='wavefront-logging')].status"
// +kubebuilder:printcolumn:name="age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="message",type="string",JSONPath=".status.message"
// Wavefront is the Schema for the wavefronts API
type Wavefront struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WavefrontSpec   `json:"spec,omitempty"`
	Status WavefrontStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WavefrontList contains a list of Wavefront
type WavefrontList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Wavefront `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Wavefront{}, &WavefrontList{})
}
