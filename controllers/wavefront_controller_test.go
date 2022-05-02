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
	"k8s.io/client-go/kubernetes/scheme"
	testing2 "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {

	t.Run("creates proxy, proxy service, collector and collector service", func(t *testing.T) {
		apiClient, dynamicClient := setup("testUrl", "testToken", "proxyName", "testClusterName", "testNameSpace")

		r := &controllers.WavefrontReconciler{

			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS("../deploy"),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 10, len(dynamicClient.Actions()))
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

		deploymentObject := getAction(dynamicClient, "create", "deployments").(testing2.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var deployment appsv1.Deployment

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)

		assert.NoError(t, err)
		assert.Equal(t, "testUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].Value)

		configMapObject := getAction(dynamicClient, "create", "configmaps").(testing2.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var configMap v1.ConfigMap

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(configMapObject.Object, &configMap)

		assert.NoError(t, err)
		assert.Contains(t, configMap.Data["config.yaml"], "testClusterName")
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		apiClient, dynamicClient := setup("testUrl", "updatedToken", "wavefront-proxy", "testClusterName", "wavefront")

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS("../deploy"),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)

		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 10, len(dynamicClient.Actions()))

		deploymentObject := getAction(dynamicClient, "patch", "deployments").(testing2.PatchActionImpl).Patch

		assert.Contains(t, string(deploymentObject), "updatedToken")
		assert.Contains(t, string(deploymentObject), "testUrl/api/")

		assert.NoError(t, err)
	})
}

func hasAction(dynamicClient *dynamicfake.FakeDynamicClient, verb, resource string) (result bool) {
	if getAction(dynamicClient, verb, resource) != nil {
		return true
	}
	return false
}

func getAction(dynamicClient *dynamicfake.FakeDynamicClient, verb, resource string) (action testing2.Action) {
	for _, action := range dynamicClient.Actions() {
		if action.GetVerb() == verb && action.GetResource().Resource == resource {
			return action
		}
	}
	return nil
}

func setup(wavefrontUrl, wavefrontToken, wavefrontProxyName, clusterName, namespace string) (client.WithWatch, *dynamicfake.FakeDynamicClient) {
	wf := &wavefrontcomv1alpha1.Wavefront{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       wavefrontcomv1alpha1.WavefrontSpec{WavefrontUrl: wavefrontUrl, WavefrontToken: wavefrontToken, ClusterName: clusterName},
		Status:     wavefrontcomv1alpha1.WavefrontStatus{},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Service{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
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

	deployment := &unstructured.Unstructured{}
	deployment.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      wavefrontProxyName,
			"namespace": namespace,
		},
	})

	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wavefrontProxyName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "proxy",
			},
		},
		Spec: v1.ServiceSpec{
			Type: "CLusterIP",
		},
	}

	daemonSet := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDaemonSet",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
	}

	configMap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      wavefrontProxyName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "collector",
			},
		},
		Data: map[string]string{
			"config.yaml": "foo",
		},
	}

	serviceAccount := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront-collector",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront-collector",
				"app.kubernetes.io/component": "collector",
			},
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(
		s,
		deployment,
		configMap,
		daemonSet,
		service,
		serviceAccount,
	)
	return apiClient, dynamicClient
}
