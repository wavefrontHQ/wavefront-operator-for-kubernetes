package v1alpha1

import (
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, expectedClusterName, r.Spec.ClusterName)
}

func TestValidateCreate(t *testing.T) {
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
	err := r.ValidateCreate()
	expectedErr := "WavefrontUrl cannot be empty.\nWavefrontToken cannot be empty.\nClusterName cannot be empty.\n"
	assert.Equal(t, expectedErr, err.Error())
}

func TestValidateUpdate(t *testing.T) {
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
	err := r.ValidateUpdate(&r)
	expectedErr := "WavefrontUrl cannot be empty.\nWavefrontToken cannot be empty.\nClusterName cannot be empty.\n"
	assert.Equal(t, expectedErr, err.Error())
}
