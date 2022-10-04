package controllers_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileAll(t *testing.T) {
	t.Run("creates proxy, proxy service, collector and collector service", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		spec := defaultWFSpec()
		r, _, _, _ := setupForCreate(spec)
		r.KubernetesManager = stubKM
		versionSent := 0.0
		metricsSent := 0
		r.SendMetrics = func(metrics []metric.Metric) error {
			metricsSent += len(metrics)
			for _, m := range metrics {
				if m.Name == "kubernetes.observability.version" {
					versionSent = m.Value
				}
			}
			return nil
		}

		results, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.Equal(t, ctrl.Result{Requeue: true}, results)

		assert.True(t, stubKM.CollectorServiceAccountContains())
		assert.True(t, stubKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:2878"))
		assert.True(t, stubKM.NodeCollectorDaemonSetContains())
		assert.True(t, stubKM.ClusterCollectorDeploymentContains())
		assert.True(t, stubKM.ProxyServiceContains("port: 2878"))
		assert.True(t, stubKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878"))
		assert.False(t, stubKM.LoggingDaemonSetContains())

		assert.Greater(t, metricsSent, 0, "should have sent metrics")
		assert.Equal(t, 99.9999, versionSent, "should send OperatorVersion")
	})

	t.Run("doesn't create any resources if wavefront spec is invalid", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		invalidWFSpec := defaultWFSpec()
		invalidWFSpec.DataExport.ExternalWavefrontProxy.Url = "http://some_url.com"
		r, _, _, _ := setupForCreate(invalidWFSpec)
		r.KubernetesManager = stubKM
		statusMetricsSent := 0
		r.SendMetrics = func(metrics []metric.Metric) error {
			for _, m := range metrics {
				if strings.HasSuffix(m.Name, ".status") {
					statusMetricsSent += 1
				}
			}
			return nil
		}

		results, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true}, results)

		assert.False(t, stubKM.AppliedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		assert.False(t, stubKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		assert.False(t, stubKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "collector", "wavefront-node-collector"))
		assert.False(t, stubKM.AppliedContains("apps/v1", "Deployment", "wavefront", "collector", "wavefront-cluster-collector"))
		assert.False(t, stubKM.AppliedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		assert.False(t, stubKM.AppliedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))

		assert.Equal(t, 0, statusMetricsSent, "should not have sent status metrics")
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		// TODO: so much setup for only one usage...
		r, wfCR, apiClient, _ := setup("testWavefrontUrl", "updatedToken", "testClusterName")
		r.KubernetesManager = stubKM

		err := apiClient.Delete(context.Background(), wfCR)

		_, err = r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.DeletedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		assert.True(t, stubKM.DeletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		assert.True(t, stubKM.DeletedContains("apps/v1", "DaemonSet", "wavefront", "collector", "wavefront-node-collector"))
		assert.True(t, stubKM.DeletedContains("apps/v1", "Deployment", "wavefront", "collector", "wavefront-cluster-collector"))
		assert.True(t, stubKM.DeletedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		assert.True(t, stubKM.DeletedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))
	})
}

