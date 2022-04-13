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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	Scheme *runtime.Scheme
	FS     fs.FS
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
	err1 := r.Client.Get(ctx, req.NamespacedName, wavefrontOperator)
	if err1 != nil {
		panic(err1.Error())
	}

	err := r.ProvisionProxy(wavefrontOperator.Spec, CreateResources)

	if err != nil {
		panic(err.Error())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WavefrontOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wavefrontcomv1.WavefrontOperator{}).
		Complete(r)
}

func (r *WavefrontOperatorReconciler) ProvisionProxy(spec wavefrontcomv1.WavefrontOperatorSpec, createResource func([]unstructured.Unstructured) (error) ) error {
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
	objects, err := InitializeKubernetesObjects(resources)
	if err != nil {
		return err
	}
	createResource(objects)
	//err = CreateResources(objects)
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

func InitializeKubernetesObjects(resources []string) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resource := range resources {
		object := &unstructured.Unstructured{}
		_, _, err := resourceDecoder.Decode([]byte(resource), nil, object)
		if err != nil {
			return nil, err
		}
		objects = append(objects, *object)
	}
	return objects, nil
}

func CreateResources(objects []unstructured.Unstructured) (error) {
	//TODO Rest mapping
	return nil
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
