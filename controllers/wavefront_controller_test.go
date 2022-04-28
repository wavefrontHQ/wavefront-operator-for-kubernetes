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

	t.Run("creates proxy and service", func(t *testing.T) {
		apiClient, dynamicClient := setupCreate("testUrl", "testToken")

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
		assert.Equal(t, 4, len(dynamicClient.Actions()))
		assert.Equal(t, "services", dynamicClient.Actions()[1].GetResource().Resource)
		assert.Equal(t, "deployments", dynamicClient.Actions()[3].GetResource().Resource)

		deploymentObject := dynamicClient.Actions()[3].(testing2.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var deployment appsv1.Deployment

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)

		assert.NoError(t, err)
		assert.Equal(t, "testUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].Value)
	})

	t.Run("updates proxy and service", func(t *testing.T) {
		apiClient, dynamicClient := setupPatch("testUrl", "updatedToken")

		r := &controllers.WavefrontReconciler{
			Client:        apiClient,
			Scheme:        nil,
			FS:            os.DirFS("../deploy"),
			DynamicClient: dynamicClient,
			RestMapper:    apiClient.RESTMapper(),
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		// see that it's updated
		assert.NoError(t, err)
		//fmt.Println(err.Error())
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 4, len(dynamicClient.Actions()))
		assert.Equal(t, "services", dynamicClient.Actions()[0].GetResource().Resource)
		assert.Equal(t, "services", dynamicClient.Actions()[1].GetResource().Resource)

		deploymentObject := dynamicClient.Actions()[3].(testing2.PatchActionImpl).Patch

		//var deployment v1.Deployment
		//err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject., &deployment)
		assert.Contains(t, string(deploymentObject), "updatedToken")
		assert.Contains(t, string(deploymentObject), "testUrl/api/")

		assert.NoError(t, err)
		//assert.Equal(t, "testUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		//assert.Equal(t, "updatedToken", deployment.Spec.Template.Spec.Containers[0].Env[1].Value)
	})
}

func setupCreate(wavefrontUrl, wavefrontToken string) (client.WithWatch, *dynamicfake.FakeDynamicClient) {
	return setup(wavefrontUrl, wavefrontToken, "testProxy", "testNamespace")
}

func setupPatch(wavefrontUrl, wavefrontToken string) (client.WithWatch, *dynamicfake.FakeDynamicClient) {
	return setup(wavefrontUrl, wavefrontToken, "wavefront-proxy", "wavefront")
}

func setup(wavefrontUrl, wavefrontToken, wavefrontProxyName, namespace string) (client.WithWatch, *dynamicfake.FakeDynamicClient) {
	wf := &wavefrontcomv1alpha1.Wavefront{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       wavefrontcomv1alpha1.WavefrontSpec{WavefrontUrl: wavefrontUrl, WavefrontToken: wavefrontToken},
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
		Kind:    "Service",
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
		"spec": map[string]interface{}{
			"testSpec": "3",
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
				"app.kubernetes.io/name":      "wavefront-proxy",
				"app.kubernetes.io/component": "proxy",
			},
		},
		Spec: v1.ServiceSpec{
			Type: "CLusterIP",
		},
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClient(
		s,
		deployment,
		service,
	)
	return apiClient, dynamicClient
}
