package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestDefault(t *testing.T) {
	spec := WavefrontSpec{
		WavefrontUrl:          "testWavefrontUrl",
		WavefrontToken:        "testToken",
		ClusterName:           "",
		CollectorEnabled:      true,
		WavefrontProxyEnabled: true,
		ProxyUrl:              "",
		ControllerManagerUID:  "",
	}
	var r = Wavefront{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       spec,
		Status:     WavefrontStatus{},
	}
	expectedClusterName := "k8s-cluster"
	r.Default()
	if r.Spec.ClusterName == "" {
		t.Errorf("Expected spec ClusterName to not be empty.")
	}
	if r.Spec.ClusterName != expectedClusterName {
		t.Errorf("Expected default clusterName :: %s, but got %s", expectedClusterName, r.Spec.ClusterName)
	}
}

func TestValidateWavefront(t *testing.T) {
	spec := WavefrontSpec{
		WavefrontUrl:          "",
		WavefrontToken:        "",
		ClusterName:           "",
		CollectorEnabled:      true,
		WavefrontProxyEnabled: true,
		ProxyUrl:              "",
		ControllerManagerUID:  "",
	}
	var r = Wavefront{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       spec,
		Status:     WavefrontStatus{},
	}
	err := r.validateWavefront()
	expectedErr := "WavefrontUrl cannot be empty.\nWavefrontToken cannot be empty.\nClusterName cannot be empty.\n"
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Expected validation error :: \n %s , but got :: \n %s.", expectedErr, err.Error())
	}
}
