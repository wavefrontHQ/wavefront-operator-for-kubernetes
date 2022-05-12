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

package v1alpha1

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var wavefrontlog = logf.Log.WithName("wavefront-resource")

func (r *Wavefront) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-wavefront-com-v1alpha1-wavefront,mutating=true,failurePolicy=fail,sideEffects=None,groups=wavefront.com,resources=wavefronts,verbs=create;update,versions=v1alpha1,name=mwavefront.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Wavefront{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Wavefront) Default() {
	wavefrontlog.Info("default", "name", r.Name)

	if r.Spec.ClusterName == "" {
		r.Spec.ClusterName = "k8s-cluster"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-wavefront-com-v1alpha1-wavefront,mutating=false,failurePolicy=fail,sideEffects=None,groups=wavefront.com,resources=wavefronts,verbs=create;update,versions=v1alpha1,name=vwavefront.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Wavefront{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Wavefront) ValidateCreate() error {
	wavefrontlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateWavefront()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Wavefront) ValidateUpdate(old runtime.Object) error {
	wavefrontlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return r.validateWavefront()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Wavefront) ValidateDelete() error {
	wavefrontlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return r.validateWavefront()
}

func (r *Wavefront) validateWavefront() error {
	var allErrors []error
	err := r.validateWavefrontSpec()
	allErrors = append(allErrors, err...)
	//delimit and add new line.
	var outErrText string
	for _, err := range allErrors {
		outErrText += fmt.Sprintln(err.Error())
	}
	if len(outErrText) != 0 {
		return errors.New(outErrText)
	}
	return nil
}

func (r *Wavefront) validateWavefrontSpec() []error {
	var allErrors []error
	if r.Spec.WavefrontUrl == "" {
		allErrors = append(allErrors, errors.New("WavefrontUrl cannot be empty."))
	}
	if r.Spec.ClusterName == "" {
		allErrors = append(allErrors, errors.New("ClusterName cannot be empty."))
	}
	if r.Spec.WavefrontToken == "" {
		allErrors = append(allErrors, errors.New("WavefrontToken cannot be empty."))
	}
	return allErrors
}
