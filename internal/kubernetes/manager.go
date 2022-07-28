package manager

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
)

type KubernetesManager interface {
	ApplyResources(resourceYamls []string, filterObject func(*unstructured.Unstructured) bool) error
	DeleteResources(resourceYamls []string) error
}

func NewKubernetesManager(mapper meta.RESTMapper, dynamicClient dynamic.Interface) (KubernetesManager, error) {
	return &kubernetesManager{
		RestMapper:    mapper,
		DynamicClient: dynamicClient,
	}, nil
}

type kubernetesManager struct {
	RestMapper    meta.RESTMapper
	DynamicClient dynamic.Interface
}

func (km kubernetesManager) ApplyResources(resourceYAMLs []string, filterObject func(*unstructured.Unstructured) bool) error {
	var dynamicClient dynamic.ResourceInterface

	for _, resource := range resourceYAMLs {
		object := &unstructured.Unstructured{}
		var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		if filterObject(object) {
			continue
		}

		mapping, err := km.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			dynamicClient = km.DynamicClient.Resource(mapping.Resource).Namespace(object.GetNamespace())
		} else {
			dynamicClient = km.DynamicClient.Resource(mapping.Resource)
		}

		_, err = dynamicClient.Get(context.TODO(), object.GetName(), v1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			_, err = dynamicClient.Create(context.TODO(), object, v1.CreateOptions{})
			if err != nil {
				return err
			}
		} else if err == nil {
			data, err := json.Marshal(object)
			if err != nil {
				return err
			}
			_, err = dynamicClient.Patch(context.TODO(), object.GetName(), types.MergePatchType, data, v1.PatchOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (km kubernetesManager) DeleteResources(resourceYAMLs []string) error {
	for _, resource := range resourceYAMLs {
		var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := km.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dynamicClient dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			dynamicClient = km.DynamicClient.Resource(mapping.Resource).Namespace(object.GetNamespace())
		} else {
			dynamicClient = km.DynamicClient.Resource(mapping.Resource)
		}

		_, err = dynamicClient.Get(context.TODO(), object.GetName(), v1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		err = dynamicClient.Delete(context.TODO(), object.GetName(), v1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
