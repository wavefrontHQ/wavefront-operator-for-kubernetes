package wftest

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Operator(options ...func(*appsv1.Deployment)) *appsv1.Deployment {
	operator := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wavefront-controller-manager",
			Namespace: DefaultNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "controller-manager",
			},
			UID: "testUID",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "projects.registry.vmware.com/tanzu_observability/kubernetes-operator:latest",
					}},
				},
			},
		},
	}
	for _, option := range options {
		option(operator)
	}
	return operator
}
