package controllers_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	test_helper "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/test"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/assert"
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
		stubKM := test_helper.NewStubKubernetesManager()

		r, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		results, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, results)

		assert.True(t, stubKM.CollectorServiceAccountContains())
		assert.True(t, stubKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: wavefront-proxy:2878"))
		assert.True(t, stubKM.NodeCollectorDaemonSetContains())
		assert.True(t, stubKM.ClusterCollectorDeploymentContains())
		assert.True(t, stubKM.ProxyServiceContains("port: 2878"))
		assert.True(t, stubKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878"))
	})

	t.Run("doesn't create any resources if wavefront spec is invalid", func(t *testing.T) {
		invalidWFSpec := defaultWFSpec()
		invalidWFSpec.DataExport.ExternalWavefrontProxy.Url = "http://some_url.com"
		r, _, _, dynamicClient, _ := setupForCreate(invalidWFSpec)

		results, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, results)

		assert.Equal(t, 6, len(dynamicClient.Actions()))
		assert.True(t, hasAction(dynamicClient, "get", "serviceaccounts"), "get ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "get", "configmaps"), "get Configmap")
		assert.True(t, hasAction(dynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(dynamicClient, "get", "daemonsets"), "get DaemonSet")
		// one deployment for collector and one for proxy
		assert.True(t, hasAction(dynamicClient, "get", "deployments"), "get Deployment")
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()
		wfSpec := defaultWFSpec()

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("clusterName: testClusterName", "defaultCollectionInterval: 60s", "enableDiscovery: true"))
	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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

	t.Run("skip creating collector if metrics is not enabled", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics = wf.Metrics{}

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.False(t, stubKM.ServiceAccountPassesFilter(t, err))

		assert.True(t, stubKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878"))
	})

	t.Run("Values from metrics.filters is propagated to default collector configmap", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

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
}

func TestReconcileProxy(t *testing.T) {
	// TODO: is this not already tested in TestReconcileAll?
	t.Run("creates proxy and proxy service", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

		r, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.ProxyDeploymentContains("value: testWavefrontUrl/api/", "name: testToken", "containerPort: 2878", "configHash: \"\""))

		assert.True(t, stubKM.ProxyServiceContains("port: 2878"))
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

		r, _, _, _ := setup("testWavefrontUrl", "updatedToken", "testClusterName")
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.ProxyDeploymentContains("name: updatedToken", "value: testWavefrontUrl/api/"))
	})

	t.Run("Skip creating proxy if DataExport.WavefrontProxy.Enable is set to false", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Enable = false
		wfSpec.DataExport.ExternalWavefrontProxy.Url = "externalProxyUrl"

		r, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.CollectorConfigMapContains("clusterName: testClusterName", "proxyAddress: externalProxyUrl"))

		proxyDeploymentObject, err := stubKM.GetUnstructuredProxyDeployment()
		assert.NoError(t, err)

		assert.False(t, stubKM.ObjectPassesFilter(
			proxyDeploymentObject,
		))

		proxyServiceObject, err := stubKM.GetUnstructuredProxyService()
		assert.NoError(t, err)

		assert.False(t, stubKM.ObjectPassesFilter(
			proxyServiceObject,
		))
	})

	t.Run("can create proxy with a user defined metric port", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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
		stubKM := test_helper.NewStubKubernetesManager()

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

	t.Run("can create proxy with HTTP configurations", func(t *testing.T) {
		stubKM := test_helper.NewStubKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		var httpProxySecet = &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: "wavefront",
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
		stubKM := test_helper.NewStubKubernetesManager()

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		var httpProxySecet = &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testHttpProxySecret",
				Namespace: "wavefront",
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

func containsPortInServicePort(t *testing.T, port int32, stubKM test_helper.StubKubernetesManager) {
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

func containsPortInContainers(t *testing.T, proxyArgName string, stubKM test_helper.StubKubernetesManager, port int32) bool {
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

func containsProxyArg(t *testing.T, proxyArg string, stubKM test_helper.StubKubernetesManager) {
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
			Namespace: "wavefront",
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
			Namespace: "wavefront",
			UID:       "testUID",
		},
		Spec:   appsv1.DeploymentSpec{},
		Status: appsv1.DeploymentStatus{},
	})

	fakesAppsV1 := k8sfake.NewSimpleClientset(initObjs...).AppsV1()

	stubKubernetesManager := test_helper.NewStubKubernetesManager()

	r := &controllers.WavefrontReconciler{
		Client:            apiClient,
		Scheme:            nil,
		FS:                os.DirFS(controllers.DeployDir),
		KubernetesManager: stubKubernetesManager,
		Appsv1:            fakesAppsV1,
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
		Namespace: "wavefront",
		Name:      "wavefront",
	}}
}
