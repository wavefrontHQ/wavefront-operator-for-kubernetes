package validation

import (
	"fmt"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func Validate(appsV1 typedappsv1.AppsV1Interface, wavefront *wf.Wavefront) error {
	err := ValidateEnvironment(appsV1)
	if err != nil {
		return err
	}
	return ValidateWavefrontSpec(wavefront)
}

func ValidateEnvironment(appsV1 typedappsv1.AppsV1Interface) error {
	daemonSet, err := appsV1.DaemonSets("wavefront-collector").Get(context.Background(), "wavefront-collector", v1.GetOptions{})
	if err == nil && daemonSet != nil {
		return fmt.Errorf("Detected legacy Wavefront installation in the wavefront-collector namespace. Please uninstall legacy installation before installing with the Wavefront Kubernetes Operator.")
	}
	return nil
}

func ValidateWavefrontSpec(wavefront *wf.Wavefront) error {
	var errs []error
	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
			errs = append(errs, fmt.Errorf("'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time"))
		}
		errs = append(errs, validateResources(&wavefront.Spec.DataExport.WavefrontProxy.Resources, "spec.dataExport.wavefrontProxy")...)
	}
	if wavefront.Spec.DataCollection.Metrics.Enable {
		errs = append(errs, validateResources(&wavefront.Spec.DataCollection.Metrics.NodeCollector.Resources, "spec.dataCollection.metrics.nodeCollector")...)
		errs = append(errs, validateResources(&wavefront.Spec.DataCollection.Metrics.ClusterCollector.Resources, "spec.dataCollection.metrics.clusterCollector")...)
	}
	return utilerrors.NewAggregate(errs)
}

func validateResources(resources *wf.Resources, resourcePath string) []error {
	var errs []error

	if compareQuantities(resources.Requests.CPU, resources.Limits.CPU) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.cpu: %s must be less than or equal to cpu limit", resourcePath, resources.Requests.CPU))
	}
	if compareQuantities(resources.Requests.Memory, resources.Limits.Memory) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.memory: %s must be less than or equal to memory limit", resourcePath, resources.Requests.Memory))
	}
	if compareQuantities(resources.Requests.EphemeralStorage, resources.Limits.EphemeralStorage) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.requests.ephemeral-storage: %s must be less than or equal to ephemeral-storage limit", resourcePath, resources.Requests.EphemeralStorage))
	}
	return errs
}

func compareQuantities(request string, limit string) int {
	requestQuantity, _ := resource.ParseQuantity(request)
	limitQuanity, _ := resource.ParseQuantity(limit)
	return requestQuantity.Cmp(limitQuanity)
}
