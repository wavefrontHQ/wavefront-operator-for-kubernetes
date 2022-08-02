package controllers_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	objYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	manager "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientgotesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileAll(t *testing.T) {
	t.Run("creates proxy, proxy service, collector and collector service", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		r, _, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		results, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, results)

		assert.True(t, stubKM.appliedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		assert.True(t, stubKM.appliedContains("apps/v1", "DaemonSet", "wavefront", "collector", "wavefront-node-collector"))
		assert.True(t, stubKM.appliedContains("apps/v1", "Deployment", "wavefront", "collector", "wavefront-cluster-collector"))

		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"proxy",
			"wavefront-proxy",
			"value: testWavefrontUrl/api/",
			"name: testToken",
			"containerPort: 2878",
		))

		assert.True(t, stubKM.appliedContains(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: wavefront-proxy:2878",
		))

		assert.True(t, stubKM.appliedContains(
			"v1",
			"Service",
			"wavefront",
			"proxy",
			"wavefront-proxy",
			"port: 2878",
		))
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		r, wfCR, apiClient, _, _ := setup("testWavefrontUrl", "updatedToken", "testClusterName")
		r.KubernetesManager = stubKM

		err := apiClient.Delete(context.Background(), wfCR)

		_, err = r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.deletedContains("v1", "ServiceAccount", "wavefront", "collector", "wavefront-collector"))
		assert.True(t, stubKM.deletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		assert.True(t, stubKM.deletedContains("apps/v1", "DaemonSet", "wavefront", "collector", "wavefront-node-collector"))
		assert.True(t, stubKM.deletedContains("apps/v1", "Deployment", "wavefront", "collector", "wavefront-cluster-collector"))
		assert.True(t, stubKM.deletedContains("v1", "Service", "wavefront", "proxy", "wavefront-proxy"))
		assert.True(t, stubKM.deletedContains("apps/v1", "Deployment", "wavefront", "proxy", "wavefront-proxy"))
	})
}