func TestReconcileCollector(t *testing.T) {
	t.Run("does not create configmap if user specified one", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.CustomConfig = "myconfig"
		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		/* Note: User is responsible for applying ConfigMap; we can't test for new ConfigMap "myconfig" */

		/* It DOES call the ApplyResources function with the ConfigMap, but it's filtered out */
		assert.True(t, stubKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		assert.False(t, stubKM.DeletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))

		configMapObject, err := stubKM.GetUnstructuredCollectorConfigMap()
		assert.NoError(t, err)

		assert.False(t, stubKM.ObjectPassesFilter(
			configMapObject,
		))
	})

	t.Run("defaults values for default collector config", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()
		wfSpec := defaultWFSpec()

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("clusterName: testClusterName", "defaultCollectionInterval: 60s", "enableDiscovery: true"))
	})

	t.Run("can change the default collection interval", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.DefaultCollectionInterval = "90s"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("defaultCollectionInterval: 90s"))
	})

	t.Run("can disable discovery", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.EnableDiscovery = false

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("enableDiscovery: false"))
	})

	t.Run("can add custom filters", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.Filters.AllowList = []string{"allowSomeTag", "allowOtherTag"}
		wfSpec.DataCollection.Metrics.Filters.DenyList = []string{"denyAnotherTag", "denyThisTag"}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("metricAllowList:\n        - allowSomeTag\n        - allowOtherTag"))
		assert.True(t, stubKM.CollectorConfigMapContains("metricDenyList:\n        - denyAnotherTag\n        - denyThisTag"))
	})

	t.Run("can add custom tags", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.Tags = map[string]string{"env": "non-production"}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("tags:\n        env: non-production"))
	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "256Mi"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.ClusterCollectorDeploymentContains("memory: 10Mi"))
	})

	t.Run("resources set for node collector", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Limits.Memory = "256Mi"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.NodeCollectorDaemonSetContains("memory: 10Mi"))
	})

	t.Run("no resources set for node and cluster collector", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		r, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		/* DaemonSet wavefront-node-collector */
		assert.True(t, stubKM.NodeCollectorDaemonSetContains("resources:"))
		assert.False(t, stubKM.NodeCollectorDaemonSetContains("limits:", "requests:"))

		/* Deployment wavefront-cluster-collector */
		assert.True(t, stubKM.ClusterCollectorDeploymentContains("resources:"))
		assert.False(t, stubKM.ClusterCollectorDeploymentContains("limits:", "requests:"))
	})

	t.Run("Values from metrics.filters is propagated to default collector configmap", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics = wf.Metrics{
			Enable: true,
			Filters: wf.Filters{
				DenyList:  []string{"first_deny", "second_deny"},
				AllowList: []string{"first_allow", "second_allow"},
			},
		}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		configMap, err := stubKM.GetAppliedYAML(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: wavefront-proxy:2878",
		)
		assert.NoError(t, err)

		configStr, found, err := unstructured.NestedString(configMap.Object, "data", "config.yaml")
		assert.Equal(t, true, found)
		assert.NoError(t, err)

		// TODO: anything to make this more readable?
		var configs map[string]interface{}
		err = yaml.Unmarshal([]byte(configStr), &configs)
		assert.NoError(t, err)
		sinks := configs["sinks"]
		sinkArray := sinks.([]interface{})
		sinkMap := sinkArray[0].(map[string]interface{})
		filters := sinkMap["filters"].(map[string]interface{})
		assert.Equal(t, 2, len(filters["metricDenyList"].([]interface{})))
		assert.Equal(t, 2, len(filters["metricAllowList"].([]interface{})))
	})

	t.Run("Tags can be set for default collector configmap", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.Tags = map[string]string{"key1": "value1", "key2": "value2"}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("key1: value1", "key2: value2"))
	})

	t.Run("Empty tags map should not populate in default collector configmap", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.Tags = map[string]string{}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.False(t, stubKM.CollectorConfigMapContains("tags"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		disabledMetricsSpec := defaultWFSpec()
		disabledMetricsSpec.DataCollection.Metrics.Enable = false

		BehavesLikeItCanBeDisabled(t, disabledMetricsSpec,
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: util.NodeCollectorName,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "collector",
					},
				},
			},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: util.ClusterCollectorName,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "collector",
					},
				},
			},
		)
	})
}

