package controllers_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper/wftest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileAll(t *testing.T) {
	t.Run("does not create other services until the proxy is running", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(), wftest.Proxy(wftest.WithReplicas(0, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.CollectorServiceAccountContains())
		require.False(t, mockKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:2878"))
		require.False(t, mockKM.NodeCollectorDaemonSetContains())
		require.False(t, mockKM.ClusterCollectorDeploymentContains())
		require.False(t, mockKM.LoggingDaemonSetContains())
		require.True(t, mockKM.ProxyServiceContains("port: 2878"))
		require.True(t, mockKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878"))

		require.Equal(t, 0, len(mockSender.SentMetrics), "should not have sent metrics")
	})

	t.Run("creates other components after the proxy is running", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(), wftest.Proxy(wftest.WithReplicas(1, 1)))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.True(t, mockKM.CollectorServiceAccountContains())
		require.True(t, mockKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:2878"))
		require.True(t, mockKM.NodeCollectorDaemonSetContains())
		require.True(t, mockKM.ClusterCollectorDeploymentContains())
		require.True(t, mockKM.LoggingDaemonSetContains())

		require.Greater(t, len(mockSender.SentMetrics), 0, "should not have sent metrics")
		require.Equal(t, 99.9999, VersionSent(mockSender), "should send OperatorVersion")
	})

	t.Run("transitions status when sub-components change (even if overall health is still unhealthy)", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Status.Status = health.Unhealthy
		})
		r, _ := componentScenario(wfCR)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		var reconciledWFCR wf.Wavefront

		require.NoError(t, r.Client.Get(
			context.Background(),
			util.ObjKey(wfCR.Namespace, wfCR.Name),
			&reconciledWFCR,
		))

		require.Contains(t, reconciledWFCR.Status.Status, health.Unhealthy)
		require.Contains(t, reconciledWFCR.Status.ResourceStatuses, wf.ResourceStatus{Status: "Running (1/1)", Name: "wavefront-proxy"})
	})

	t.Run("doesn't create any resources if wavefront spec is invalid", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = true
			w.Spec.DataExport.ExternalWavefrontProxy.Url = "http://some_url.com"
		}))
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		results, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, results)

		require.False(t, mockKM.AppliedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		require.False(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		require.False(t, mockKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "collector", "wavefront-node-collector"))
		require.False(t, mockKM.AppliedContains("apps/v1", "Deployment", "wavefront", "collector", "wavefront-cluster-collector"))
		require.False(t, mockKM.AppliedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		require.False(t, mockKM.AppliedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))

		require.Equal(t, 0, StatusMetricsSent(mockSender), "should not have sent status metrics")
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		r, mockKM := emptyScenario(nil)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))
		_ = r.MetricConnection.Connect("http://example.com")

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.DeletedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		require.True(t, mockKM.DeletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		require.True(t, mockKM.DeletedContains("apps/v1", "DaemonSet", "wavefront", "node-collector", "wavefront-node-collector"))
		require.True(t, mockKM.DeletedContains("apps/v1", "Deployment", "wavefront", "cluster-collector", "wavefront-cluster-collector"))
		require.True(t, mockKM.DeletedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		require.True(t, mockKM.DeletedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))

		require.Equal(t, 1, mockSender.Closes)
	})

	t.Run("Defaults Custom Registry", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR())
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("image: projects.registry.vmware.com/tanzu_observability/kubernetes-collector"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("image: projects.registry.vmware.com/tanzu_observability/kubernetes-collector"))
		require.True(t, mockKM.LoggingDaemonSetContains("image: projects.registry.vmware.com/tanzu_observability/kubernetes-operator-fluentd"))
		require.True(t, mockKM.ProxyDeploymentContains("image: projects.registry.vmware.com/tanzu_observability/proxy"))
	})

	t.Run("Can Configure Custom Registry", func(t *testing.T) {
		r, mockKM := componentScenario(
			wftest.CR(),
			wftest.Operator(func(d *appsv1.Deployment) {
				d.Spec.Template.Spec.Containers[0].Image = "docker.io/kubernetes-operator:latest"
			}),
		)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("image: docker.io/kubernetes-collector"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("image: docker.io/kubernetes-collector"))
		require.True(t, mockKM.LoggingDaemonSetContains("image: docker.io/kubernetes-operator-fluentd"))
		require.True(t, mockKM.ProxyDeploymentContains("image: docker.io/proxy"))
	})

	t.Run("Child components inherits controller's namespace", func(t *testing.T) {
		wfCR := wftest.CR(func(w *wf.Wavefront) {
			w.Namespace = "customNamespace"
		})
		r, mockKM := componentScenario(wfCR)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		request := defaultRequest()
		request.Namespace = wfCR.Namespace
		_, err := r.Reconcile(context.Background(), request)
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("namespace: customNamespace"))
		require.True(t, mockKM.ClusterCollectorDeploymentContains("namespace: customNamespace"))
		require.True(t, mockKM.LoggingDaemonSetContains("namespace: customNamespace"))
		require.True(t, mockKM.ProxyDeploymentContains("namespace: customNamespace"))
	})
}

