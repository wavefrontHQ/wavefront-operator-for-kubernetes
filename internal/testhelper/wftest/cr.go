package wftest

import (
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CROption func(*wf.Wavefront)

func CR(options ...CROption) *wf.Wavefront {
	cr := &wf.Wavefront{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "wavefront",
			Namespace: "testNamespace",
		},
		Spec: wf.WavefrontSpec{
			Namespace:            "testNamespace",
			ClusterName:          "testClusterName",
			WavefrontUrl:         "testWavefrontUrl",
			WavefrontTokenSecret: "testToken",
			DataCollection: wf.DataCollection{
				Metrics: wf.Metrics{
					Enable: true,
				},
				Logging: wf.Logging{
					Enable: true,
				},
			},
			DataExport: wf.DataExport{
				WavefrontProxy: wf.WavefrontProxy{
					Enable:     true,
					MetricPort: 2878,
				},
			},
		},
	}
	for _, option := range options {
		option(cr)
	}
	return cr
}
