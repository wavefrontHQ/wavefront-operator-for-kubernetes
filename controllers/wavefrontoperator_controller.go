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
	"path/filepath"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wavefrontcomv1 "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1"
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
func (r *WavefrontOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// read in collector and proxy
	// read in specific config from Operator CRD instance,
	// pass in all vars as template variables to list of template files (.templ?) being filled
	// generically create or update them to the API
	// shut down collector and proxy if Operator is being shut down?

	err := r.provisionProxy(req)

	if err != nil {
		fmt.Println(err.Error())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WavefrontOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wavefrontcomv1.WavefrontOperator{}).
		Complete(r)
}

func (r *WavefrontOperatorReconciler) provisionProxy(req ctrl.Request) error {
	spec := wavefrontcomv1.WavefrontOperatorSpec{
		ClusterName:    "fake-cluster-name",
		WavefrontToken: "fake-token",
	}
	resourceFiles, _ := ResourceFiles("./deploy")
	ReadAndInterpolateResources(r.FS, spec, resourceFiles)

	// TODO turn those resources into valid k8s objects
	// TODO call k8s api to provision proxy resources using k8s objects
	return nil
}

func ReadAndInterpolateResources(f fs.FS, spec wavefrontcomv1.WavefrontOperatorSpec, resourceFiles []string) ([]string, error) {
	var resourceYamls []string
	for _, resourceFile := range resourceFiles {
		resourceTemplate, err := template.ParseFS(f, resourceFile)
		if err != nil {
			return nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, spec)
		if err != nil {
			return nil, err
		}
		resourceYamls = append(resourceYamls, buffer.String())
	}
	return resourceYamls, nil
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
				files = append(files, path)
			}
			return nil
		},
	)

	return files, err
}