func TestReconcileCollector(t *testing.T) {
	t.Run("does not create configmap if user specified one", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.CustomConfig = "myconfig"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		/* Note: User is responsible for applying ConfigMap; we can't test for new ConfigMap "myconfig" */

		/* It DOES call the ApplyResources function with the ConfigMap, but it's filtered out */
		require.True(t, mockKM.AppliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		require.False(t, mockKM.DeletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))

		configMapObject, err := mockKM.GetUnstructuredCollectorConfigMap()
		require.NoError(t, err)

		require.False(t, mockKM.ObjectPassesFilter(
			configMapObject,
		))
	})

	t.Run("can change the default collection interval", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.DefaultCollectionInterval = "90s"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("defaultCollectionInterval: 90s"))
	})

	t.Run("can disable discovery", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.EnableDiscovery = false
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("enableDiscovery: false"))
	})

	t.Run("can add custom filters", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Filters.AllowList = []string{"allowSomeTag", "allowOtherTag"}
			w.Spec.DataCollection.Metrics.Filters.DenyList = []string{"denyAnotherTag", "denyThisTag"}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("metricAllowList:\n        - allowSomeTag\n        - allowOtherTag"))
		require.True(t, mockKM.CollectorConfigMapContains("metricDenyList:\n        - denyAnotherTag\n        - denyThisTag"))
	})

	t.Run("can add custom tags", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Tags = map[string]string{"env": "non-production"}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("tags:\n        env: non-production"))
	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "200m"
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "10Mi"
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "256Mi"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())

		require.NoError(t, err)

		require.True(t, mockKM.ClusterCollectorDeploymentContains("memory: 10Mi"))
	})

	t.Run("resources set for node collector", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "200m"
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Requests.Memory = "10Mi"
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
			w.Spec.DataCollection.Metrics.NodeCollector.Resources.Limits.Memory = "256Mi"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.NodeCollectorDaemonSetContains("memory: 10Mi"))
	})

	t.Run("no resources set for node and cluster collector", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR())

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		/* DaemonSet wavefront-node-collector */
		require.True(t, mockKM.NodeCollectorDaemonSetContains("resources:"))
		require.False(t, mockKM.NodeCollectorDaemonSetContains("limits:", "requests:"))

		/* Deployment wavefront-cluster-collector */
		require.True(t, mockKM.ClusterCollectorDeploymentContains("resources:"))
		require.False(t, mockKM.ClusterCollectorDeploymentContains("limits:", "requests:"))
	})

	t.Run("Values from metrics.filters is propagated to default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics = wf.Metrics{
				Enable: true,
				Filters: wf.Filters{
					DenyList:  []string{"first_deny", "second_deny"},
					AllowList: []string{"first_allow", "second_allow"},
				},
			}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		configMap, err := mockKM.GetAppliedYAML(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: wavefront-proxy:2878",
		)
		require.NoError(t, err)

		configStr, found, err := unstructured.NestedString(configMap.Object, "data", "config.yaml")
		require.Equal(t, true, found)
		require.NoError(t, err)

		// TODO: anything to make this more readable?
		var configs map[string]interface{}
		err = yaml.Unmarshal([]byte(configStr), &configs)
		require.NoError(t, err)
		sinks := configs["sinks"]
		sinkArray := sinks.([]interface{})
		sinkMap := sinkArray[0].(map[string]interface{})
		filters := sinkMap["filters"].(map[string]interface{})
		require.Equal(t, 2, len(filters["metricDenyList"].([]interface{})))
		require.Equal(t, 2, len(filters["metricAllowList"].([]interface{})))
	})

	t.Run("Tags can be set for default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Tags = map[string]string{"key1": "value1", "key2": "value2"}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.CollectorConfigMapContains("key1: value1", "key2: value2"))
	})

	t.Run("Empty tags map should not populate in default collector configmap", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Tags = map[string]string{}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.False(t, mockKM.CollectorConfigMapContains("tags"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		CanBeDisabled(t,
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataCollection.Metrics.Enable = false
			}),
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.NodeCollectorName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "node-collector",
					},
				},
			},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.ClusterCollectorName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "cluster-collector",
					},
				},
			},
		)
	})
}

