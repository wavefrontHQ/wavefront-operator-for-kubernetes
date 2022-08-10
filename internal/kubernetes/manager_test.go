package kubernetes_manager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kubernetes_manager "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/kubernetes"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fake2 "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	testing2 "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetesManager(t *testing.T) {
	t.Run("creates or updates kubernetes objects with resource yaml strings", func(t *testing.T) {
		fakeServiceYaml := `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: fake-app-kubernetes-name
  name: fake-name
  namespace: fake-namespace
spec:
  ports:
  - name: fake-port-name
    port: 1111
    protocol: TCP
  selector:
    app.kubernetes.io/name: fake-app-kubernetes-name
  type: ClusterIP
`
		fakeYamls := []string{
			fakeServiceYaml,
			fakeServiceYaml, // duplicated to cause a patch
		}

		testRestMapper := meta.NewDefaultRESTMapper(
			[]schema.GroupVersion{
				{Group: "apps", Version: "v1"},
			},
		)
		testRestMapper.Add(schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Service",
		}, meta.RESTScopeNamespace)

		clientBuilder := fake.NewClientBuilder()
		clientBuilder = clientBuilder.WithRESTMapper(testRestMapper)

		fakeApiClient := clientBuilder.Build()

		s := scheme.Scheme
		fakeDynamicClient := fake2.NewSimpleDynamicClient(s)

		km, err := kubernetes_manager.NewKubernetesManager(
			fakeApiClient.RESTMapper(),
			fakeDynamicClient,
		)
		assert.NoError(t, err)

		err = km.ApplyResources(fakeYamls, func(obj *unstructured.Unstructured) bool {
			return false
		})
		assert.NoError(t, err)

		// TODO: action count

		assert.True(t, hasAction(fakeDynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(fakeDynamicClient, "create", "services"), "create Service")
		assert.True(t, hasAction(fakeDynamicClient, "patch", "services"), "patch Service")
	})

	t.Run("deletes multiple kubernetes objects with resource yaml strings if they exist", func(t *testing.T) {
		fakeServiceYaml := `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: fake-app-kubernetes-name
  name: fake-name
  namespace: fake-namespace
spec:
  type: ClusterIP
`
		fakeMissingDeploymentYaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
 labels:
   app.kubernetes.io/component: fake-app-kubernetes-component
 name: fake-name
 namespace: fake-namespace
spec:
 replicas: 1
 selector:
   matchLabels:
     app.kubernetes.io/component: fake-app-kubernetes-component
`
		fakeDaemonsetYaml := `
apiVersion: apps/v1
kind: DaemonSet
metadata:
 labels:
   app.kubernetes.io/name: fake-app-kubernetes-name
   app.kubernetes.io/component: fake-app-kubernetes-component
 name: fake-daemonset-name
 namespace: fake-namespace
spec:
 selector:
   matchLabels:
     app.kubernetes.io/name: fake-app-kubernetes-name
     app.kubernetes.io/component: fake-app-kubernetes-component
`

		fakeYamls := []string{
			fakeServiceYaml,
			fakeMissingDeploymentYaml,
			fakeDaemonsetYaml,
		}

		testRestMapper := meta.NewDefaultRESTMapper(
			[]schema.GroupVersion{
				{Group: "apps", Version: "v1"},
			},
		)
		testRestMapper.Add(schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Service",
		}, meta.RESTScopeNamespace)
		testRestMapper.Add(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		}, meta.RESTScopeNamespace)
		testRestMapper.Add(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		}, meta.RESTScopeNamespace)

		clientBuilder := fake.NewClientBuilder()
		clientBuilder = clientBuilder.WithRESTMapper(testRestMapper)

		fakeApiClient := clientBuilder.Build()

		s := scheme.Scheme
		fakeDynamicClient := fake2.NewSimpleDynamicClient(s)
		_ = fakeDynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "fake-name",
				"namespace": "fake-namespace",
				"labels": map[string]interface{}{
					"app.kubernetes.io/name": "fake-app-kubernetes-name",
				},
			},
			"spec": map[string]interface{}{
				"type": "ClusterIP",
			},
		}})
		_ = fakeDynamicClient.Tracker().Add(&unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "DaemonSet",
			"metadata": map[string]interface{}{
				"name":      "fake-daemonset-name",
				"namespace": "fake-namespace",
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":      "fake-app-kubernetes-name",
					"app.kubernetes.io/component": "fake-app-kubernetes-component",
				},
			},
		}})

		km, err := kubernetes_manager.NewKubernetesManager(
			fakeApiClient.RESTMapper(),
			fakeDynamicClient,
		)
		assert.NoError(t, err)

		err = km.DeleteResources(fakeYamls)
		assert.NoError(t, err)
		// TODO: action count

		assert.True(t, hasAction(fakeDynamicClient, "get", "services"), "get Service")
		assert.True(t, hasAction(fakeDynamicClient, "delete", "services"), "delete Service")
		assert.True(t, hasAction(fakeDynamicClient, "get", "deployments"), "get Deployment")

		// Notice the 'False'; deployment didn't exist
		assert.False(t, hasAction(fakeDynamicClient, "delete", "deployments"), "delete Deployment")

		assert.True(t, hasAction(fakeDynamicClient, "get", "daemonsets"), "get DaemonSet")
		assert.True(t, hasAction(fakeDynamicClient, "delete", "daemonsets"), "delete DaemonSet")
	})
}

func hasAction(dynamicClient *fake2.FakeDynamicClient, verb, resource string) (result bool) {
	if getAction(dynamicClient, verb, resource) != nil {
		return true
	}
	return false
}

func getAction(dynamicClient *fake2.FakeDynamicClient, verb, resource string) (action testing2.Action) {
	for _, action := range dynamicClient.Actions() {
		if action.GetVerb() == verb && action.GetResource().Resource == resource {
			return action
		}
	}
	return nil
}