func TestReconcileCollector(t *testing.T) {
	t.Run("does not create configmap if user specified one", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.CustomConfig = "myconfig"
		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		// User is responsible for applying ConfigMap
		// TODO: I believe this is set in the spec, which is pass by value in setup...
		//assert.False(t, stubKM.appliedContains("ConfigMap", "myconfig"))

		// It DOES call the ApplyResources function with the ConfigMap, but it's filtered out
		// TODO: test in KubernetesManager
		assert.True(t, stubKM.appliedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))
		assert.False(t, stubKM.deletedContains("v1", "ConfigMap", "wavefront", "collector", "default-wavefront-collector-config"))

		configMapYAML := `
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app.kubernetes.io/name : wavefront
    app.kubernetes.io/component: collector
  name: default-wavefront-collector-config
  namespace: wavefront
`
		var resourceDecoder = objYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		configMapObject := &unstructured.Unstructured{}
		_, _, err = resourceDecoder.Decode([]byte(configMapYAML), nil, configMapObject)
		assert.NoError(t, err)

		assert.False(t, stubKM.objectPassesFilter(
			configMapObject,
		))
	})

	t.Run("defaults values for default collector config", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}
		wfSpec := defaultWFSpec()

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		//configMap := getCreatedConfigMap(t, dynamicClient)
		//assert.Contains(t, configMap.Data["config.yaml"])
		//assert.Contains(t, configMap.Data["config.yaml"])
		//assert.Contains(t, configMap.Data["config.yaml"])

		assert.True(t, stubKM.appliedContains(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"defaultCollectionInterval: 60s",
			"enableDiscovery: true",
		))
	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "256Mi"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ClusterCollectorName)
		//assert.Equal(t, "10Mi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"collector",
			"wavefront-cluster-collector",
			"memory: 10Mi",
		))
	})

	t.Run("resources set for node collector", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Limits.Memory = "256Mi"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//daemonSet := getCreatedDaemonSet(t, dynamicClient)
		//assert.Equal(t, "memory: 10Mi", daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())

		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"DaemonSet",
			"wavefront",
			"collector",
			"wavefront-node-collector",
			"memory: 10Mi",
		))
	})

	t.Run("no resources set for node and cluster collector", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		r, _, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		// TODO: lots of lines of test code... what we can do better? Squash them onto one line?
		//daemonSet := getCreatedDaemonSet(t, dynamicClient)
		//assert.Nil(t, daemonSet.Spec.Template.Spec.Containers[0].Resources.Limits)
		//assert.Nil(t, daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests)
		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"DaemonSet",
			"wavefront",
			"collector",
			"wavefront-node-collector",
			"resources:",
		))
		assert.False(t, stubKM.appliedContains(
			"apps/v1",
			"DaemonSet",
			"wavefront",
			"collector",
			"wavefront-node-collector",
			"limits:",
		))
		assert.False(t, stubKM.appliedContains(
			"apps/v1",
			"DaemonSet",
			"wavefront",
			"collector",
			"wavefront-node-collector",
			"requests:",
		))

		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ClusterCollectorName)
		//assert.Nil(t, deployment.Spec.Template.Spec.Containers[0].Resources.Limits)
		//assert.Nil(t, deployment.Spec.Template.Spec.Containers[0].Resources.Requests)
		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"collector",
			"wavefront-cluster-collector",
			"resources:",
		))
		assert.False(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"collector",
			"wavefront-cluster-collector",
			"limits:",
		))
		assert.False(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"collector",
			"wavefront-cluster-collector",
			"requests:",
		))
	})

	t.Run("skip creating collector if metrics is not enabled", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics = wf.Metrics{}

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		//assert.Equal(t, 4, len(dynamicClient.Actions()))
		//assert.True(t, hasAction(dynamicClient, "create", "services"), "create Service")
		//assert.True(t, hasAction(dynamicClient, "create", "deployments"), "create Deployment")
		serviceAccountYAML := `
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: wavefront
    app.kubernetes.io/component: collector
  name: wavefront-collector
  namespace: wavefront
`
		var resourceDecoder = objYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		serviceAccountObject := &unstructured.Unstructured{}
		_, _, err = resourceDecoder.Decode([]byte(serviceAccountYAML), nil, serviceAccountObject)
		assert.NoError(t, err)

		// TODO: NOTE: the filter is only based on app.kubernetes.io/component value
		// so I only tested one object
		assert.False(t, stubKM.objectPassesFilter(
			serviceAccountObject,
		))

		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
		//assert.Equal(t, "testWavefrontUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		//assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name)
		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"proxy",
			"wavefront-proxy",
			"value: testWavefrontUrl/api/",
			"name: testToken",
			"containerPort: 2878",
		))
	})

	t.Run("Values from metrics.filters is propagated to default collector configmap", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics = wf.Metrics{
			Enable: true,
			Filters: wf.Filters{
				DenyList:  []string{"first_deny", "second_deny"},
				AllowList: []string{"first_allow", "second_allow"},
			},
		}

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		configMap, err := stubKM.getAppliedYAML(
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
		stubKM := &stubKubernetesManager{}

		r, _, _, _, _ := setupForCreate(defaultWFSpec())
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())

		assert.NoError(t, err)

		//assert.True(t, hasAction(dynamicClient, "create", "deployments"), "create Deployment")

		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
		//assert.Equal(t, "testWavefrontUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		//assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name)
		//assert.Equal(t, int32(2878), deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
		//assert.Equal(t, "", deployment.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"proxy",
			"wavefront-proxy",
			"value: testWavefrontUrl/api/",
			"name: testToken",
			"containerPort: 2878",
			"configHash: \"\"",
		))

		//service := getCreatedService(t, dynamicClient)
		//assert.Equal(t, int32(2878), service.Spec.Ports[0].Port)
		assert.True(t, stubKM.appliedContains(
			"v1",
			"Service",
			"wavefront",
			"proxy",
			"wavefront-proxy",
			"port: 2878",
		))
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		r, _, _, _, _ := setup("testWavefrontUrl", "updatedToken", "testClusterName")
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		assert.True(t, stubKM.appliedContains(
			"apps/v1",
			"Deployment",
			"wavefront",
			"proxy",
			"wavefront-proxy",
			"name: updatedToken",
			"value: testWavefrontUrl/api/",
		))
	})

	t.Run("Skip creating proxy if DataExport.WavefrontProxy.Enable is set to false", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Enable = false

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//assert.Equal(t, 8, len(dynamicClient.Actions()))
		//configMap := getCreatedConfigMap(t, dynamicClient)
		//assert.Contains(t, configMap.Data["config.yaml"], "externalProxyUrl")
		assert.True(t, stubKM.appliedContains(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: externalProxyUrl",
		))

		proxyDeploymentYAML := `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: wavefront
    app.kubernetes.io/component: proxy
  name: wavefront-proxy
  namespace: wavefront
`
		// TODO: all of this is ugly and should be a helper function or something
		var resourceDecoder = objYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		proxyDeploymentObject := &unstructured.Unstructured{}
		_, _, err = resourceDecoder.Decode([]byte(proxyDeploymentYAML), nil, proxyDeploymentObject)
		assert.NoError(t, err)

		assert.False(t, stubKM.objectPassesFilter(
			proxyDeploymentObject,
		))

		proxyServiceYAML := `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: wavefront
    app.kubernetes.io/component: proxy
  name: wavefront-proxy
  namespace: wavefront
`
		proxyServiceObject := &unstructured.Unstructured{}
		_, _, err = resourceDecoder.Decode([]byte(proxyServiceYAML), nil, proxyServiceObject)
		assert.NoError(t, err)

		assert.False(t, stubKM.objectPassesFilter(
			proxyServiceObject,
		))
	})

	t.Run("can create proxy with a user defined metric port", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.MetricPort = 1234

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//assert.True(t, containsPortInContainers(1234, "pushListenerPorts", *stubKM))
		containsPortInContainers(t, "pushListenerPorts", *stubKM, 1234)
		containsPortInServicePort(t, 1234, *stubKM)
		//
		//configMap := getCreatedConfigMap(t, dynamicClient)
		//assert.Contains(t, configMap.Data["config.yaml"], "wavefront-proxy:1234")
		assert.True(t, stubKM.appliedContains(
			"v1",
			"ConfigMap",
			"wavefront",
			"collector",
			"default-wavefront-collector-config",
			"clusterName: testClusterName",
			"proxyAddress: wavefront-proxy:1234",
		))
	})

	t.Run("can create proxy with a user defined delta counter port", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.DeltaCounterPort = 50000
		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//containsPortInContainers(t, 50000, "deltaCounterPorts")
		containsPortInContainers(t, "deltaCounterPorts", *stubKM, 50000)
		containsPortInServicePort(t, 50000, *stubKM)
	})

	t.Run("can create proxy with a user defined Wavefront tracing", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.Port = 30000
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.SamplingRate = ".1"
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.SamplingDuration = 45

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//containsPortInContainers(t, 30000, "traceListenerPorts")
		containsPortInContainers(t, "traceListenerPorts", *stubKM, 30000)
		containsPortInServicePort(t, 30000, *stubKM)
		containsProxyArg(t, "--traceSamplingRate .1", *stubKM)
		containsProxyArg(t, "--traceSamplingDuration 45", *stubKM)
	})

	t.Run("can create proxy with a user defined Jaeger distributed tracing", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.Port = 30001
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.GrpcPort = 14250
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.HttpPort = 30080
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.ApplicationName = "jaeger"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//containsPortInContainers(t, 30001, "traceJaegerListenerPorts")
		containsPortInContainers(t, "traceJaegerListenerPorts", *stubKM, 30001)
		containsPortInServicePort(t, 30001, *stubKM)
		//containsPortInContainers(t, 14250, "traceJaegerGrpcListenerPorts")
		containsPortInContainers(t, "traceJaegerGrpcListenerPorts", *stubKM, 14250)
		containsPortInServicePort(t, 14250, *stubKM)
		//containsPortInContainers(t, 30080, "traceJaegerHttpListenerPorts")
		containsPortInContainers(t, "traceJaegerHttpListenerPorts", *stubKM, 30080)
		containsPortInServicePort(t, 30080, *stubKM)
		containsProxyArg(t, "--traceJaegerApplicationName jaeger", *stubKM)
	})

	t.Run("can create proxy with a user defined ZipKin distributed tracing", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.Port = 9411
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.ApplicationName = "zipkin"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//containsPortInContainers(t, 9411, "traceZipkinListenerPorts")
		containsPortInContainers(t, "traceZipkinListenerPorts", *stubKM, 9411)
		containsPortInServicePort(t, 9411, *stubKM)
		containsProxyArg(t, "--traceZipkinApplicationName zipkin", *stubKM)
	})

	t.Run("can create proxy with histogram ports enabled", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Histogram.Port = 40000
		wfSpec.DataExport.WavefrontProxy.Histogram.MinutePort = 40001
		wfSpec.DataExport.WavefrontProxy.Histogram.HourPort = 40002
		wfSpec.DataExport.WavefrontProxy.Histogram.DayPort = 40003

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//containsPortInContainers(t, 40000, "histogramDistListenerPorts")
		containsPortInContainers(t, "histogramDistListenerPorts", *stubKM, 40000)
		containsPortInServicePort(t, 40000, *stubKM)
		//containsPortInContainers(t, 40001, "histogramMinuteListenerPorts")
		containsPortInContainers(t, "histogramMinuteListenerPorts", *stubKM, 40001)
		containsPortInServicePort(t, 40001, *stubKM)
		//containsPortInContainers(t, 40002, "histogramHourListenerPorts")
		containsPortInContainers(t, "histogramHourListenerPorts", *stubKM, 40002)
		containsPortInServicePort(t, 40002, *stubKM)
		//containsPortInContainers(t, 40003, "histogramDayListenerPorts")
		containsPortInContainers(t, "histogramDayListenerPorts", *stubKM, 40003)
		containsPortInServicePort(t, 40003, *stubKM)
	})

	t.Run("can create proxy with a user defined proxy args", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Args = "--prefix dev \r\n --customSourceTags mySource"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsProxyArg(t, "--prefix dev", *stubKM)
		containsProxyArg(t, "--customSourceTags mySource", *stubKM)
	})

	t.Run("can create proxy with preprocessor rules", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Preprocessor = "preprocessor-rules"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", *stubKM)

		deployment, err := stubKM.getAppliedDeployment("proxy", controllers.ProxyName)
		assert.NoError(t, err)

		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
		volumeMountHasPath(t, deployment, "preprocessor", "/etc/wavefront/preprocessor")
		volumeHasConfigMap(t, deployment, "preprocessor", "preprocessor-rules")
	})

	t.Run("resources set for the proxy", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Resources.Requests.CPU = "100m"
		wfSpec.DataExport.WavefrontProxy.Resources.Requests.Memory = "1Gi"
		wfSpec.DataExport.WavefrontProxy.Resources.Limits.CPU = "1000m"
		wfSpec.DataExport.WavefrontProxy.Resources.Limits.Memory = "4Gi"

		r, _, _, _, _ := setupForCreate(wfSpec)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)


		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
		deployment, err := stubKM.getAppliedDeployment("proxy", controllers.ProxyName)
		assert.NoError(t, err)

		assert.Equal(t, "1Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		assert.Equal(t, "4Gi", deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	})

	t.Run("can create proxy with HTTP configurations", func(t *testing.T) {
		stubKM := &stubKubernetesManager{}

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

		r, _, _, _, _ := setupForCreate(wfSpec, httpProxySecet)
		r.KubernetesManager = stubKM

		_, err := r.Reconcile(context.Background(), defaultRequest())
		assert.NoError(t, err)

		//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
		deployment, err := stubKM.getAppliedDeployment("proxy", controllers.ProxyName)
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
		stubKM := &stubKubernetesManager{}

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
		r, _, _, _, _ := setupForCreate(wfSpec, httpProxySecet)
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

func containsPortInServicePort(t *testing.T, port int32, stubKM stubKubernetesManager) {
	serviceYAMLUnstructured, err := stubKM.getAppliedYAML(
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
	//service := getCreatedService(t, stubKM)
	for _, servicePort := range service.Spec.Ports {
		if servicePort.Port == port {
			return
		}
	}
	assert.Fail(t, fmt.Sprintf("Did not find the port: %d", port))
}

func containsPortInContainers(t *testing.T, proxyArgName string, stubKM stubKubernetesManager, port int32) bool {
	//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
	deploymentYAMLUnstructured, err := stubKM.getAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
	assert.NoError(t, err)

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	assert.NoError(t, err)

	//containers, found, err := unstructured.NestedSlice(
	//	deploymentYAMLUnstructured.Object,
	//	"spec",
	//	"template",
	//	"spec",
	//	"containers",
	//)
	//assert.NoError(t, err)
	//assert.True(t, found)
	//log.Printf(">>>>>>>>>>> containers[0]: %+v", containers[0])

	//var resourceDecoder = objYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	//containerPortsObject := &unstructured.Unstructured{}
	//_, _, err = resourceDecoder.Decode([]byte(containers[0].(string)), nil, containerPortsObject)
	//assert.NoError(t, err)
	//log.Printf(">>>>>>>>>>> containerPortsObject: %+v", containerPortsObject)

	//containerObj := containers[0].(map[string]map[string]interface{})
	//log.Printf(">>>>>>>>>>> containerObj: %+v", containerObj)

	//containerPorts := containerObj.

	//containerPorts, found, err := unstructured.NestedSlice(
	//	containerPortsObject.Object,
	//	"ports",
	//)
	//assert.NoError(t, err)
	//assert.True(t, found)

	//log.Printf(">>>>>>>>>>> containerPorts: %+v", containerPorts)

	foundPort := false
	for _, containerPort := range deployment.Spec.Template.Spec.Containers[0].Ports {
		//for _, containerPort := range containerObj { // TODO: need to get "ports" field
		if containerPort.ContainerPort == port {
			foundPort = true
			break
		}

		fmt.Printf("%+v", containerPort)

		//containerPortCheck := fmt.Sprintf("ContainerPort: %d", port)

		//if containerPort.(string) == containerPortCheck {
		//	foundPort = true
		//	break
		//}
	}
	//if !foundPort {
	//	log.Printf("Did not find the port: %d", port)
	//	return false
	//}
	assert.True(t, foundPort, fmt.Sprintf("Did not find the port: %d", port))

	//if !strings.Contains(proxyArgsEnvValue, fmt.Sprintf("--%s %d", proxyArgName, port)) {
	//	log.Printf("Env did not have proxy args: %s", proxyArgsEnvValue)
	//	return false
	//}
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

func containsProxyArg(t *testing.T, proxyArg string, stubKM stubKubernetesManager) {
	deployment, err := stubKM.getAppliedDeployment("proxy", "wavefront-proxy")
	assert.NoError(t, err)

	//deployment := getCreatedDeployment(t, dynamicClient, controllers.ProxyName)
	value := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	assert.Contains(t, value, fmt.Sprintf("%s", proxyArg))
}

func getCreatedConfigMap(t *testing.T, dynamicClient *dynamicfake.FakeDynamicClient) v1.ConfigMap {
	configMapObject := getAction(dynamicClient, "create", "configmaps").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
	var configMap v1.ConfigMap
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(configMapObject.Object, &configMap)
	assert.NoError(t, err)
	return configMap
}

func getCreatedDeployment(t *testing.T, dynamicClient *dynamicfake.FakeDynamicClient, deploymentName string) appsv1.Deployment {
	deploymentObject := getCreateObject(dynamicClient, "deployments", deploymentName)
	var deployment appsv1.Deployment
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)
	assert.NoError(t, err)
	return deployment
}

func getCreatedService(t *testing.T, dynamicClient *dynamicfake.FakeDynamicClient) v1.Service {
	serviceObject := getAction(dynamicClient, "create", "services").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
	var service v1.Service

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(serviceObject.Object, &service)
	assert.NoError(t, err)
	return service
}

func getCreatedDaemonSet(t *testing.T, dynamicClient *dynamicfake.FakeDynamicClient) appsv1.DaemonSet {
	daemonSetObject := getCreateObject(dynamicClient, "daemonsets", controllers.NodeCollectorName)
	var ds appsv1.DaemonSet
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(daemonSetObject.Object, &ds)
	assert.NoError(t, err)
	return ds
}

func defaultWFSpec() wf.WavefrontSpec {
	return wf.WavefrontSpec{
		ClusterName:          "testClusterName",
		WavefrontUrl:         "testWavefrontUrl",
		WavefrontTokenSecret: "testToken",
		DataExport: wf.DataExport{
			ExternalWavefrontProxy: wf.ExternalWavefrontProxy{
				Url: "externalProxyUrl",
			},
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
func getCreateObject(dynamicClient *dynamicfake.FakeDynamicClient, resource string, metadataName string) *unstructured.Unstructured {
	//deploymentObject := getAction(dynamicClient, "create", "deployments").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
	for _, action := range dynamicClient.Actions() {
		if action.GetVerb() == "create" && action.GetResource().Resource == resource {
			resourceObj := action.(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
			if resourceObj.GetName() == metadataName {
				return resourceObj
			}
		}
	}
	return nil
}

func getPatch(dynamicClient *dynamicfake.FakeDynamicClient, resource string, metadataName string) []byte {
	//deploymentObject := getAction(dynamicClient, "create", "deployments").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
	for _, action := range dynamicClient.Actions() {
		if action.GetVerb() == "patch" && action.GetResource().Resource == resource {
			resourceObj := action.(clientgotesting.PatchActionImpl)
			if resourceObj.GetName() == metadataName {
				return resourceObj.Patch
			}
		}
	}
	return nil
}

func hasAction(dynamicClient *dynamicfake.FakeDynamicClient, verb, resource string) (result bool) {
	if getAction(dynamicClient, verb, resource) != nil {
		return true
	}
	return false
}

func getAction(dynamicClient *dynamicfake.FakeDynamicClient, verb, resource string) (action clientgotesting.Action) {
	for _, action := range dynamicClient.Actions() {
		if action.GetVerb() == verb && action.GetResource().Resource == resource {
			return action
		}
	}
	return nil
}

func setupForCreate(spec wf.WavefrontSpec, initObjs ...runtime.Object) (*controllers.WavefrontReconciler, *wf.Wavefront, client.WithWatch, *dynamicfake.FakeDynamicClient, typedappsv1.AppsV1Interface) {
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

	testRestMapper := meta.NewDefaultRESTMapper(
		[]schema.GroupVersion{
			{Group: "apps", Version: "v1"},
		})
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, meta.RESTScopeNamespace)
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "wavefront.com",
		Version: "v1alpha1",
		Kind:    "Wavefront",
	}, meta.RESTScopeNamespace)
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, meta.RESTScopeNamespace)
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, meta.RESTScopeNamespace)
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, meta.RESTScopeNamespace)
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ServiceAccount",
	}, meta.RESTScopeNamespace)
	testRestMapper.Add(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "DaemonSet",
	}, meta.RESTScopeNamespace)

	clientBuilder := fake.NewClientBuilder()
	clientBuilder = clientBuilder.WithScheme(s).WithObjects(wfCR)
	clientBuilder = clientBuilder.WithScheme(s).WithRuntimeObjects(initObjs...)
	clientBuilder = clientBuilder.WithRESTMapper(testRestMapper)
	apiClient := clientBuilder.Build()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(s)

	initObjs = append(initObjs, &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
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

	kubernetesManager, err := manager.NewKubernetesManager(apiClient.RESTMapper(), dynamicClient)
	if err != nil {
		panic(err)
	}

	r := &controllers.WavefrontReconciler{
		Client:            apiClient,
		Scheme:            nil,
		FS:                os.DirFS(controllers.DeployDir),
		KubernetesManager: kubernetesManager,
		Appsv1:            fakesAppsV1,
	}

	return r, wfCR, apiClient, dynamicClient, fakesAppsV1
}

