package controllers_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileAll(t *testing.T) {
	t.Run("creates proxy, proxy service, collector and collector service", func(t *testing.T) {
		r, _, _, dynamicClient, _ := setupForCreate(defaultWFSpec())

		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 12, len(dynamicClient.Actions()))
		assert.True(t, hasAction(dynamicClient, "get", "serviceaccounts"), "get ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "create", "serviceaccounts"), "create ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "get", "configmaps"), "get ConfigMap")
		assert.True(t, hasAction(dynamicClient, "create", "configmaps"), "create Configmap")
		assert.True(t, hasAction(dynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(dynamicClient, "create", "services"), "create Service")
		assert.True(t, hasAction(dynamicClient, "get", "daemonsets"), "get DaemonSet")
		assert.True(t, hasAction(dynamicClient, "create", "daemonsets"), "create DaemonSet")
		assert.True(t, hasAction(dynamicClient, "get", "deployments"), "get Deployment")
		assert.True(t, hasAction(dynamicClient, "create", "deployments"), "create Deployment")

		deployment := getCreatedDeployment(t, dynamicClient, "wavefront-proxy")
		assert.Equal(t, "testWavefrontUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, int32(2878), deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)

		configMap := getCreatedConfigMap(t, dynamicClient)
		assert.Contains(t, configMap.Data["config.yaml"], "testClusterName")
		assert.Contains(t, configMap.Data["config.yaml"], "wavefront-proxy:2878")

		service := getCreatedService(t, dynamicClient)
		assert.Equal(t, int32(2878), service.Spec.Ports[0].Port)
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		wfCR, apiClient, dynamicClient, fakesAppsV1 := setup("testWavefrontUrl", "updatedToken", "testClusterName")
		err := apiClient.Delete(context.Background(), wfCR)

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakesAppsV1,
		}
		_, err = r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, 12, len(dynamicClient.Actions()))

		assert.True(t, hasAction(dynamicClient, "get", "serviceaccounts"), "get ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "delete", "serviceaccounts"), "delete ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "get", "configmaps"), "get ConfigMap")
		assert.True(t, hasAction(dynamicClient, "delete", "configmaps"), "delete Configmap")
		assert.True(t, hasAction(dynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(dynamicClient, "delete", "services"), "delete Service")
		assert.True(t, hasAction(dynamicClient, "get", "daemonsets"), "get DaemonSet")
		assert.True(t, hasAction(dynamicClient, "delete", "daemonsets"), "delete DaemonSet")
		assert.True(t, hasAction(dynamicClient, "get", "deployments"), "get Deployment")
		assert.True(t, hasAction(dynamicClient, "delete", "deployments"), "delete Deployment")
	})
}

func TestReconcileCollector(t *testing.T) {
	t.Run("does not create configmap if user specified one", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.CustomConfig = "myconfig"
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)

		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 10, len(dynamicClient.Actions()))
		assert.False(t, hasAction(dynamicClient, "get", "configmaps"), "get ConfigMap")
		assert.False(t, hasAction(dynamicClient, "create", "configmaps"), "create Configmap")
	})

	t.Run("defaults values for default collector config", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)

		configMap := getCreatedConfigMap(t, dynamicClient)
		assert.Contains(t, configMap.Data["config.yaml"], "clusterName: testClusterName")
		assert.Contains(t, configMap.Data["config.yaml"], "defaultCollectionInterval: 60s")
		assert.Contains(t, configMap.Data["config.yaml"], "enableDiscovery: true")
	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
		wfSpec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "256Mi"
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)

		deployment := getCreatedDeployment(t, dynamicClient, "wavefront-cluster-collector")
		assert.Equal(t, "10Mi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
	})

	t.Run("resources set for node collector", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "200m"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Requests.Memory = "10Mi"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
		wfSpec.DataCollection.Metrics.NodeCollector.Resources.Limits.Memory = "256Mi"
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		daemonSet := getCreatedDaemonSet(t, dynamicClient)
		assert.Equal(t, "10Mi", daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
	})

	t.Run("no resources set for node and cluster collector", func(t *testing.T) {
		r, _, _, dynamicClient, _ := setupForCreate(defaultWFSpec())

		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		daemonSet := getCreatedDaemonSet(t, dynamicClient)
		assert.Nil(t, daemonSet.Spec.Template.Spec.Containers[0].Resources.Limits)
		assert.Nil(t, daemonSet.Spec.Template.Spec.Containers[0].Resources.Requests)

		deployment := getCreatedDeployment(t, dynamicClient, "wavefront-cluster-collector")
		assert.Nil(t, deployment.Spec.Template.Spec.Containers[0].Resources.Limits)
		assert.Nil(t, deployment.Spec.Template.Spec.Containers[0].Resources.Requests)
	})

	t.Run("Skip creating collector if metrics is not enabled", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataCollection.Metrics = wf.Metrics{}
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)

		results, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 4, len(dynamicClient.Actions()))
		assert.True(t, hasAction(dynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(dynamicClient, "create", "services"), "create Service")
		assert.True(t, hasAction(dynamicClient, "get", "deployments"), "get Deployment")
		assert.True(t, hasAction(dynamicClient, "create", "deployments"), "create Deployment")

		deployment := getCreatedDeployment(t, dynamicClient, "wavefront-proxy")
		assert.Equal(t, "testWavefrontUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name)

	})

}