func TestReconcileProxy(t *testing.T) {
	t.Run("creates proxy and proxy service", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		r, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878", "configHash: \"\""))

		assert.True(t, stubKM.ProxyServiceContains("port: 2878"))
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		r, _, _, _ := setup("testWavefrontUrl", "updatedToken", "testClusterName")
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.ProxyDeploymentContains("name: updatedToken", "value: testWavefrontUrl/api/"))
	})

	t.Run("can create proxy with a user defined metric port", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.MetricPort = 1234

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsPortInContainers(t, "pushListenerPorts", *stubKM, 1234)
		containsPortInServicePort(t, 1234, *stubKM)

		assert.True(t, stubKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:1234"))
	})

	t.Run("can create proxy with a user defined delta counter port", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.DeltaCounterPort = 50000
		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsPortInContainers(t, "deltaCounterPorts", *stubKM, 50000)
		containsPortInServicePort(t, 50000, *stubKM)
	})

	t.Run("can create proxy with a user defined Wavefront tracing", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.Port = 30000
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.SamplingRate = ".1"
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.SamplingDuration = 45

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsPortInContainers(t, "traceListenerPorts", *stubKM, 30000)
		containsPortInServicePort(t, 30000, *stubKM)

		containsProxyArg(t, "--traceSamplingRate .1", *stubKM)
		containsProxyArg(t, "--traceSamplingDuration 45", *stubKM)
	})

	t.Run("can create proxy with a user defined Jaeger distributed tracing", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.Port = 30001
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.GrpcPort = 14250
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.HttpPort = 30080
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.ApplicationName = "jaeger"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsPortInContainers(t, "traceJaegerListenerPorts", *stubKM, 30001)
		containsPortInServicePort(t, 30001, *stubKM)

		containsPortInContainers(t, "traceJaegerGrpcListenerPorts", *stubKM, 14250)
		containsPortInServicePort(t, 14250, *stubKM)

		containsPortInContainers(t, "traceJaegerHttpListenerPorts", *stubKM, 30080)
		containsPortInServicePort(t, 30080, *stubKM)

		containsProxyArg(t, "--traceJaegerApplicationName jaeger", *stubKM)
	})

	t.Run("can create proxy with a user defined ZipKin distributed tracing", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.Port = 9411
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.ApplicationName = "zipkin"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsPortInContainers(t, "traceZipkinListenerPorts", *stubKM, 9411)
		containsPortInServicePort(t, 9411, *stubKM)

		containsProxyArg(t, "--traceZipkinApplicationName zipkin", *stubKM)
	})

	t.Run("can create proxy with histogram ports enabled", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Histogram.Port = 40000
		wfSpec.DataExport.WavefrontProxy.Histogram.MinutePort = 40001
		wfSpec.DataExport.WavefrontProxy.Histogram.HourPort = 40002
		wfSpec.DataExport.WavefrontProxy.Histogram.DayPort = 40003

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsPortInContainers(t, "histogramDistListenerPorts", *stubKM, 40000)
		containsPortInServicePort(t, 40000, *stubKM)

		containsPortInContainers(t, "histogramMinuteListenerPorts", *stubKM, 40001)
		containsPortInServicePort(t, 40001, *stubKM)

		containsPortInContainers(t, "histogramHourListenerPorts", *stubKM, 40002)
		containsPortInServicePort(t, 40002, *stubKM)

		containsPortInContainers(t, "histogramDayListenerPorts", *stubKM, 40003)
		containsPortInServicePort(t, 40003, *stubKM)
	})

	t.Run("can create proxy with a user defined proxy args", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Args = "--prefix dev \r\n --customSourceTags mySource"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsProxyArg(t, "--prefix dev", *stubKM)
		containsProxyArg(t, "--customSourceTags mySource", *stubKM)
	})

	t.Run("can create proxy with preprocessor rules", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Preprocessor = "preprocessor-rules"

		// TODO: setupForCreate() finally now only returns reconciler... except inside setup()
		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", *stubKM)

		deployment, err := stubKM.GetAppliedDeployment("proxy", util.ProxyName)
		assert.NoError(t, err)

		volumeMountHasPath(t, deployment, "preprocessor", "/etc/wavefront/preprocessor")
		volumeHasConfigMap(t, deployment, "preprocessor", "preprocessor-rules")
	})

	t.Run("resources set for the proxy", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Resources.Requests.CPU = "100m"
		wfSpec.DataExport.WavefrontProxy.Resources.Requests.Memory = "1Gi"
		wfSpec.DataExport.WavefrontProxy.Resources.Limits.CPU = "1000m"
		wfSpec.DataExport.WavefrontProxy.Resources.Limits.Memory = "4Gi"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		deployment, err := stubKM.GetAppliedDeployment("proxy", util.ProxyName)
		assert.NoError(t, err)

		assert.Equal(t, "1Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		assert.Equal(t, "4Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	})

	t.Run("adjusting proxy replicas", func(t *testing.T) {
		t.Run("changes the number of desired replicas", func(t *testing.T) {
			stubKM := testhelper.NewMockKubernetesManager()

			wfSpec := defaultWFSpec()
			wfSpec.DataExport.WavefrontProxy.Replicas = 2

			r, _, _, _ := setupForCreate(wfSpec)
			r.KubernetesManager = stubKM

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			deployment, err := stubKM.GetAppliedDeployment("proxy", util.ProxyName)
			require.NoError(t, err)

			require.Equal(t, int32(2), *deployment.Spec.Replicas)
		})

		t.Run("defaults to one when no available proxy exists", func(t *testing.T) {
			stubKM := testhelper.NewMockKubernetesManager()

			wfSpec := defaultWFSpec()
			wfSpec.DataExport.WavefrontProxy.Replicas = 1
			wfSpec.DataCollection.Logging.Enable = true

			r, _, _, _ := setupForCreate(wfSpec, &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       util.Deployment,
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.Namespace,
					Name:      util.ProxyName,
				},
				Spec: appsv1.DeploymentSpec{},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 0,
				},
			})
			r.KubernetesManager = stubKM

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			nodeCollector, err := stubKM.GetAppliedDaemonSet("collector", util.NodeCollectorName)
			require.NoError(t, err)

			require.Equal(t, "1", nodeCollector.Spec.Template.Annotations["proxy-available-replicas"])

			clusterCollector, err := stubKM.GetAppliedDeployment("collector", util.ClusterCollectorName)
			require.NoError(t, err)

			require.Equal(t, "1", clusterCollector.Spec.Template.Annotations["proxy-available-replicas"])

			logging, err := stubKM.GetAppliedDaemonSet("logging", util.LoggingName)
			require.NoError(t, err)

			require.Equal(t, "1", logging.Spec.Template.Annotations["proxy-available-replicas"])
		})

		t.Run("defaults to one when no proxies exists", func(t *testing.T) {
			stubKM := testhelper.NewMockKubernetesManager()

			wfSpec := defaultWFSpec()
			wfSpec.DataExport.WavefrontProxy.Replicas = 2
			wfSpec.DataCollection.Logging.Enable = true

			r, _, _, _ := setupForCreate(wfSpec)
			r.KubernetesManager = stubKM

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			nodeCollector, err := stubKM.GetAppliedDaemonSet("collector", util.NodeCollectorName)
			require.NoError(t, err)

			require.Equal(t, "1", nodeCollector.Spec.Template.Annotations["proxy-available-replicas"])

			clusterCollector, err := stubKM.GetAppliedDeployment("collector", util.ClusterCollectorName)
			require.NoError(t, err)

			require.Equal(t, "1", clusterCollector.Spec.Template.Annotations["proxy-available-replicas"])

			logging, err := stubKM.GetAppliedDaemonSet("logging", util.LoggingName)
			require.NoError(t, err)

			require.Equal(t, "1", logging.Spec.Template.Annotations["proxy-available-replicas"])
		})

		t.Run("updates available replicas when based availability", func(t *testing.T) {
			stubKM := testhelper.NewMockKubernetesManager()

			wfSpec := defaultWFSpec()
			wfSpec.DataExport.WavefrontProxy.Replicas = 2
			wfSpec.DataCollection.Logging.Enable = true

			r, _, _, _ := setupForCreate(wfSpec, &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       util.Deployment,
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.Namespace,
					Name:      util.ProxyName,
				},
				Spec: appsv1.DeploymentSpec{},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 2,
				},
			})
			r.KubernetesManager = stubKM

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			nodeCollector, err := stubKM.GetAppliedDaemonSet("collector", util.NodeCollectorName)
			require.NoError(t, err)

			require.Equal(t, "2", nodeCollector.Spec.Template.Annotations["proxy-available-replicas"])

			clusterCollector, err := stubKM.GetAppliedDeployment("collector", util.ClusterCollectorName)
			require.NoError(t, err)

			require.Equal(t, "2", clusterCollector.Spec.Template.Annotations["proxy-available-replicas"])

			logging, err := stubKM.GetAppliedDaemonSet("logging", util.LoggingName)
			require.NoError(t, err)

			require.Equal(t, "2", logging.Spec.Template.Annotations["proxy-available-replicas"])
		})
	})

	t.Run("can create proxy with HTTP configurations", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		var httpProxySecet = &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: util.Namespace,
				UID:       "testUID",
			},
			Data: map[string][]byte{
				"http-url":            []byte("https://myproxyhost_url:8080"),
				"basic-auth-username": []byte("myUser"),
				"basic-auth-password": []byte("myPassword"),
				"tls-root-ca-bundle":  []byte("myCert"),
			},
		}

		r, _, _, _ := setupForCreate(wfSpec, httpProxySecet)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		deployment, err := stubKM.GetAppliedDeployment("proxy", util.ProxyName)
		assert.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyhost_url ", *stubKM)
		containsProxyArg(t, "--proxyPort 8080", *stubKM)
		containsProxyArg(t, "--proxyUser myUser", *stubKM)
		containsProxyArg(t, "--proxyPassword myPassword", *stubKM)

		volumeMountHasPath(t, deployment, "http-proxy-ca", "/tmp/ca")
		volumeHasSecret(t, deployment, "http-proxy-ca", "testHttpProxySecret")

		assert.NotEmpty(t, deployment.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
	})

	t.Run("can create proxy with HTTP configurations only contains http-url", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		var httpProxySecet = &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: util.Namespace,
				UID:       "testUID",
			},
			Data: map[string][]byte{
				"http-url": []byte("https://myproxyhost_url:8080"),
			},
		}

		r, _, _, _ := setupForCreate(wfSpec, httpProxySecet)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyhost_url ", *stubKM)
		containsProxyArg(t, "--proxyPort 8080", *stubKM)
	})

	t.Run("can be disabled", func(t *testing.T) {
		disabledMetricsSpec := defaultWFSpec()
		disabledMetricsSpec.DataExport.WavefrontProxy.Enable = false

		BehavesLikeItCanBeDisabled(t, disabledMetricsSpec,
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: util.ProxyName,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "proxy",
					},
				},
			},
		)
	})
}