func TestReconcileProxy(t *testing.T) {
	t.Run("creates proxy and proxy service", func(t *testing.T) {

		r, mockKM := emptyScenario(wftest.CR())

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878", "configHash: \"\""))

		require.True(t, mockKM.ProxyServiceContains("port: 2878"))
	})

	t.Run("does not create proxy when it is configured to use an external proxy", func(t *testing.T) {
		r, mockKM := emptyScenario(
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Enable = false
				w.Spec.DataExport.ExternalWavefrontProxy.Url = "https://example.com"
			}),
			wftest.Proxy(wftest.WithReplicas(0, 1)),
		)
		mockSender := &testhelper.MockSender{}
		r.MetricConnection = metric.NewConnection(testhelper.StubSenderFactory(mockSender, nil))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		r.MetricConnection.Flush()

		require.False(t, mockKM.ProxyDeploymentContains())
		require.False(t, mockKM.ProxyServiceContains())
		require.Greater(t, len(mockSender.SentMetrics), 0)
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.WavefrontTokenSecret = "updatedToken"
			w.Spec.WavefrontUrl = "updatedUrl"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		require.True(t, mockKM.ProxyDeploymentContains(
			"name: updatedToken",
			"value: updatedUrl/api/",
		))
	})

	t.Run("can create proxy with a user defined metric port", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.MetricPort = 1234
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "pushListenerPorts", *mockKM, 1234)
		containsPortInServicePort(t, 1234, *mockKM)

		require.True(t, mockKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:1234"))
	})

	t.Run("can create proxy with a user defined delta counter port", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.DeltaCounterPort = 50000
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "deltaCounterPorts", *mockKM, 50000)
		containsPortInServicePort(t, 50000, *mockKM)
	})

	t.Run("can create proxy with a user defined Wavefront tracing", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Tracing = wf.Tracing{
				Wavefront: wf.WavefrontTracing{
					Port:             30000,
					SamplingRate:     ".1",
					SamplingDuration: 45,
				},
			}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "traceListenerPorts", *mockKM, 30000)
		containsPortInServicePort(t, 30000, *mockKM)

		containsProxyArg(t, "--traceSamplingRate .1", *mockKM)
		containsProxyArg(t, "--traceSamplingDuration 45", *mockKM)
	})

	t.Run("can create proxy with a user defined Jaeger distributed tracing", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Tracing = wf.Tracing{
				Jaeger: wf.JaegerTracing{
					Port:            30001,
					GrpcPort:        14250,
					HttpPort:        30080,
					ApplicationName: "jaeger",
				},
			}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "traceJaegerListenerPorts", *mockKM, 30001)
		containsPortInServicePort(t, 30001, *mockKM)

		containsPortInContainers(t, "traceJaegerGrpcListenerPorts", *mockKM, 14250)
		containsPortInServicePort(t, 14250, *mockKM)

		containsPortInContainers(t, "traceJaegerHttpListenerPorts", *mockKM, 30080)
		containsPortInServicePort(t, 30080, *mockKM)

		containsProxyArg(t, "--traceJaegerApplicationName jaeger", *mockKM)
	})

	t.Run("can create proxy with a user defined ZipKin distributed tracing", func(t *testing.T) {
		wfSpec := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Tracing.Zipkin.Port = 9411
			w.Spec.DataExport.WavefrontProxy.Tracing.Zipkin.ApplicationName = "zipkin"
		})

		r, mockKM := emptyScenario(wfSpec)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "traceZipkinListenerPorts", *mockKM, 9411)
		containsPortInServicePort(t, 9411, *mockKM)

		containsProxyArg(t, "--traceZipkinApplicationName zipkin", *mockKM)
	})

	t.Run("can create proxy with OLTP enabled", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.OLTP = wf.OLTP{
				GrpcPort:                       4317,
				HttpPort:                       4318,
				ResourceAttrsOnMetricsIncluded: true,
			}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "otlpGrpcListenerPorts", *mockKM, 4317)
		containsPortInServicePort(t, 4317, *mockKM)

		containsPortInContainers(t, "otlpHttpListenerPorts", *mockKM, 4318)
		containsPortInServicePort(t, 4318, *mockKM)

		containsProxyArg(t, "--otlpResourceAttrsOnMetricsIncluded true", *mockKM)
	})

	t.Run("can create proxy with histogram ports enabled", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Histogram.Port = 40000
			w.Spec.DataExport.WavefrontProxy.Histogram.MinutePort = 40001
			w.Spec.DataExport.WavefrontProxy.Histogram.HourPort = 40002
			w.Spec.DataExport.WavefrontProxy.Histogram.DayPort = 40003
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsPortInContainers(t, "histogramDistListenerPorts", *mockKM, 40000)
		containsPortInServicePort(t, 40000, *mockKM)

		containsPortInContainers(t, "histogramMinuteListenerPorts", *mockKM, 40001)
		containsPortInServicePort(t, 40001, *mockKM)

		containsPortInContainers(t, "histogramHourListenerPorts", *mockKM, 40002)
		containsPortInServicePort(t, 40002, *mockKM)

		containsPortInContainers(t, "histogramDayListenerPorts", *mockKM, 40003)
		containsPortInServicePort(t, 40003, *mockKM)
	})

	t.Run("can create proxy with a user defined proxy args", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Args = "--prefix dev \r\n --customSourceTags mySource"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--prefix dev", *mockKM)
		containsProxyArg(t, "--customSourceTags mySource", *mockKM)
	})

	t.Run("can create proxy with preprocessor rules", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Preprocessor = "preprocessor-rules"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", *mockKM)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		volumeMountHasPath(t, deployment, "preprocessor", "/etc/wavefront/preprocessor")
		volumeHasConfigMap(t, deployment, "preprocessor", "preprocessor-rules")
	})

	t.Run("resources set for the proxy", func(t *testing.T) {
		r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "100m"
			w.Spec.DataExport.WavefrontProxy.Resources.Requests.Memory = "1Gi"
			w.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "1000m"
			w.Spec.DataExport.WavefrontProxy.Resources.Limits.Memory = "4Gi"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		require.Equal(t, "1Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "4Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	})

	t.Run("adjusting proxy replicas", func(t *testing.T) {
		t.Run("changes the number of desired replicas", func(t *testing.T) {
			r, mockKM := emptyScenario(wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Replicas = 2
			}))

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
			require.NoError(t, err)

			require.Equal(t, int32(2), *deployment.Spec.Replicas)
		})

		t.Run("updates available replicas when based availability", func(t *testing.T) {
			r, mockKM := emptyScenario(
				wftest.CR(func(w *wf.Wavefront) {
					w.Spec.DataExport.WavefrontProxy.Replicas = 2
				}),
				wftest.Proxy(wftest.WithReplicas(2, 2)),
			)

			_, err := r.Reconcile(context.Background(), defaultRequest())
			require.NoError(t, err)

			require.True(t, mockKM.NodeCollectorDaemonSetContains("proxy-available-replicas: \"2\""))
			require.True(t, mockKM.ClusterCollectorDeploymentContains("proxy-available-replicas: \"2\""))
			require.True(t, mockKM.LoggingDaemonSetContains("proxy-available-replicas: \"2\""))
		})
	})

	t.Run("can create proxy with HTTP configurations", func(t *testing.T) {
		r, mockKM := emptyScenario(
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
			}),
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testHttpProxySecret",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string][]byte{
					"http-url":            []byte("https://myproxyhost_url:8080"),
					"basic-auth-username": []byte("myUser"),
					"basic-auth-password": []byte("myPassword"),
					"tls-root-ca-bundle":  []byte("myCert"),
				},
			},
		)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyhost_url ", *mockKM)
		containsProxyArg(t, "--proxyPort 8080", *mockKM)
		containsProxyArg(t, "--proxyUser myUser", *mockKM)
		containsProxyArg(t, "--proxyPassword myPassword", *mockKM)

		initContainerVolumeMountHasPath(t, deployment, "http-proxy-ca", "/tmp/ca")
		volumeHasSecret(t, deployment, "http-proxy-ca", "testHttpProxySecret")

		require.NotEmpty(t, deployment.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
	})

	t.Run("can create proxy with HTTP configurations only contains http-url", func(t *testing.T) {
		r, mockKM := emptyScenario(
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
			}),
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testHttpProxySecret",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string][]byte{
					"http-url": []byte("https://myproxyhost_url:8080"),
				},
			},
		)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyhost_url ", *mockKM)
		containsProxyArg(t, "--proxyPort 8080", *mockKM)
	})

	t.Run("can create proxy with HTTP configuration where url is a service", func(t *testing.T) {
		r, mockKM := emptyScenario(
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
			}),
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testHttpProxySecret",
					Namespace: wftest.DefaultNamespace,
				},
				Data: map[string][]byte{
					"http-url": []byte("myproxyservice:8080"),
				},
			},
		)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
		require.NoError(t, err)

		containsProxyArg(t, "--proxyHost myproxyservice", *mockKM)
		containsProxyArg(t, "--proxyPort 8080", *mockKM)

		require.NotEmpty(t, deployment.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
	})

	t.Run("can be disabled", func(t *testing.T) {
		CanBeDisabled(t,
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataExport.WavefrontProxy.Enable = false
			}),
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.ProxyName,
					Namespace: wftest.DefaultNamespace,
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
		r, mockKM := componentScenario(wftest.CR())

		_, err := r.Reconcile(context.Background(), defaultRequest())
		ds, err := mockKM.GetAppliedDaemonSet("logging", util.LoggingName)
		require.NoError(t, err)
		require.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])

		require.NoError(t, err)
		require.True(t, mockKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", util.LoggingName))
	})

	t.Run("default resources for logging", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR())

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.True(t, mockKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", "wavefront-logging"))
		require.True(t, mockKM.LoggingDaemonSetContains("resources"))
		require.False(t, mockKM.LoggingDaemonSetContains("limits:", "requests:"))
	})

	t.Run("resources set for logging", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Logging.Resources.Requests.CPU = "200m"
			w.Spec.DataCollection.Logging.Resources.Requests.Memory = "10Mi"
			w.Spec.DataCollection.Logging.Resources.Limits.Memory = "256Mi"
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.True(t, mockKM.AppliedContains("apps/v1", "DaemonSet", "wavefront", "logging", "wavefront-logging"))
		require.True(t, mockKM.LoggingDaemonSetContains("memory: 10Mi"))
		require.True(t, mockKM.LoggingDaemonSetContains("cpu: 200m"))
	})

	t.Run("Verify log tag allow list", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Logging.Filters = wf.LogFilters{
				TagDenyList:  nil,
				TagAllowList: map[string][]string{"namespace_name": {"kube-sys", "wavefront"}, "pod_name": {"pet-clinic"}},
			}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.True(t, mockKM.LoggingConfigMapContains("key $.namespace_name"))
		require.True(t, mockKM.LoggingConfigMapContains("key $.pod_name"))
		require.True(t, mockKM.LoggingConfigMapContains("pattern /(^kube-sys$|^wavefront$)/"))
		require.True(t, mockKM.LoggingConfigMapContains("pattern /(^pet-clinic$)/"))
	})

	t.Run("Verify log tag deny list", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Logging.Filters = wf.LogFilters{
				TagDenyList:  map[string][]string{"namespace_name": {"deny-kube-sys", "deny-wavefront"}, "pod_name": {"deny-pet-clinic"}},
				TagAllowList: nil,
			}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.True(t, mockKM.LoggingConfigMapContains("key $.namespace_name"))
		require.True(t, mockKM.LoggingConfigMapContains("key $.pod_name"))
		require.True(t, mockKM.LoggingConfigMapContains("pattern /(^deny-kube-sys$|^deny-wavefront$)/"))
		require.True(t, mockKM.LoggingConfigMapContains("pattern /(^deny-pet-clinic$)/"))
	})

	t.Run("Verify tags are added to logging pods", func(t *testing.T) {
		r, mockKM := componentScenario(wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Logging.Tags = map[string]string{"key1": "value1", "key2": "value2"}
		}))

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)
		require.True(t, mockKM.LoggingConfigMapContains("key1 value1", "key2 value2"))
	})

	t.Run("can be disabled", func(t *testing.T) {
		CanBeDisabled(t,
			wftest.CR(func(w *wf.Wavefront) {
				w.Spec.DataCollection.Logging.Enable = false
			}),
			&appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.LoggingName,
					Namespace: wftest.DefaultNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "logging",
					},
				},
			},
		)
	})
}

