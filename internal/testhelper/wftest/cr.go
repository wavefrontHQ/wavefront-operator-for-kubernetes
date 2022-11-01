package wftest

import (
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CR(options ...func(*wf.Wavefront)) *wf.Wavefront {
	cr := &wf.Wavefront{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront",
			Namespace: DefaultNamespace,
		},
		Spec: wf.WavefrontSpec{
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