func TestReconcileLogging(t *testing.T) {
	t.Run("Create logging if DataCollection.Logging.Enable is set to true", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Logging.Enable = true

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		ds, err := stubKM.GetAppliedDaemonSet("logging", util.LoggingName)
		assert.NoError(t, err)
		assert.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])

		assert.NoError(t, err)
		assert.True(t, stubKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", util.LoggingName))
	})

	t.Run("default resources for logging", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Logging.Enable = true

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.True(t, stubKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", "wavefront-logging"))
		assert.True(t, stubKM.LoggingDaemonSetContains("resources"))
		assert.False(t, stubKM.LoggingDaemonSetContains("limits:", "requests:"))
	})

	t.Run("resources set for logging", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Logging.Enable = true

		wfSpec.DataCollection.Logging.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Logging.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Logging.Resources.Limits.Memory = "256Mi"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.True(t, stubKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", "wavefront-logging"))
		assert.True(t, stubKM.LoggingDaemonSetContains("memory: 10Mi"))
		assert.True(t, stubKM.LoggingDaemonSetContains("cpu: 200m"))
	})

	t.Run("Verify log tag allow list", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Logging.Enable = true
		wfSpec.DataCollection.Logging.Filters = wf.LogFilters{
			TagDenyList:  nil,
			TagAllowList: map[string][]string{"namespace_name": {"kube-sys", "wavefront"}, "pod_name": {"pet-clinic"}},
		}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.True(t, stubKM.LoggingConfigMapContains("key $.namespace_name"))
		assert.True(t, stubKM.LoggingConfigMapContains("key $.pod_name"))
		assert.True(t, stubKM.LoggingConfigMapContains("pattern /(^kube-sys$|^wavefront$)/"))
		assert.True(t, stubKM.LoggingConfigMapContains("pattern /(^pet-clinic$)/"))
	})

	t.Run("Verify log tag deny list", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Logging.Enable = true
		wfSpec.DataCollection.Logging.Filters = wf.LogFilters{
			TagDenyList:  map[string][]string{"namespace_name": {"deny-kube-sys", "deny-wavefront"}, "pod_name": {"deny-pet-clinic"}},
			TagAllowList: nil,
		}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.True(t, stubKM.LoggingConfigMapContains("key $.namespace_name"))
		assert.True(t, stubKM.LoggingConfigMapContains("key $.pod_name"))
		assert.True(t, stubKM.LoggingConfigMapContains("pattern /(^deny-kube-sys$|^deny-wavefront$)/"))
		assert.True(t, stubKM.LoggingConfigMapContains("pattern /(^deny-pet-clinic$)/"))
	})

	t.Run("Verify tags are added to logging pods", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Logging.Enable = true
		wfSpec.DataCollection.Logging.Tags = map[string]string{"key1": "value1", "key2": "value2"}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.True(t, stubKM.LoggingConfigMapContains("key1 value1", "key2 value2"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		disabledMetricsSpec := defaultWFSpec()
		disabledMetricsSpec.DataCollection.Logging.Enable = false

		BehavesLikeItCanBeDisabled(t, disabledMetricsSpec,
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: util.LoggingName,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "logging",
					},
				},
			},
		)
	})
}

