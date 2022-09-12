package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
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
		assert.Equal(t, "'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time", Validate(wfCR).Error())
	})

	t.Run("Validation error when CPU request is greater than CPU limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "200m"
		assert.Equal(t, "invalid spec.dataExport.wavefrontProxy.resources.requests.cpu: 500m must be less than or equal to cpu limit", Validate(wfCR).Error())
	})

	t.Run("CPU expressed differently should not be an error", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "0.5"
		assert.Nilf(t, Validate(wfCR), "did not expect validation error")
	})

	t.Run("Validation error when Memory request is greater than Memory limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.Memory = "500Mi"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.Memory = "200Mi"
		validationError := Validate(wfCR)
		assert.NotNilf(t, validationError, "expected validation error")
		assert.Equal(t, "invalid spec.dataExport.wavefrontProxy.resources.requests.memory: 500Mi must be less than or equal to memory limit", validationError.Error())
	})

	t.Run("Validation error when EphemeralStorage request is greater than limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.EphemeralStorage = "1Gi"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.EphemeralStorage = "500Mi"
		validationError := Validate(wfCR)
		assert.NotNilf(t, validationError, "expected validation error")
		assert.Equal(t, "invalid spec.dataExport.wavefrontProxy.resources.requests.ephemeral-storage: 1Gi must be less than or equal to ephemeral-storage limit", validationError.Error())
	})

	t.Run("Validation error om node collector resources", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
		assert.Equal(t, "invalid spec.dataCollection.metrics.nodeCollector.resources.requests.cpu: 500m must be less than or equal to cpu limit", Validate(wfCR).Error())
	})

	t.Run("Validation error on cluster collector resources", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "500Mi"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "200Mi"
		validationError := Validate(wfCR)
		assert.NotNilf(t, validationError, "expected validation error")
		assert.Equal(t, "invalid spec.dataCollection.metrics.clusterCollector.resources.requests.memory: 500Mi must be less than or equal to memory limit", validationError.Error())
	})

	t.Run("Test multiple errors", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "500Mi"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "200Mi"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
		validationError := Validate(wfCR)
		assert.NotNilf(t, validationError, "expected validation error")
		assert.Equal(t, "[invalid spec.dataCollection.metrics.clusterCollector.resources.requests.cpu: 500m must be less than or equal to cpu limit, invalid spec.dataCollection.metrics.clusterCollector.resources.requests.memory: 500Mi must be less than or equal to memory limit]", validationError.Error())
	})
}

func defaultWFCR() *wf.Wavefront {
	return &wf.Wavefront{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: util.Namespace,
			Name:      util.Namespace,
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