func setup(wavefrontUrl, wavefrontTokenSecret, clusterName string) (*controllers.WavefrontReconciler, *wf.Wavefront, client.WithWatch, *dynamicfake.FakeDynamicClient, typedappsv1.AppsV1Interface) {
	wfSpec := defaultWFSpec()
	wfSpec.WavefrontUrl = wavefrontUrl
	wfSpec.WavefrontTokenSecret = wavefrontTokenSecret
	wfSpec.ClusterName = clusterName
	namespace := "wavefront"
	reconciler, wfCR, apiClient, dynamicClient, fakesAppsV1 := setupForCreate(wfSpec)

	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      controllers.ProxyName,
			"namespace": namespace,
		},
	}})

	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "default-wavefront-collector-config",
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
		"data": map[string]interface{}{
			"config.yaml": "foo",
		},
	}})

	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "DaemonSet",
		"metadata": map[string]interface{}{
			"name":      controllers.NodeCollectorName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
	}})

	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      controllers.ClusterCollectorName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
	}})

	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      controllers.ProxyName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "proxy",
			},
		},
		"spec": map[string]interface{}{
			"type": "ClusterIP",
		},
	}})
	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ServiceAccount",
		"metadata": map[string]interface{}{
			"name":      "wavefront-collector",
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
	}})

	return reconciler, wfCR, apiClient, dynamicClient, fakesAppsV1
}

