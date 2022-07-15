package controllers

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

type ResourceManager struct {
	FS            fs.FS
	RestMapper    meta.RESTMapper
	Appsv1        v1.AppsV1Interface
	DynamicClient dynamic.Interface
}

func NewResourceManager(f fs.FS, restMapper meta.RESTMapper, appsV1 v1.AppsV1Interface, dynamicClient dynamic.Interface) (resourceManager *ResourceManager) {
	resourceManager = &ResourceManager{
		FS:            f,
		RestMapper:    restMapper,
		Appsv1:        appsV1,
		DynamicClient: dynamicClient,
	}
	return resourceManager
}

// Read, Create, Update and Delete Resources.
func (rm *ResourceManager) readAndCreateResources(spec v1alpha1.WavefrontSpec) error {
	controllerManagerUID, err := rm.getControllerManagerUID()
	if err != nil {
		return err
	}
	spec.ControllerManagerUID = string(controllerManagerUID)

	resources, err := rm.readAndInterpolateResources(spec)
	if err != nil {
		return err
	}

	err = rm.createKubernetesObjects(resources, spec)
	if err != nil {
		return err
	}
	return nil
}

func (rm *ResourceManager) readAndInterpolateResources(spec v1alpha1.WavefrontSpec) ([]string, error) {
	var resources []string

	resourceFiles, err := resourceFiles("yaml")
	if err != nil {
		return nil, err
	}

	for _, resourceFile := range resourceFiles {
		resourceTemplate, err := newTemplate(resourceFile).ParseFS(rm.FS, resourceFile)
		if err != nil {
			return nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, spec)
		if err != nil {
			return nil, err
		}
		resources = append(resources, buffer.String())
	}
	return resources, nil
}

func (rm *ResourceManager) createKubernetesObjects(resources []string, wavefrontSpec v1alpha1.WavefrontSpec) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := rm.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
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

		err = rm.createResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rm *ResourceManager) createResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = rm.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = rm.DynamicClient.Resource(mapping.Resource)
	}

	_, err := dynamicClient.Get(context.TODO(), obj.GetName(), v12.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = dynamicClient.Create(context.TODO(), obj, v12.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err == nil {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = dynamicClient.Patch(context.TODO(), obj.GetName(), types.MergePatchType, data, v12.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return err
}

func (rm *ResourceManager) getControllerManagerUID() (types.UID, error) {
	deployment, err := rm.Appsv1.Deployments("wavefront").Get(context.Background(), "wavefront-controller-manager", v12.GetOptions{})
	if err != nil {
		return "", err
	}
	return deployment.UID, nil
}

func resourceFiles(suffix string) ([]string, error) {
	var files []string

	err := filepath.Walk(DeployDir,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, suffix) {
				files = append(files, info.Name())
			}
			return nil
		},
	)

	return files, err
}

func (rm *ResourceManager) readAndDeleteResources() error {
	resources, err := rm.readAndInterpolateResources(v1alpha1.WavefrontSpec{})
	if err != nil {
		return err
	}

	err = rm.deleteKubernetesObjects(resources)
	if err != nil {
		return err
	}
	return nil
}

func (rm *ResourceManager) deleteKubernetesObjects(resources []string) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := rm.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		err = rm.deleteResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rm *ResourceManager) deleteResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = rm.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = rm.DynamicClient.Resource(mapping.Resource)
	}
	_, err := dynamicClient.Get(context.TODO(), obj.GetName(), v12.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return dynamicClient.Delete(context.TODO(), obj.GetName(), v12.DeleteOptions{})
}