func VersionSent(mockSender *testhelper.MockSender) float64 {
	var versionSent float64
	for _, m := range mockSender.SentMetrics {
		if m.Name == "kubernetes.observability.version" {
			versionSent = m.Value
		}
	}
	return versionSent
}

func StatusMetricsSent(mockSender *testhelper.MockSender) int {
	var statusMetricsSent int
	for _, m := range mockSender.SentMetrics {
		if strings.HasSuffix(m.Name, ".status") {
			statusMetricsSent += 1
		}
	}
	return statusMetricsSent
}

func CanBeDisabled(t *testing.T, wfCR *wf.Wavefront, existingResources ...runtime.Object) {
	t.Run("on CR creation", func(t *testing.T) {
		r, mockKM := emptyScenario(wfCR)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		for _, e := range existingResources {
			objMeta := e.(metav1.ObjectMetaAccessor).GetObjectMeta()
			gvk := e.GetObjectKind().GroupVersionKind()
			require.Falsef(t, mockKM.AppliedContains(
				gvk.GroupVersion().String(), gvk.Kind,
				objMeta.GetLabels()["app.kubernetes.io/name"],
				objMeta.GetLabels()["app.kubernetes.io/component"],
				objMeta.GetName(),
			), "%s/%s should not have been applied", gvk.Kind, objMeta.GetName())
		}
	})

	t.Run("on CR update", func(t *testing.T) {
		r, mockKM := emptyScenario(wfCR, existingResources...)

		_, err := r.Reconcile(context.Background(), defaultRequest())
		require.NoError(t, err)

		for _, e := range existingResources {
			objMeta := e.(metav1.ObjectMetaAccessor).GetObjectMeta()
			gvk := e.GetObjectKind().GroupVersionKind()
			require.True(t, mockKM.DeletedContains(
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
			require.Equal(t, path, volumeMount.MountPath)
			return
		}
	}
	require.Failf(t, "could not find volume mount", "could not find volume mount named %s on deployment %s", name, deployment.Name)
}

func volumeHasConfigMap(t *testing.T, deployment appsv1.Deployment, name string, configMapName string) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == name {
			require.Equal(t, configMapName, volume.ConfigMap.Name)
			return
		}
	}
	require.Failf(t, "could not find volume", "could not find volume named %s on deployment %s", name, deployment.Name)
}

func volumeHasSecret(t *testing.T, deployment appsv1.Deployment, name string, secretName string) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == name {
			require.Equal(t, secretName, volume.Secret.SecretName)
			return
		}
	}
	require.Failf(t, "could not find secret", "could not find secret named %s on deployment %s", name, deployment.Name)
}

