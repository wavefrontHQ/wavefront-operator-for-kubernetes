package validation

import (
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Validate(wavefront *wf.Wavefront) string {
	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		if len(wavefront.Spec.DataExport.ExternalWavefrontProxy.Url) != 0 {
			return "It is not valid to define an external proxy (externalWavefrontProxy.url) and enable the wavefront proxy (wavefrontProxy.enable) in your Kubernetes cluster."
		}
		request := resource.MustParse(wavefront.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU)
		limit := resource.MustParse(wavefront.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU)
		//TODO: Figure out correct way for comparison of CPU
		if request.Cmp(limit) > 0 {
			return "wavefront is invalid (spec.dataExport.wavefrontProxy.resources.requests): " + wavefront.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU + " must be less than or equal to cpu limit."
		}
	}
	return ""
}
