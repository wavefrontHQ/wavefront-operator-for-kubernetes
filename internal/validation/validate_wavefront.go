package validation

import (
	"fmt"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

var legacyComponentsToCheck = map[string]map[string]string{
	"wavefront-collector":      {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
	"default":                  {"wavefront-proxy": util.Deployment},
	"wavefront":                {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
	"pks-system":               {"wavefront-collector": util.Deployment, "wavefront-proxy": util.Deployment},
	"tanzu-observability-saas": {"wavefront-collector": util.DaemonSet, "wavefront-proxy": util.Deployment},
}

func Validate(appsV1 typedappsv1.AppsV1Interface, wavefront *wf.Wavefront) error {
	err := validateEnvironment(appsV1)
	if err != nil {
		return err
	}
	return validateWavefrontSpec(wavefront)
}

func validateEnvironment(appsV1 typedappsv1.AppsV1Interface) error {
	for namespace, resourceMap := range legacyComponentsToCheck {
		for resourceName, resourceType := range resourceMap {
			if resourceType == util.DaemonSet {
				daemonSet, err := appsV1.DaemonSets(namespace).Get(context.Background(), resourceName, v1.GetOptions{})
				if err == nil && daemonSet != nil {
					return legacyEnvironmentError(namespace)
				}
			}
			if resourceType == util.Deployment {
				deployment, err := appsV1.Deployments(namespace).Get(context.Background(), resourceName, v1.GetOptions{})
				if err == nil && deployment != nil {
					return legacyEnvironmentError(namespace)
				}
			}
		}
	}
	return nil
}

func legacyEnvironmentError(namespace string) error {
	return fmt.Errorf("Detected legacy Wavefront installation in the %s namespace. Please uninstall legacy installation before installing with the Wavefront Kubernetes Operator.", namespace)
}

func validateWavefrontSpec(wavefront *wf.Wavefront) error {
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
