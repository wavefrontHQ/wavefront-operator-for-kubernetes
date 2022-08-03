package validation

import (
	"fmt"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

func Validate(wavefront *wf.Wavefront) error {
	var errs []error
	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
			errs = append(errs, fmt.Errorf("'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time"))
		}
		errs = append(errs, validateResources(&wavefront.Spec.DataExport.WavefrontProxy.Resources, "spec.dataExport.wavefrontProxy")...)
	}
	return utilerrors.NewAggregate(errs)
}

func validateResources(resources *wf.Resources, resourcePath string) []error {
	var errs []error

	if compareQuantities(resources.Requests.CPU, resources.Limits.CPU) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.cpu: %s must be less than or equal to cpu limit.", resourcePath, resources.Requests.CPU))
	}
	if compareQuantities(resources.Requests.Memory, resources.Limits.Memory) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.memory: %s must be less than or equal to memory limit.", resourcePath, resources.Requests.Memory))
	}
	return errs
}

func compareQuantities(request string, limit string) int {
	requestQuantity, _ := resource.ParseQuantity(request)
	limitQuanity, _ := resource.ParseQuantity(limit)
	return requestQuantity.Cmp(limitQuanity)
}
