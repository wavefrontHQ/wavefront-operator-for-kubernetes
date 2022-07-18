package controllers

import (
	"context"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
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

func (km KubernetesManager) CreateOrUpdateFromYamls(yamls []string) error {
	var dynamicClient dynamic.ResourceInterface

	resource := yamls[0]
	object := &unstructured.Unstructured{}
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
	if err != nil {
		return err
	}

	mapping, err := km.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	dynamicClient = km.DynamicClient.Resource(mapping.Resource)

	_, err = dynamicClient.Create(context.TODO(), &unstructured.Unstructured{}, v1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (km KubernetesManager) createObjects(resources []string, wavefrontSpec v1alpha1.WavefrontSpec) error {
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

		objLabels := object.GetLabels()
		if labelVal, _ := objLabels["app.kubernetes.io/component"]; labelVal == "collector" && !wavefrontSpec.DataCollection.Metrics.Enable {
			continue
		}
		if labelVal, _ := objLabels["app.kubernetes.io/component"]; labelVal == "proxy" && !wavefrontSpec.DataExport.WavefrontProxy.Enable {
			continue
		}
		if object.GetKind() == "ConfigMap" && wavefrontSpec.DataCollection.Metrics.CollectorConfigName != object.GetName() {
			continue
		}

		err = km.createResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (km KubernetesManager) createResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = km.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = km.DynamicClient.Resource(mapping.Resource)
	}

	_, err := dynamicClient.Get(context.TODO(), obj.GetName(), v1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = dynamicClient.Create(context.TODO(), obj, v1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err == nil {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = dynamicClient.Patch(context.TODO(), obj.GetName(), types.MergePatchType, data, v1.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return err
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
