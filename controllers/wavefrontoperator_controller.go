/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const DeployDir = "../deploy"

// WavefrontOperatorReconciler reconciles a WavefrontOperator object
type WavefrontOperatorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	FS            fs.FS
	DynamicClient dynamic.Interface
	RestMapper    meta.RESTMapper
}

//+kubebuilder:rbac:groups=wavefront.com,resources=wavefrontoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wavefront.com,resources=wavefrontoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wavefront.com,resources=wavefrontoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WavefrontOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *WavefrontOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO: write separate story shut down collector and proxy if Operator is being shut down?

	wavefrontOperator := &wavefrontcomv1.WavefrontOperator{}
	err := r.Client.Get(ctx, req.NamespacedName, wavefrontOperator)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.readAndCreateResources(wavefrontOperator.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WavefrontOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wavefrontcomv1.WavefrontOperator{}).
		Complete(r)
}

func NewWavefrontOperatorReconciler(client client.Client, scheme *runtime.Scheme) (operator *WavefrontOperatorReconciler, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &WavefrontOperatorReconciler{
		Client:        client,
		Scheme:        scheme,
		FS:            os.DirFS(DeployDir),
		DynamicClient: dynamicClient,
		RestMapper:    mapper,
	}, nil
}

func (r *WavefrontOperatorReconciler) readAndCreateResources(spec wavefrontcomv1.WavefrontOperatorSpec) error {

	resources, err := r.readAndInterpolateResources(spec)
	if err != nil {
		return err
	}

	err = r.createKubernetesObjects(resources)
	if err != nil {
		return err
	}
	return nil
}

func (r *WavefrontOperatorReconciler) readAndInterpolateResources(spec wavefrontcomv1.WavefrontOperatorSpec) ([]string, error) {
	var resources []string

	resourceFiles, err := r.resourceFiles()
	if err != nil {
		return nil, err
	}

	for _, resourceFile := range resourceFiles {
		resourceTemplate, err := template.ParseFS(r.FS, resourceFile)
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

func (r *WavefrontOperatorReconciler) createKubernetesObjects(resources []string) error {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := r.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		err = r.createResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *WavefrontOperatorReconciler) createResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	var dynamicClient dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dynamicClient = r.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dynamicClient = r.DynamicClient.Resource(mapping.Resource)
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
		_, err = dynamicClient.Patch(context.TODO(), obj.GetName(), types.StrategicMergePatchType, data, v1.PatchOptions{})
		if err != nil {
			return err
		}
	}
	return err
}

func (r *WavefrontOperatorReconciler) resourceFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(DeployDir,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
				files = append(files, info.Name())
			}
			return nil
		},
	)

	return files, err
}
