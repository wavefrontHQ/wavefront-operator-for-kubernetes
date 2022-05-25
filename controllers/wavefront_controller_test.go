package controllers_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	wavefrontcomv1alpha1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
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

func TestReconcile(t *testing.T) {

	t.Run("creates proxy, proxy service, collector and collector service", func(t *testing.T) {
		_, apiClient, dynamicClient, fakeAppsV1 := setupForCreate(wavefrontcomv1alpha1.WavefrontSpec{
			CollectorEnabled:      true,
			ProxyUrl:              "testProxyUrl",
			WavefrontProxyEnabled: true,
			WavefrontUrl:          "testWavefrontUrl",
			WavefrontTokenSecret:  "testToken",
			ClusterName:           "testClusterName",
			ControllerManagerUID:  "",
		})

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakeAppsV1,
		}
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

		deploymentObject := getCreateObject(dynamicClient, "deployments", "wavefront-proxy")
		var deployment appsv1.Deployment
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)
		assert.NoError(t, err)
		assert.Equal(t, "testWavefrontUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name)

		configMapObject := getAction(dynamicClient, "create", "configmaps").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var configMap v1.ConfigMap

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(configMapObject.Object, &configMap)

		assert.NoError(t, err)
		assert.Contains(t, configMap.Data["config.yaml"], "testClusterName")
		assert.Contains(t, configMap.Data["config.yaml"], "wavefront-proxy:2878")

	})

	t.Run("resources set for cluster collector", func(t *testing.T) {
		_, apiClient, dynamicClient, fakeAppsV1 := setupForCreate(wavefrontcomv1alpha1.WavefrontSpec{
			CollectorEnabled:      true,
			ProxyUrl:              "testProxyUrl",
			WavefrontProxyEnabled: true,
			WavefrontUrl:          "testWavefrontUrl",
			WavefrontTokenSecret:  "testToken",
			ClusterName:           "testClusterName",
			ControllerManagerUID:  "",
			Metrics: wavefrontcomv1alpha1.Metrics{
				Cluster: wavefrontcomv1alpha1.Collector{
					Resources: wavefrontcomv1alpha1.Resources{
						Requests: wavefrontcomv1alpha1.Resource{
							CPU:    "200m",
							Memory: "10Mi",
						},
						Limits: wavefrontcomv1alpha1.Resource{
							CPU:    "200m",
							Memory: "256Mi",
						},
					},
				},
			},
		})

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakeAppsV1,
		}
		_, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)

		deploymentObject := getCreateObject(dynamicClient, "deployments", "wavefront-cluster-collector")
		var deployment appsv1.Deployment
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)
		assert.NoError(t, err)
		assert.Equal(t, "10Mi", deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		_, apiClient, dynamicClient, fakesAppsV1 := setup("testWavefrontUrl", "updatedToken", "wavefront-proxy", "wavefront-collector-config", "wavefront-collector", "testClusterName", "wavefront")

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

		deploymentObject := getAction(dynamicClient, "patch", "deployments").(clientgotesting.PatchActionImpl).Patch

		assert.Contains(t, string(deploymentObject), "updatedToken")
		assert.Contains(t, string(deploymentObject), "testWavefrontUrl/api/")

		assert.NoError(t, err)
	})

	t.Run("delete CRD should delete resources", func(t *testing.T) {
		wf, apiClient, dynamicClient, fakesAppsV1 := setup("testWavefrontUrl", "updatedToken", "wavefront-proxy", "wavefront-collector-config", "wavefront-collector", "testClusterName", "wavefront")
		apiClient.Delete(context.Background(), wf)

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakesAppsV1,
		}
		_, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, 11, len(dynamicClient.Actions()))

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

	t.Run("Skip creating collector if collectorEnabled is set to false", func(t *testing.T) {
		_, apiClient, dynamicClient, fakeAppsV1 := setupForCreate(wavefrontcomv1alpha1.WavefrontSpec{
			CollectorEnabled:      false,
			ProxyUrl:              "",
			WavefrontProxyEnabled: true,
			WavefrontUrl:          "testWavefrontUrl",
			WavefrontTokenSecret:  "testToken",
			ClusterName:           "testClusterName",
			ControllerManagerUID:  "",
		})

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakeAppsV1,
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 4, len(dynamicClient.Actions()))
		assert.True(t, hasAction(dynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(dynamicClient, "create", "services"), "create Service")
		assert.True(t, hasAction(dynamicClient, "get", "deployments"), "get Deployment")
		assert.True(t, hasAction(dynamicClient, "create", "deployments"), "create Deployment")

		deploymentObject := getAction(dynamicClient, "create", "deployments").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var deployment appsv1.Deployment

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)

		assert.NoError(t, err)
		assert.Equal(t, "testWavefrontUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name)

	})

	t.Run("Skip creating proxy if WavefrontProxyEnabled is set to false", func(t *testing.T) {
		_, apiClient, dynamicClient, fakeAppsV1 := setupForCreate(wavefrontcomv1alpha1.WavefrontSpec{
			CollectorEnabled:      true,
			ProxyUrl:              "testProxyUrl",
			WavefrontProxyEnabled: false,
			WavefrontUrl:          "testWavefrontUrl",
			WavefrontTokenSecret:  "testToken",
			ClusterName:           "testClusterName",
			ControllerManagerUID:  "",
		})

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS(controllers.DeployDir),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
			Appsv1:        fakeAppsV1,
		}
		_, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, 8, len(dynamicClient.Actions()))

		assert.True(t, hasAction(dynamicClient, "get", "serviceaccounts"), "get ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "create", "serviceaccounts"), "create ServiceAccount")
		assert.True(t, hasAction(dynamicClient, "get", "configmaps"), "get ConfigMap")
		assert.True(t, hasAction(dynamicClient, "create", "configmaps"), "create Configmap")
		assert.True(t, hasAction(dynamicClient, "get", "daemonsets"), "get DaemonSet")
		assert.True(t, hasAction(dynamicClient, "create", "daemonsets"), "create DaemonSet")

		configMapObject := getAction(dynamicClient, "create", "configmaps").(clientgotesting.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var configMap v1.ConfigMap

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(configMapObject.Object, &configMap)

		assert.NoError(t, err)
		assert.Contains(t, configMap.Data["config.yaml"], "testProxyUrl")
	})
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

