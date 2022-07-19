package controllers

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

func (km KubernetesManager) CreateOrUpdateResources(resourceYamls []string, filterObject func(*unstructured.Unstructured) bool) error {
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

func (km KubernetesManager) deleteObjects(resources []string) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := km.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		err = km.deleteResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (km KubernetesManager) deleteResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = km.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = km.DynamicClient.Resource(mapping.Resource)
	}
	_, err := dynamicClient.Get(context.TODO(), obj.GetName(), v1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return dynamicClient.Delete(context.TODO(), obj.GetName(), v1.DeleteOptions{})
}