func defaultRequest() reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: "wavefront",
		Name:      "wavefront",
	}}
}

type stubKubernetesManager struct {
	deletedYAMLs []string
	appliedYAMLs []string
	usedFilter   func(*unstructured.Unstructured) bool
}

func contains(
	yamls []string,
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	reg, err := k8sYAMLHeader(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName)
	if err != nil {
		panic(err)
	}

	for _, yamlStr := range yamls {
		if reg.MatchString(yamlStr) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return false
				}
			}
			return true
		}
	}

	return false
}

func k8sYAMLHeader(apiVersion string, kind string, appKubernetesIOName string, appKubernetesIOComponent string, metadataName string) (*regexp.Regexp, error) {
	headerMatchStr := fmt.Sprintf(
		`apiVersion: %s
kind: %s
metadata:
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: %s
  name: %s`,
		apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName)

	reg, err := regexp.Compile(headerMatchStr)
	return reg, err
}

func (skm stubKubernetesManager) deletedContains(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	return contains(
		skm.deletedYAMLs,
		apiVersion,
		kind,
		appKubernetesIOName,
		appKubernetesIOComponent,
		metadataName,
		otherChecks...,
	)
}

func (skm stubKubernetesManager) appliedContains(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	return contains(
		skm.appliedYAMLs,
		apiVersion,
		kind,
		appKubernetesIOName,
		appKubernetesIOComponent,
		metadataName,
		otherChecks...,
	)
}