func setupForCreate(spec wavefrontcomv1alpha1.WavefrontSpec) (*wavefrontcomv1alpha1.Wavefront, client.WithWatch, *dynamicfake.FakeDynamicClient, typedappsv1.AppsV1Interface) {
	var wf = &wavefrontcomv1alpha1.Wavefront{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       spec,
		Status:     wavefrontcomv1alpha1.WavefrontStatus{},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Service{})
	s.AddKnownTypes(wavefrontcomv1alpha1.GroupVersion, wf)

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
	clientBuilder = clientBuilder.WithScheme(s).WithObjects(wf).WithRESTMapper(testRestMapper)
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

	return wf, apiClient, dynamicClient, fakesAppsV1
}

func setup(wavefrontUrl, wavefrontTokenSecret, proxyName, collectorConfigName, collectorName, clusterName, namespace string) (*wavefrontcomv1alpha1.Wavefront, client.WithWatch, *dynamicfake.FakeDynamicClient, typedappsv1.AppsV1Interface) {
	wf, apiClient, dynamicClient, fakesAppsV1 := setupForCreate(wavefrontcomv1alpha1.WavefrontSpec{
		CollectorEnabled:      true,
		ProxyUrl:              "",
		WavefrontProxyEnabled: true,
		WavefrontUrl:          wavefrontUrl,
		WavefrontTokenSecret:  wavefrontTokenSecret,
		ClusterName:           clusterName,
		ControllerManagerUID:  "",
	})

	dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      proxyName,
			"namespace": namespace,
		},
	}})
	dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      collectorConfigName,
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
	dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "DaemonSet",
		"metadata": map[string]interface{}{
			"name":      collectorName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
	}})
	dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      proxyName,
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
	dynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ServiceAccount",
		"metadata": map[string]interface{}{
			"name":      collectorName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
	}})

	return wf, apiClient, dynamicClient, fakesAppsV1
}
