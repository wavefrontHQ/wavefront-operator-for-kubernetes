package controllers_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/controllers"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	testing2 "k8s.io/client-go/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	wf := &wavefrontcomv1.WavefrontOperator{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       wavefrontcomv1.WavefrontOperatorSpec{WavefrontUrl: "testUrl", WavefrontToken: "testToken"},
		Status:     wavefrontcomv1.WavefrontOperatorStatus{},
	}
	s := scheme.Scheme
	s.AddKnownTypes(wavefrontcomv1.GroupVersion, wf)

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
	client := clientBuilder.Build()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(
		runtime.NewScheme(),
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "testProxy",
					"namespace": "testNamespace",
				},
				"spec": map[string]interface{}{
					"testSpec": "3",
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name":      "testName",
					"namespace": "testNamespace",
				},
				"spec": map[string]interface{}{
					"testSpec": "3",
				},
			},
		},
	)

	t.Run("creates proxy and service", func(t *testing.T) {
		r := &controllers.WavefrontOperatorReconciler{
			Client:        client,
			Scheme:        nil,
			FS:            os.DirFS("../deploy"),
			DynamicClient: dynamicClient,
			RestMapper:    client.RESTMapper(),
		}
		results, err := r.Reconcile(context.Background(), reconcile.Request{})

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, results)
		assert.Equal(t, 4, len(dynamicClient.Actions()))
		assert.Equal(t, "services", dynamicClient.Actions()[1].GetResource().Resource)
		assert.Equal(t, "deployments", dynamicClient.Actions()[3].GetResource().Resource)

		deploymentObject := dynamicClient.Actions()[3].(testing2.CreateActionImpl).GetObject().(*unstructured.Unstructured)
		var deployment v1.Deployment
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentObject.Object, &deployment)

		assert.NoError(t, err)
		assert.Equal(t, "testUrl/api/", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
		assert.Equal(t, "testToken", deployment.Spec.Template.Spec.Containers[0].Env[1].Value)
	})
}