func initContainerVolumeMountHasPath(t *testing.T, deployment appsv1.Deployment, name, path string) {
	for _, volumeMount := range deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts {
		if volumeMount.Name == name {
			require.Equal(t, path, volumeMount.MountPath)
			return
		}
	}
	require.Failf(t, "could not find init container volume mount", "could not find init container volume mount named %s on deployment %s", name, deployment.Name)
}

func containsPortInServicePort(t *testing.T, port int32, mockKM testhelper.MockKubernetesManager) {
	serviceYAMLUnstructured, err := mockKM.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
	require.NoError(t, err)

	var service v1.Service

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(serviceYAMLUnstructured.Object, &service)
	require.NoError(t, err)

	for _, servicePort := range service.Spec.Ports {
		if servicePort.Port == port {
			return
		}
	}
	require.Fail(t, fmt.Sprintf("Did not find the port: %d", port))
}

func containsPortInContainers(t *testing.T, proxyArgName string, mockKM testhelper.MockKubernetesManager, port int32) bool {
	t.Helper()
	deploymentYAMLUnstructured, err := mockKM.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		util.ProxyName,
	)
	require.NoError(t, err)

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	require.NoError(t, err)

	foundPort := false
	for _, containerPort := range deployment.Spec.Template.Spec.Containers[0].Ports {
		if containerPort.ContainerPort == port {
			foundPort = true
			break
		}
	}
	require.True(t, foundPort, fmt.Sprintf("Did not find the port: %d", port))

	proxyArgsEnvValue := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	require.Contains(t, proxyArgsEnvValue, fmt.Sprintf("--%s %d", proxyArgName, port))
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

