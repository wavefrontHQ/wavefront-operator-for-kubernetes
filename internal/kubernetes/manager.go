package manager

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
)

type KubernetesManager struct {
	RestMapper    meta.RESTMapper
	DynamicClient dynamic.Interface
}

type ResourceAction func(resources []*unstructured.Unstructured) error

type Filter func(resource *unstructured.Unstructured) bool

type Transformer func(resources []*unstructured.Unstructured) []*unstructured.Unstructured

func FilterResources(filter Filter, resources []*unstructured.Unstructured) []*unstructured.Unstructured {
	var filtered []*unstructured.Unstructured
	for _, resource := range resources {
		if !filter(resource) {
			filtered = append(filtered, resource)
		}
	}
	return resources
}

func ParseResources(resourceYAMLs []string) ([]*unstructured.Unstructured, error) {
	var resources []*unstructured.Unstructured
	for _, resourceYAML := range resourceYAMLs {
		resource := &unstructured.Unstructured{}
		var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, _, err := resourceDecoder.Decode([]byte(resourceYAML), nil, resource)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// Responsibilities
// 1. Client Construction (RESTMapping)
// 2. Create or Update based on existence
func ApplyResources(restMapper meta.RESTMapper, dynamicClient dynamic.Interface) ResourceAction {
	return func(resourceYAMLs []*unstructured.Unstructured) error {
		var dynamicResourceClient dynamic.ResourceInterface
		for _, resource := range resourceYAMLs {
			gvk := resource.GroupVersionKind()
			mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return err
			}

			if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
				dynamicResourceClient = dynamicClient.Resource(mapping.Resource).Namespace(resource.GetNamespace())
			} else {
				dynamicResourceClient = dynamicClient.Resource(mapping.Resource)
			}

			_, err = dynamicResourceClient.Get(context.TODO(), resource.GetName(), v1.GetOptions{})
			if err != nil && errors.IsNotFound(err) {
				_, err = dynamicResourceClient.Create(context.TODO(), resource, v1.CreateOptions{})
				if err != nil {
					return err
				}
			} else if err == nil {
				data, err := json.Marshal(resource)
				if err != nil {
					return err
				}
				_, err = dynamicResourceClient.Patch(context.TODO(), resource.GetName(), types.MergePatchType, data, v1.PatchOptions{})
				if err != nil {
					return err
				}
			}
		}

		return nil
	}
}

// Responsibilities
// 1. Parsing
// 2. Filtering
// 3. Client Construction (RESTMapping)
// 4. Create or Update based on existence
func (km KubernetesManager) ApplyResources(resourceYamls []string, filterObject func(*unstructured.Unstructured) bool) error {
	var dynamicClient dynamic.ResourceInterface

	for _, resource := range resourceYamls {
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

func (km KubernetesManager) DeleteResources(resourceYamls []string) error {
	for _, resource := range resourceYamls {
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
