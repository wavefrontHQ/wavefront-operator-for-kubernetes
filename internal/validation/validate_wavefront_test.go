package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func TestValidate(t *testing.T) {
	t.Run("wf spec and environment are valid", func(t *testing.T) {
		appsV1 := setup()
		require.True(t, Validate(appsV1, defaultWFCR()).IsValid())
		require.False(t, Validate(appsV1, defaultWFCR()).IsError())
	})

	t.Run("wf spec is invalid", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.ExternalWavefrontProxy.Url = "https://testproxy.com"
		appsV1 := setup()
		result := Validate(appsV1, wfCR)
		require.False(t, result.IsValid())
		require.True(t, result.IsError())
		require.NotEmpty(t, result.Message())
	})

	t.Run("legacy install is running", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		result := Validate(appsV1, defaultWFCR())
		require.False(t, result.IsValid())
		require.True(t, result.IsError())
		require.NotEmpty(t, result.Message())
	})

	t.Run("legacy install is running after operator install", func(t *testing.T) {
		legacyCollector := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-collector",
				Namespace: "wavefront-collector",
			},
		}
		legacyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "wavefront-collector",
			},
		}
		nodeCollector := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.NodeCollectorName,
				Namespace: util.Namespace(),
			},
		}
		proxy := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ProxyName,
				Namespace: util.Namespace(),
			},
		}
		appsV1 := setup(legacyCollector, legacyDeployment, nodeCollector, proxy)
		wfCR := defaultWFCR()

		result := Validate(appsV1, wfCR)
		require.False(t, result.IsValid())
		require.False(t, result.IsError())
		require.True(t, result.IsWarning())
		require.NotEmpty(t, result.Message())
	})

	t.Run("allow legacy install", func(t *testing.T) {
		appsV1 := legacyEnvironmentSetup("wavefront")
		wfCR := defaultWFCR()
		wfCR.Spec.AllowLegacyInstall = true
		result := Validate(appsV1, wfCR)
		require.True(t, result.IsValid())
		require.False(t, result.IsError())
	})

}

func TestValidateWavefrontSpec(t *testing.T) {
	t.Run("Has no validation errors", func(t *testing.T) {
		wfCR := defaultWFCR()
		require.Empty(t, validateWavefrontSpec(wfCR))
	})

	t.Run("Validation error when both wavefront proxy and external proxy are defined", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.ExternalWavefrontProxy.Url = "https://testproxy.com"
		require.Equal(t, "'externalWavefrontProxy.url' and 'wavefrontProxy.enable' should not be set at the same time", validateWavefrontSpec(wfCR).Error())
	})

	t.Run("Validation error when CPU request is greater than CPU limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "200m"
		require.Equal(t, "invalid spec.dataExport.wavefrontProxy.resources.requests.cpu: 500m must be less than or equal to cpu limit", validateWavefrontSpec(wfCR).Error())
	})

	t.Run("CPU expressed differently should not be an error", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.CPU = "0.5"
		require.Nilf(t, validateWavefrontSpec(wfCR), "did not expect validation error")
	})

	t.Run("Validation error when Memory request is greater than Memory limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.Memory = "500Mi"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.Memory = "200Mi"
		validationError := validateWavefrontSpec(wfCR)
		require.NotNilf(t, validationError, "expected validation error")
		require.Equal(t, "invalid spec.dataExport.wavefrontProxy.resources.requests.memory: 500Mi must be less than or equal to memory limit", validationError.Error())
	})

	t.Run("Validation error when EphemeralStorage request is greater than limit", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Requests.EphemeralStorage = "1Gi"
		wfCR.Spec.DataExport.WavefrontProxy.Resources.Limits.EphemeralStorage = "500Mi"
		validationError := validateWavefrontSpec(wfCR)
		require.NotNilf(t, validationError, "expected validation error")
		require.Equal(t, "invalid spec.dataExport.wavefrontProxy.resources.requests.ephemeral-storage: 1Gi must be less than or equal to ephemeral-storage limit", validationError.Error())
	})

	t.Run("Validation error om node collector resources", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataCollection.Metrics.NodeCollector.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataCollection.Metrics.NodeCollector.Resources.Limits.CPU = "200m"
		require.Equal(t, "invalid spec.dataCollection.metrics.nodeCollector.resources.requests.cpu: 500m must be less than or equal to cpu limit", validateWavefrontSpec(wfCR).Error())
	})

	t.Run("Validation error on cluster collector resources", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "500Mi"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "200Mi"
		validationError := validateWavefrontSpec(wfCR)
		require.NotNilf(t, validationError, "expected validation error")
		require.Equal(t, "invalid spec.dataCollection.metrics.clusterCollector.resources.requests.memory: 500Mi must be less than or equal to memory limit", validationError.Error())
	})

	t.Run("Test multiple errors", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.Memory = "500Mi"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.Memory = "200Mi"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Requests.CPU = "500m"
		wfCR.Spec.DataCollection.Metrics.ClusterCollector.Resources.Limits.CPU = "200m"
		validationError := validateWavefrontSpec(wfCR)
		require.NotNilf(t, validationError, "expected validation error")
		require.Equal(t, "[invalid spec.dataCollection.metrics.clusterCollector.resources.requests.cpu: 500m must be less than or equal to cpu limit, invalid spec.dataCollection.metrics.clusterCollector.resources.requests.memory: 500Mi must be less than or equal to memory limit]", validationError.Error())
	})

	t.Run("Test No Proxy configuration", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Enable = false
		validationError := validateWavefrontSpec(wfCR)
		require.NotNilf(t, validationError, "expected validation error")
	})

	t.Run("Test External Proxy configuration", func(t *testing.T) {
		wfCR := defaultWFCR()
		wfCR.Spec.DataExport.WavefrontProxy.Enable = false
		wfCR.Spec.DataExport.ExternalWavefrontProxy.Url = "https://external-wf-proxy"
		require.Empty(t, validateWavefrontSpec(wfCR))
	})
}