func containsProxyArg(t *testing.T, proxyArg string, mockKM testhelper.MockKubernetesManager) {
	deployment, err := mockKM.GetAppliedDeployment("proxy", util.ProxyName)
	require.NoError(t, err)

	value := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	require.Contains(t, value, fmt.Sprintf("%s", proxyArg))
}

func emptyScenario(wfCR *wf.Wavefront, initObjs ...runtime.Object) (*controllers.WavefrontReconciler, *testhelper.MockKubernetesManager) {
	s := scheme.Scheme
	s.AddKnownTypes(wf.GroupVersion, &wf.Wavefront{})

	namespace := wftest.DefaultNamespace
	if wfCR != nil {
		namespace = wfCR.Namespace
	}

	if !containsObject(initObjs, operatorInNamespace(namespace)) {
		operator := wftest.Operator()
		operator.SetNamespace(namespace)
		initObjs = append(initObjs, operator)
	}

	clientBuilder := fake.NewClientBuilder().WithScheme(s)
	if wfCR != nil {
		clientBuilder = clientBuilder.WithObjects(wfCR)
	}
	clientBuilder = clientBuilder.WithRuntimeObjects(initObjs...)
	objClient := clientBuilder.Build()

	mockKM := testhelper.NewMockKubernetesManager()

	r := &controllers.WavefrontReconciler{
		OperatorVersion:   "99.99.99",
		Client:            objClient,
		FS:                os.DirFS(controllers.DeployDir),
		KubernetesManager: mockKM,
		MetricConnection:  metric.NewConnection(testhelper.StubSenderFactory(nil, nil)),
	}

	return r, mockKM
}