func TestReconcileProxy(t *testing.T) {
	t.Run("updates proxy and service", func(t *testing.T) {
		_, apiClient, dynamicClient, fakesAppsV1 := setup("testWavefrontUrl", "updatedToken", "testClusterName")

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakesAppsV1,
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 12, len(dynamicClient.Actions()))

		deploymentObject := getPatch(dynamicClient, "deployments", "wavefront-proxy")

		assert.Contains(t, string(deploymentObject), "updatedToken")
		assert.Contains(t, string(deploymentObject), "testWavefrontUrl/api/")

		assert.NoError(t, err)
	})

	t.Run("Skip creating proxy if DataExport.WavefrontProxy.Enable is set to false", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Enable = false

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		results, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)

		assert.Equal(t, 8, len(dynamicClient.Actions()))

		configMap := getCreatedConfigMap(t, dynamicClient)
		assert.Contains(t, configMap.Data["config.yaml"], "externalProxyUrl")
	})

	t.Run("can create proxy with a user defined metric port", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.MetricPort = 1234
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)

		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsPortInContainers(t, 1234, "pushListenerPorts", dynamicClient)
		containsPortInServicePort(t, 1234, dynamicClient)

		configMap := getCreatedConfigMap(t, dynamicClient)
		assert.Contains(t, configMap.Data["config.yaml"], "wavefront-proxy:1234")
	})

	t.Run("can create proxy with a user defined delta counter port", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.DeltaCounterPort = 50000
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsPortInContainers(t, 50000, "deltaCounterPorts", dynamicClient)
		containsPortInServicePort(t, 50000, dynamicClient)
	})

	t.Run("can create proxy with a user defined Wavefront tracing", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.Port = 30000
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.SamplingRate = ".1"
		wfSpec.DataExport.WavefrontProxy.Tracing.Wavefront.SamplingDuration = 45

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsPortInContainers(t, 30000, "traceListenerPorts", dynamicClient)
		containsPortInServicePort(t, 30000, dynamicClient)
		containsProxyArg(t, "--traceSamplingRate .1", dynamicClient)
		containsProxyArg(t, "--traceSamplingDuration 45", dynamicClient)
	})

	t.Run("can create proxy with a user defined Jaeger distributed tracing", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.Port = 30001
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.GrpcPort = 14250
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.HttpPort = 30080
		wfSpec.DataExport.WavefrontProxy.Tracing.Jaeger.ApplicationName = "jaeger"

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsPortInContainers(t, 30001, "traceJaegerListenerPorts", dynamicClient)
		containsPortInServicePort(t, 30001, dynamicClient)
		containsPortInContainers(t, 14250, "traceJaegerGrpcListenerPorts", dynamicClient)
		containsPortInServicePort(t, 14250, dynamicClient)
		containsPortInContainers(t, 30080, "traceJaegerHttpListenerPorts", dynamicClient)
		containsPortInServicePort(t, 30080, dynamicClient)
		containsProxyArg(t, "--traceJaegerApplicationName jaeger", dynamicClient)
	})

	t.Run("can create proxy with a user defined ZipKin distributed tracing", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.Port = 9411
		wfSpec.DataExport.WavefrontProxy.Tracing.Zipkin.ApplicationName = "zipkin"

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsPortInContainers(t, 9411, "traceZipkinListenerPorts", dynamicClient)
		containsPortInServicePort(t, 9411, dynamicClient)
		containsProxyArg(t, "--traceZipkinApplicationName zipkin", dynamicClient)
	})

	t.Run("can create proxy with histogram ports enabled", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Histogram.Port = 40000
		wfSpec.DataExport.WavefrontProxy.Histogram.MinutePort = 40001
		wfSpec.DataExport.WavefrontProxy.Histogram.HourPort = 40002
		wfSpec.DataExport.WavefrontProxy.Histogram.DayPort = 40003

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsPortInContainers(t, 40000, "histogramDistListenerPorts", dynamicClient)
		containsPortInServicePort(t, 40000, dynamicClient)
		containsPortInContainers(t, 40001, "histogramMinuteListenerPorts", dynamicClient)
		containsPortInServicePort(t, 40001, dynamicClient)
		containsPortInContainers(t, 40002, "histogramHourListenerPorts", dynamicClient)
		containsPortInServicePort(t, 40002, dynamicClient)
		containsPortInContainers(t, 40003, "histogramDayListenerPorts", dynamicClient)
		containsPortInServicePort(t, 40003, dynamicClient)
	})

	t.Run("can create proxy with a user defined proxy args", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Args = "--prefix dev \r\n --customSourceTags mySource"

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsProxyArg(t, "--prefix dev", dynamicClient)
		containsProxyArg(t, "--customSourceTags mySource", dynamicClient)
	})

	t.Run("can create proxy with preprocessor rules", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.Preprocessor = "preprocessor-rules"

		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)
		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		containsProxyArg(t, "--preprocessorConfigFile /etc/wavefront/preprocessor/rules.yaml", dynamicClient)

		deployment := getCreatedDeployment(t, dynamicClient, "wavefront-proxy")
		volumeMountHasPath(t, deployment, "preprocessor", "/etc/wavefront/preprocessor")
		volumeHasConfigMap(t, deployment, "preprocessor", "preprocessor-rules")
	})

	t.Run("can create proxy with HTTP configurations", func(t *testing.T) {
		wfSpec := defaultWFSpec()
		wfSpec.DataExport.WavefrontProxy.HttpProxy.Secret = "testHttpProxySecret"
		r, _, _, dynamicClient, _ := setupForCreate(wfSpec)

		_, err := r.Reconcile(context.Background(), reconcile.Request{})
		assert.NoError(t, err)

		deployment := getCreatedDeployment(t, dynamicClient, "wavefront-proxy")

		value := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
		fmt.Println("value" + value)
		containsProxyArg(t, "--proxyHost 10.202.210.216", dynamicClient)
		containsProxyArg(t, "--proxyPort 8080", dynamicClient)
		containsProxyArg(t, "--proxyUser validUser", dynamicClient)
		containsProxyArg(t, "--proxyPassword validPassword123", dynamicClient)
		volumeMountHasPath(t, deployment, "http-proxy-ca", "/tmp/ca")
		volumeHasSecret(t, deployment, "http-proxy-ca", "http-proxy-secret")
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

func containsPortInServicePort(t *testing.T, port int32, dynamicClient *dynamicfake.FakeDynamicClient) {
	service := getCreatedService(t, dynamicClient)
	for _, servicePort := range service.Spec.Ports {
		if servicePort.Port == port {
			return
		}
	}
	assert.Fail(t, fmt.Sprintf("Did not find the port: %d", port))
}

func containsPortInContainers(t *testing.T, port int32, proxyArgName string, dynamicClient *dynamicfake.FakeDynamicClient) {
	deployment := getCreatedDeployment(t, dynamicClient, "wavefront-proxy")
	foundPort := false
	for _, containerPort := range deployment.Spec.Template.Spec.Containers[0].Ports {
		if containerPort.ContainerPort == port {
			foundPort = true
		}
	}
	assert.True(t, foundPort, fmt.Sprintf("Did not find the port: %d", port))
	value := getEnvValueForName(deployment.Spec.Template.Spec.Containers[0].Env, "WAVEFRONT_PROXY_ARGS")
	assert.Contains(t, value, fmt.Sprintf("--%s %d", proxyArgName, port))
}

func getEnvValueForName(envs []v1.EnvVar, name string) string {
	for _, envVar := range envs {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	return ""
}

func containsProxyArg(t *testing.T, proxyArg string, dynamicClient *dynamicfake.FakeDynamicClient) {
	deployment := getCreatedDeployment(t, dynamicClient, "wavefront-proxy")
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
	daemonSetObject := getCreateObject(dynamicClient, "daemonsets", "wavefront-node-collector")
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

func setupForCreate(spec wf.WavefrontSpec) (*controllers.WavefrontReconciler, *wf.Wavefront, client.WithWatch, *dynamicfake.FakeDynamicClient, typedappsv1.AppsV1Interface) {
	var wfCR = &wf.Wavefront{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       spec,
		Status:     wf.WavefrontStatus{},
	}

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
		StringData: map[string]string{
			"http-url":            "https://myproxyhost_url:8080",
			"basic-auth-username": "myUser",
			"basic-auth-password": "myPassword",
			"tls-root-ca-bundle":  "myCert",
		},
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
	clientBuilder = clientBuilder.WithScheme(s).WithObjects(wfCR, httpProxySecet).WithRESTMapper(testRestMapper)
	apiClient := clientBuilder.Build()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(s)

	fakesAppsV1 := k8sfake.NewSimpleClientset(&appsv1.Deployment{
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
	}).AppsV1()

	r := &controllers.WavefrontReconciler{
		Client:        apiClient,
		Scheme:        nil,
		FS:            os.DirFS(controllers.DeployDir),
		DynamicClient: dynamicClient,
		RestMapper:    apiClient.RESTMapper(),
		Appsv1:        fakesAppsV1,
	}
	return r, wfCR, apiClient, dynamicClient, fakesAppsV1
}

func setup(wavefrontUrl, wavefrontTokenSecret, clusterName string) (*wf.Wavefront, client.WithWatch, *dynamicfake.FakeDynamicClient, typedappsv1.AppsV1Interface) {
	wfSpec := defaultWFSpec()
	wfSpec.WavefrontUrl = wavefrontUrl
	wfSpec.WavefrontTokenSecret = wavefrontTokenSecret
	wfSpec.ClusterName = clusterName
	namespace := "wavefront"
	_, wfCR, apiClient, dynamicClient, fakesAppsV1 := setupForCreate(wfSpec)

	_ = dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "wavefront-proxy",
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
			"name":      "wavefront-node-collector",
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
			"name":      "wavefront-cluster-collector",
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
			"name":      "wavefront-proxy",
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

	return wfCR, apiClient, dynamicClient, fakesAppsV1
}
