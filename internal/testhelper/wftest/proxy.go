package wftest

import (
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Proxy(options ...func(*appsv1.Deployment)) *appsv1.Deployment {
	replicas := int32(1)
	proxy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.ProxyName,
			Namespace: DefaultNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "proxy",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
	for _, option := range options {
		option(proxy)
	}
	return proxy
}

func WithReplicas(availableReplicas, replicas int) func(*appsv1.Deployment) {
	specReplicas := int32(replicas)
	return func(d *appsv1.Deployment) {
		d.Status.AvailableReplicas = int32(availableReplicas)
		d.Spec.Replicas = &specReplicas
	}
}