func componentScenario(wfCR *wf.Wavefront, initObjs ...runtime.Object) (*controllers.WavefrontReconciler, *testhelper.MockKubernetesManager) {
	if !containsObject(initObjs, proxyInNamespace(wfCR.Namespace)) {
		proxy := wftest.Proxy(wftest.WithReplicas(1, 1))
		proxy.SetNamespace(wfCR.Namespace)
		initObjs = append(initObjs, proxy)
	}
	return emptyScenario(wfCR, initObjs...)
}

func operatorInNamespace(namespace string) func(obj client.Object) bool {
	return func(obj client.Object) bool {
		labels := obj.GetLabels()
		return obj.GetNamespace() == namespace &&
			labels["app.kubernetes.io/name"] == "wavefront" &&
			labels["app.kubernetes.io/component"] == "controller-manager"
	}
}

func proxyInNamespace(namespace string) func(obj client.Object) bool {
	return func(obj client.Object) bool {
		labels := obj.GetLabels()
		return obj.GetNamespace() == namespace &&
			labels["app.kubernetes.io/name"] == "wavefront" &&
			labels["app.kubernetes.io/component"] == "proxy"
	}
}

func containsObject(runtimeObjs []runtime.Object, matches func(obj client.Object) bool) bool {
	for _, runtimeObj := range runtimeObjs {
		obj, ok := runtimeObj.(client.Object)
		if !ok {
			continue
		}
		if matches(obj) {
			return true
		}
	}
	return false
}

func defaultRequest() reconcile.Request {
	return reconcile.Request{NamespacedName: util.ObjKey(wftest.DefaultNamespace, "wavefront")}
}