func BehavesLikeItCanBeDisabled(t *testing.T, disabledSpec wf.WavefrontSpec, existingResources ...runtime.Object) {
	t.Run("on CR creation", func(t *testing.T) {
		stubKM := testhelper.NewMockKubernetesManager()

		r, _, _, _ := setupForCreate(disabledSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		for _, e := range existingResources {
			objMeta := e.(metav1.ObjectMetaAccessor).GetObjectMeta()
			gvk := e.GetObjectKind().GroupVersionKind()
			assert.Falsef(t, stubKM.AppliedContains(
				gvk.GroupVersion().String(), gvk.Kind,
				objMeta.GetLabels()["app.kubernetes.io/name"],
				objMeta.GetLabels()["app.kubernetes.io/component"],
				objMeta.GetName(),
			), "%s/%s should not have been applied", gvk.Kind, objMeta.GetName())
		}
	})

	t.Run("on CR update", func(t *testing.T) {
		mockKM := testhelper.NewMockKubernetesManager()

		r, _, _, _ := setupForCreate(disabledSpec, existingResources...)
		r.KubernetesManager = mockKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		for _, e := range existingResources {
			objMeta := e.(metav1.ObjectMetaAccessor).GetObjectMeta()
			gvk := e.GetObjectKind().GroupVersionKind()
			assert.True(t, mockKM.DeletedContains(
				gvk.GroupVersion().String(), gvk.Kind,
				objMeta.GetLabels()["app.kubernetes.io/name"],
				objMeta.GetLabels()["app.kubernetes.io/component"],
				objMeta.GetName(),
			), "%s/%s should have been deleted", gvk.Kind, objMeta.GetName())
		}
	})
}

func volumeMountHasPath(t *testing.T, deployment appsv1.Deployment, name, path string) {
	for _, volumeMount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		if volumeMount.Name == name {
			assert.Equal(t, path, volumeMount.MountPath)
			return
		}
	}
	assert.Failf(t, "could not find volume mount", "could not find volume mount named %s on deployment %s", name, deployment.Name)
}