func TestValidateEnvironment(t *testing.T) {
	t.Run("No existing collector daemonset", func(t *testing.T) {
		appsV1 := setup()
		require.NoError(t, validateEnvironment(appsV1, defaultWFCR()))
	})

	t.Run("Return error when only proxy deployment found", func(t *testing.T) {
		namespace := "wavefront"
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: namespace,
			},
		}
		appsV1 := setup(proxyDeployment)
		validationError := validateEnvironment(appsV1, defaultWFCR())
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Return error when legacy manual install found in namespace wavefront-collector", func(t *testing.T) {
		namespace := "wavefront-collector"
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-collector",
				Namespace: namespace,
			},
		}
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "default",
			},
		}
		appsV1 := setup(collectorDaemonSet, proxyDeployment)
		validationError := validateEnvironment(appsV1, defaultWFCR())
		require.NotNilf(t, validationError, "expected validation error")
		require.Contains(t, validationError.Error(), "Found legacy Wavefront installation in")
	})

	t.Run("Return error when legacy tkgi install found in namespace tanzu-observability-saas", func(t *testing.T) {
		namespace := "tanzu-observability-saas"
		appsV1 := legacyEnvironmentSetup(namespace)
		validationError := validateEnvironment(appsV1, defaultWFCR())
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Return error when collector daemonset found in legacy helm install namespace wavefront", func(t *testing.T) {
		namespace := "wavefront"
		appsV1 := legacyEnvironmentSetup(namespace)
		validationError := validateEnvironment(appsV1, defaultWFCR())
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Return error when collector deployment found in legacy tkgi install namespace pks-system", func(t *testing.T) {
		namespace := "pks-system"
		appsV1 := legacyEnvironmentSetup(namespace)
		validationError := validateEnvironment(appsV1, defaultWFCR())
		require.NotNilf(t, validationError, "expected validation error")
		requireValidationMessage(t, validationError, namespace)
	})

	t.Run("Returns nil when allow legacy install is enabled", func(t *testing.T) {
		namespace := "wavefront"
		appsV1 := legacyEnvironmentSetup(namespace)
		wfCR := defaultWFCR()
		wfCR.Spec.AllowLegacyInstall = true
		validationError := validateEnvironment(appsV1, wfCR)
		require.Nilf(t, validationError, "expected validation error")
	})

}

func requireValidationMessage(t *testing.T, validationError error, namespace string) {
	require.Equal(t, legacyEnvironmentError(namespace).Error(), validationError.Error())
}

func legacyEnvironmentSetup(namespace string) typedappsv1.AppsV1Interface {
	collectorDaemonSet := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront-collector",
			Namespace: namespace,
		},
	}
	proxyDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront-proxy",
			Namespace: namespace,
		},
	}
	appsV1 := setup(collectorDaemonSet, proxyDeployment)
	return appsV1
}

func setup(initObjs ...runtime.Object) typedappsv1.AppsV1Interface {
	return k8sfake.NewSimpleClientset(initObjs...).AppsV1()
}

func defaultWFCR() *wf.Wavefront {
	return &wf.Wavefront{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: util.Namespace(),
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
