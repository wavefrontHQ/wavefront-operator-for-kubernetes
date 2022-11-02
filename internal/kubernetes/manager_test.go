package kubernetes_manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
	kubernetes_manager "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const fakeServiceYAML = `
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

const fakeServiceUpdatedYAML = `
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
    port: 1112
    protocol: TCP
  selector:
    app.kubernetes.io/name: fake-app-kubernetes-name
  type: ClusterIP
`

const otherFakeServiceYAML = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: other-fake-app-kubernetes-name
  name: other-fake-name
  namespace: fake-namespace
spec:
  ports:
  - name: fake-port-name
    port: 1111
    protocol: TCP
  selector:
    app.kubernetes.io/name: other-fake-app-kubernetes-name
  type: ClusterIP
`

func TestKubernetesManager(t *testing.T) {
	t.Run("applying", func(t *testing.T) {
		t.Run("rejects invalid objects", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			err := km.ApplyResources([]string{"invalid: {{object}}"}, excludeNothing)

			assert.ErrorContains(t, err, "yaml: invalid")
		})

		t.Run("creates kubernetes objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			assert.NoError(t, km.ApplyResources([]string{fakeServiceYAML}, excludeNothing))

			require.NoError(t, objClient.Get(context.Background(), types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			}, &corev1.Service{}))
		})

		t.Run("patches kubernetes objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			err := km.ApplyResources([]string{fakeServiceYAML, fakeServiceUpdatedYAML}, excludeNothing)
			assert.NoError(t, err)

			var service corev1.Service
			require.NoError(t, objClient.Get(context.Background(), types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			}, &service))

			require.Equal(t, int32(1112), service.Spec.Ports[0].Port)
		})

		t.Run("filters objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			err := km.ApplyResources([]string{fakeServiceYAML}, excludeEverything)
			assert.NoError(t, err)

			require.Error(t, objClient.Get(context.Background(), types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			}, &corev1.Service{}))
		})

		t.Run("reports client errors", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(&errClient{errors.New("some error")})

			err := km.ApplyResources([]string{fakeServiceYAML}, excludeNothing)
			assert.Error(t, err)
		})
	})

	t.Run("deleting", func(t *testing.T) {
		t.Run("rejects invalid objects", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			err := km.DeleteResources([]string{"invalid: {{object}}"})

			assert.ErrorContains(t, err, "yaml: invalid")
		})

		t.Run("deletes objects that exist", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			_ = km.ApplyResources([]string{fakeServiceYAML}, excludeNothing)

			require.NoError(t, km.DeleteResources([]string{fakeServiceYAML}))

			assert.Error(t, objClient.Get(context.Background(), types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			}, &corev1.Service{}))

		})

		t.Run("reports client errors", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(&errClient{errors.New("some error")})

			assert.Error(t, km.DeleteResources([]string{fakeServiceYAML}))
		})

		t.Run("does not return an error for objects that do not exist", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			require.NoError(t, km.DeleteResources([]string{fakeServiceYAML}))
		})

		t.Run("deletes all resources", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			_ = km.ApplyResources([]string{fakeServiceYAML, otherFakeServiceYAML}, excludeNothing)

			require.NoError(t, km.DeleteResources([]string{fakeServiceYAML, otherFakeServiceYAML}))

			assert.Error(t, objClient.Get(context.Background(), types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "fake-name",
			}, &corev1.Service{}))

			assert.Error(t, objClient.Get(context.Background(), types.NamespacedName{
				Namespace: "fake-namespace",
				Name:      "other-fake-name",
			}, &corev1.Service{}))
		})
	})
}

func excludeEverything(_ *unstructured.Unstructured) bool {
	return true
}

func excludeNothing(_ *unstructured.Unstructured) bool {
	return false
}

type errClient struct {
	err error
}

func (c *errClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object) error {
	return c.err
}

func (c *errClient) Create(_ context.Context, _ client.Object, _ ...client.CreateOption) error {
	return c.err
}

func (c *errClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
	return c.err
}

func (c *errClient) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	return c.err
}
