// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package wavefrontproxy

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const wfProxyContainerName = "wavefront-proxy"

// newDeployment returns a Wavefront Proxy Deployment object
// Make sure if you add/modify any of the DeploymentSpec logic,
// the corresponding comparison logic for deployment spec is also updated in deploymentSpecChanged.
func newDeployment(ip *InternalWavefrontProxy) *appsv1.Deployment {
	labels := getLabels(ip)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ip.instance.Name,
			Namespace: ip.instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ip.instance.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: newPodTemplateSpec(ip),
		},
	}
	return dep
}

// Make sure if you add/modify any of the podTemplate Spec logic,
// the corresponding comparison logic for podTemplate spec is also updated in podTemplateSpecChanged.
func newPodTemplateSpec(ip *InternalWavefrontProxy) corev1.PodTemplateSpec {
	labels := getLabels(ip)
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: labels},
		Spec:       newPodSpec(ip),
	}
}

// Make sure if you add/modify any of the podSpec logic,
// the corresponding comparison logic for podspec is also updated in podSpecChanged.
func newPodSpec(ip *InternalWavefrontProxy) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            wfProxyContainerName,
			Image:           ip.instance.Spec.Image,
			ImagePullPolicy: defaultImagePullPolicy,
			Env:             constructEnvVars(ip),
			Ports:           ip.ContainerPorts,
			VolumeMounts:    ip.volumeMount,
		}},
		Volumes: ip.volume,
	}
}

func constructEnvVars(ip *InternalWavefrontProxy) []corev1.EnvVar {
	return []corev1.EnvVar{{
		Name:  "WAVEFRONT_URL",
		Value: ip.instance.Spec.Url,
	}, {
		Name:  "WAVEFRONT_TOKEN",
		Value: ip.instance.Spec.Token,
	}, {
		Name:  "WAVEFRONT_PROXY_ARGS",
		Value: ip.EnvWavefrontProxyArgs,
	}}
}

// --------------- Wavefront proxy Service related functions go below this line.------------- //

// newService returns a Wavefront Proxy Deployment object
func newService(ip *InternalWavefrontProxy) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ip.instance.Name,
			Namespace: ip.instance.Namespace,
		},
		Spec: svcSpec(ip),
	}
	return svc
}

//new pvc returns a pvc
func createPVC(ip *InternalWavefrontProxy) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ip.instance.Spec.StorageClaimName,
			Namespace: ip.instance.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("5G")}},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
}

func selectorForSvc(ip *InternalWavefrontProxy) map[string]string {
	return getLabels(ip)
}

func svcSpec(ip *InternalWavefrontProxy) corev1.ServiceSpec {
	return corev1.ServiceSpec{
		Ports:    ip.ServicePorts,
		Selector: selectorForSvc(ip),
	}
}