func volumeHasConfigMap(t *testing.T, deployment appsv1.Deployment, name string, configMapName string) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == name {
			assert.Equal(t, configMapName, volume.ConfigMap.Name)
			return
		}
	}
	assert.Failf(t, "could not find volume", "could not find volume named %s on deployment %s", name, deployment.Name)
}

func volumeHasSecret(t *testing.T, deployment appsv1.Deployment, name string, secretName string) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == name {
			assert.Equal(t, secretName, volume.Secret.SecretName)
			return
		}
	}
	assert.Failf(t, "could not find secret", "could not find secret named %s on deployment %s", name, deployment.Name)
}

func containsPortInServicePort(t *testing.T, port int32, stubKM testhelper.MockKubernetesManager) {
	serviceYAMLUnstructured, err := stubKM.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
	assert.NoError(t, err)

	var service v1.Service

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(serviceYAMLUnstructured.Object, &service)
	assert.NoError(t, err)

	for _, servicePort := range service.Spec.Ports {
		if servicePort.Port == port {
			return
		}
	}
	assert.Fail(t, fmt.Sprintf("Did not find the port: %d", port))
}

func containsPortInContainers(t *testing.T, proxyArgName string, stubKM testhelper.MockKubernetesManager, port int32) bool {
	deploymentYAMLUnstructured, err := stubKM.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		util.ProxyName,
	)
	assert.NoError(t, err)

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	assert.NoError(t, err)

	foundPort := false
	for _, containerPort := range deployment.Spec.Template.Spec.Containers[0].Ports {
		if containerPort.ContainerPort == port {
			foundPort = true
			break
		}

		fmt.Printf("%+v", containerPort)
	}
	assert.True(t, foundPort, fmt.Sprintf("Did not find the port: %d", port))

	proxyArgsEnvValue := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	assert.Contains(t, proxyArgsEnvValue, fmt.Sprintf("--%s %d", proxyArgName, port))
	return true
}

