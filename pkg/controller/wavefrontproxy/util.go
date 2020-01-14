// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package wavefrontproxy

import (
	"reflect"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// getLabels returns the labels for the given WavefrontProxy CR name.
func getLabels(ip *InternalWavefrontProxy) map[string]string {
	// For consistency, labels assigned are as per suggested yaml at
	// https://github.com/wavefrontHQ/wavefront-kubernetes/blob/master/wavefront-proxy/wavefront.yaml
	// If any changes are made, make sure they are reflected in both places.
	// TODO: For now, these labels are good enough. When we make the operator cluster scoped,
	// we will need to add namespace as an additional label here.
	return map[string]string{
		"app":  "wavefront-proxy",
		"name": ip.instance.Name,
	}
}

// specChanged compares the current vs desired Deployments.
func specChanged(existingDep *appsv1.Deployment, desiredDep *appsv1.Deployment) bool {
	if deploymentSpecChanged(existingDep, desiredDep) {
		return true
	}
	return false
}

// podTemplateSpecChanged compares the current vs desired podTemplateSpec related parameters.
func podTemplateSpecChanged(currPTSpec *corev1.PodTemplateSpec, desiredPTSpec *corev1.PodTemplateSpec) bool {
	// Compare Labels.
	if !reflect.DeepEqual(currPTSpec.ObjectMeta.Labels, desiredPTSpec.ObjectMeta.Labels) {
		return true
	}

	if podSpecChanged(&currPTSpec.Spec, &desiredPTSpec.Spec) {
		return true
	}

	return false
}

// Note: We do not check the following for updates:
// 1. ConfigMap changes: Preprocessor config map changes and Advanced config changes.
// 2. AdditionalPort changes. (Given they are modified only when advanced configMap is
// modified.
// In case of modifying config maps, user will need to recreate the CR.
// podSpecChanged compares the current to desired podSpec related parameters.
func podSpecChanged(currPodSpec *corev1.PodSpec, desPodSpec *corev1.PodSpec) bool {
	// Compare Images
	if len(currPodSpec.Containers) == 1 && len(desPodSpec.Containers) == 1 {
		c := currPodSpec.Containers[0]
		d := desPodSpec.Containers[0]
		if c.Image != d.Image {
			return true
		}

		if c.Name != d.Name {
			return true
		}

		if c.ImagePullPolicy != d.ImagePullPolicy {
			return true
		}

		if !reflect.DeepEqual(c.Env, d.Env) {
			return true
		}

		// Compare container ports.
		// TODO: Probably this comparison is not required since we compare Env args already.
		var currCP []corev1.ContainerPort
		for _, containerPort := range c.Ports {
			currCP = append(currCP, corev1.ContainerPort{Name: containerPort.Name,
				ContainerPort: containerPort.ContainerPort})
		}
		desCP := d.Ports
		sort.Slice(currCP, func(i, j int) bool { return currCP[i].Name < currCP[j].Name })
		sort.Slice(desCP, func(i, j int) bool { return desCP[i].Name < desCP[j].Name })

		if !reflect.DeepEqual(currCP, desCP) {
			return true
		}

	} else {
		log.Info("WARN :: Multiple Containers in deployment which is unexpected.")
		return false
	}

	return false
}

// deploymentSpecChanged compares the current vs desired DeploymentSpec related parameters.
func deploymentSpecChanged(currDep *appsv1.Deployment, desiredDep *appsv1.Deployment) bool {
	// Compare Replicas. (Defaults to 1)
	var desiredReplicas int32
	if desiredDep.Spec.Replicas == nil {
		desiredReplicas = 1
	}

	if *currDep.Spec.Replicas != desiredReplicas {
		return true
	}

	// Compare Selector Labels.
	if !reflect.DeepEqual(currDep.Spec.Selector, desiredDep.Spec.Selector) {
		return true
	}

	if podTemplateSpecChanged(&currDep.Spec.Template, &desiredDep.Spec.Template) {
		return true
	}

	return false
}

// verifyAndModifySvc compares current vs desired service (from desired CR) and returns the desired
// service as needed.
func verifyAndModifySvc(svc corev1.Service, ip *InternalWavefrontProxy) *corev1.Service {
	isSvcSpecChanged := false

	// verify service's selector change
	desiredSelector := selectorForSvc(ip)
	if !reflect.DeepEqual(svc.Spec.Selector, desiredSelector) {
		isSvcSpecChanged = true
		svc.Spec.Selector = desiredSelector
	}

	// verify service's service ports change
	desiredServicePorts := ip.ServicePorts
	if !reflect.DeepEqual(svc.Spec.Ports, desiredServicePorts) {
		isSvcSpecChanged = true
		svc.Spec.Ports = desiredServicePorts
	}

	if isSvcSpecChanged {
		return &svc
	}

	return nil
}

// getCommaSeparatedPorts converts a comma separated string into a slice.
func getCommaSeparatedPorts(ports string) []string {
	p := strings.Split(ports, ",")
	for i := range p {
		p[i] = strings.TrimSpace(p[i])
	}
	return p
}