func (skm stubKubernetesManager) objectPassesFilter(object *unstructured.Unstructured) bool {
	// TODO: filter returning true if filtered is confusing
	return !skm.usedFilter(object)
}

func (skm *stubKubernetesManager) ApplyResources(resourceYAMLs []string, filterObject func(*unstructured.Unstructured) bool) error {
	skm.appliedYAMLs = resourceYAMLs
	skm.usedFilter = filterObject
	return nil
}

func (skm *stubKubernetesManager) DeleteResources(resourceYAMLs []string) error {
	skm.deletedYAMLs = resourceYAMLs
	return nil
}

func (skm stubKubernetesManager) getAppliedYAML(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) (*unstructured.Unstructured, error) {
	reg, err := k8sYAMLHeader(apiVersion, kind, appKubernetesIOName, appKubernetesIOComponent, metadataName)
	if err != nil {
		return nil, err
	}

	for _, yamlStr := range skm.appliedYAMLs {
		if reg.MatchString(yamlStr) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return nil, errors.New("no YAML matched conditions passed")
				}
			}
			object := &unstructured.Unstructured{}
			var resourceDecoder = objYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
			_, _, err := resourceDecoder.Decode([]byte(yamlStr), nil, object)
			return object, err
		}
	}
	return nil, nil
}

func (skm stubKubernetesManager) getAppliedDeployment(appKubernetesIOComponent, metadataName string) (appsv1.Deployment, error) {
	deploymentYAMLUnstructured, err := skm.getAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentYAMLUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}