func getEnvValueForName(envs []v1.EnvVar, name string) string {
	for _, envVar := range envs {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	return ""
}

func containsProxyArg(t *testing.T, proxyArg string, stubKM testhelper.MockKubernetesManager) {
	deployment, err := stubKM.GetAppliedDeployment("proxy", util.ProxyName)
	assert.NoError(t, err)

	value := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	assert.Contains(t, value, fmt.Sprintf("%s", proxyArg))
}

func defaultWFSpec() wf.WavefrontSpec {
	return wf.WavefrontSpec{
		ClusterName:          "testClusterName",
		WavefrontUrl:         "testWavefrontUrl",
		WavefrontTokenSecret: "testToken",
		DataExport: wf.DataExport{
			WavefrontProxy: wf.WavefrontProxy{
				Enable:     true,
				MetricPort: 2878,
			},
		},
		DataCollection: wf.DataCollection{
			Metrics: wf.Metrics{
				Enable:                    true,
				EnableDiscovery:           true,
				DefaultCollectionInterval: "60s",
			},
		},
		ControllerManagerUID: "",
	}
}

func setupForCreate(spec wf.WavefrontSpec, initObjs ...runtime.Object) (*controllers.WavefrontReconciler, *wf.Wavefront, client.WithWatch, typedappsv1.AppsV1Interface) {
	var wfCR = &wf.Wavefront{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: util.Namespace,
			Name:      "wavefront",
		},
		Spec:   spec,
		Status: wf.WavefrontStatus{},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Service{})
	s.AddKnownTypes(wf.GroupVersion, wfCR)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithScheme(s).WithObjects(wfCR)
	clientBuilder = clientBuilder.WithScheme(s).WithRuntimeObjects(initObjs...)
	apiClient := clientBuilder.Build()

	initObjs = append(initObjs, &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       util.Deployment,
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront-controller-manager",
			Namespace: util.Namespace,
			UID:       "testUID",
		},
		Spec:   appsv1.DeploymentSpec{},
		Status: appsv1.DeploymentStatus{},
	})

	fakesAppsV1 := k8sfake.NewSimpleClientset(initObjs...).AppsV1()

	r := &controllers.WavefrontReconciler{
		OperatorVersion:   "99.99.99",
		Client:            apiClient,
		Scheme:            nil,
		FS:                os.DirFS(controllers.DeployDir),
		KubernetesManager: testhelper.NewMockKubernetesManager(),
		SendMetrics: func(_ []metric.Metric) error {
			return nil
		},
		Appsv1: fakesAppsV1,
	}

	return r, wfCR, apiClient, fakesAppsV1
}

func setup(wavefrontUrl, wavefrontTokenSecret, clusterName string) (*controllers.WavefrontReconciler, *wf.Wavefront, client.WithWatch, typedappsv1.AppsV1Interface) {
	wfSpec := defaultWFSpec()
	wfSpec.WavefrontUrl = wavefrontUrl
	wfSpec.WavefrontTokenSecret = wavefrontTokenSecret
	wfSpec.ClusterName = clusterName
	reconciler, wfCR, apiClient, fakesAppsV1 := setupForCreate(wfSpec)

	return reconciler, wfCR, apiClient, fakesAppsV1
}

func defaultRequest() reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: util.Namespace,
		Name:      "wavefront",
	}}
}
