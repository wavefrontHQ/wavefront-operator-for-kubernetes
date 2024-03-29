package kubernetes_manager

import (
	"context"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Client interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error

	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

type KubernetesManager struct {
	objClient Client
}

func NewKubernetesManager(objClient Client) *KubernetesManager {
	return &KubernetesManager{objClient: objClient}
}

func (km *KubernetesManager) ApplyResources(resourceYAMLs []string, exclude func(*unstructured.Unstructured) bool) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resourceYAML := range resourceYAMLs {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resourceYAML), nil, object)
		if err != nil {
			return err
		}

		if exclude(object) {
			continue
		}

		var getObj unstructured.Unstructured
		getObj.SetGroupVersionKind(*gvk)
		err = km.objClient.Get(context.Background(), util.ObjKey(object.GetNamespace(), object.GetName()), &getObj)
		if errors.IsNotFound(err) {
			err = km.objClient.Create(context.Background(), object)
		} else if err == nil {
			var diffObj unstructured.Unstructured
			diffObj.SetGroupVersionKind(*gvk)
			err = km.objClient.Patch(context.Background(), object, client.MergeFrom(&diffObj))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (km *KubernetesManager) DeleteResources(resourceYAMLs []string) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resourceYAML := range resourceYAMLs {
		object := &unstructured.Unstructured{}
		_, _, err := resourceDecoder.Decode([]byte(resourceYAML), nil, object)
		if err != nil {
			return err
		}

		err = km.objClient.Delete(context.Background(), object)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
