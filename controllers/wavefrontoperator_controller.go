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
	"fmt"
	"io/fs"
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
	"path/filepath"
	"strings"
	"text/template"

	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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

// TODO: Functional e2e test "manually" for now to check operator deploying proxy on kind.
func (r *WavefrontOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// read in collector and proxy
	// read in specific config from Operator CRD instance,
	// pass in all vars as template variables to list of template files (.templ?) being filled
	// generically create or update them to the API
	// shut down collector and proxy if Operator is being shut down?

	wavefrontOperator := &wavefrontcomv1.WavefrontOperator{}
	err := r.Client.Get(ctx, req.NamespacedName, wavefrontOperator)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.ProvisionProxy(wavefrontOperator.Spec)

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

func (r *WavefrontOperatorReconciler) ProvisionProxy(spec wavefrontcomv1.WavefrontOperatorSpec) error {
	resourceFiles, err := ResourceFiles("../deploy")
	if err != nil {
		return err
	}
	resources, err := ReadAndInterpolateResources(r.FS, spec, resourceFiles)
	if err != nil {
		return err
	}

	//TODO:  we believe that the below code needs to be refactored to return unstructured
	// object and gvk to be able to create it using a RESTMapper.
	// Refer to https://gitlab.eng.vmware.com/tobs-k8s-group/tmc-wavefront-operator/-/blob/master/actions.go#L248
	err = r.InitializeKubernetesObjects(resources)
	if err != nil {
		return err
	}

	// TODO: Create k8s objects using RESTMapper client
	return nil
}

func ReadAndInterpolateResources(fileSystem fs.FS, spec wavefrontcomv1.WavefrontOperatorSpec, resourceFiles []string) ([]string, error) {
	var resources []string
	for _, resourceFile := range resourceFiles {
		fmt.Printf("resourceFile %+v \n", resourceFile)
		fmt.Printf("fileSystem %+v", fileSystem)
		resourceTemplate, err := template.ParseFS(fileSystem, resourceFile)
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

func (r *WavefrontOperatorReconciler) InitializeKubernetesObjects(resources []string) error {

	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, gvk, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return err
		}

		mapping, err := r.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)

		err = r.CreateResources(mapping, object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *WavefrontOperatorReconciler) CreateResources(mapping *meta.RESTMapping, obj *unstructured.Unstructured) error {
	//TODO Rest mapping

	var client dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		client = r.DynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		client = r.DynamicClient.Resource(mapping.Resource)
	}

	_, err := client.Get(context.TODO(), obj.GetName(), v1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = client.Create(context.TODO(), obj, v1.CreateOptions{})
	} else if err == nil {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = client.Patch(context.TODO(), obj.GetName(), types.StrategicMergePatchType, data, v1.PatchOptions{})
	}
	return err
}

// TODO: Change ResourceFiles to take fs.FS instead of the dir name
func ResourceFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir,
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
func NewRESTMapper(cfg *rest.Config) (*restmapper.DeferredDiscoveryRESTMapper, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc)), nil
}
