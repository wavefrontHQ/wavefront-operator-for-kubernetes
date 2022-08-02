package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidate(t *testing.T) {
	t.Run("Has no validation errors", func(t *testing.T) {
		wfCR := defaultWFCR()
		assert.Empty(t, Validate(wfCR))
	})

	t.Run("Validation error when both wavefront proxy and external proxy are defined", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.ExternalWavefrontProxy.Url = "https://testproxy.com"
		assert.Equal(t, "It is not valid to define an external proxy (externalWavefrontProxy.url) and enable the wavefront proxy (wavefrontProxy.enable) in your Kubernetes cluster.", Validate(wfCR))
	})

	t.Run("Validation error when proxy CPU request is greater than CPU limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "0.5"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "500m"
		assert.Equal(t, "wavefront is invalid (spec.dataExport.wavefrontProxy.resources.requests): 200m must be less than or equal to cpu limit.", Validate(wfCR))
	})
}

func defaultWFCR() *wf.Wavefront {
	return &wf.Wavefront{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "wavefront",
			Name:      "wavefront",
		},
		Spec: wf.WavefrontSpec{
			ClusterName:  "testClusterName",
			WavefrontUrl: "testWavefrontUrl",
			DataExport: wf.DataExport{
				WavefrontProxy: wf.WavefrontProxy{
					Enable: true,
				},
			},
			DataCollection: wf.DataCollection{
				Metrics: wf.Metrics{
					Enable: true,
				},
			},
		},
		Status: wf.WavefrontStatus{},
	}
}
