package wavefrontcollector

import (
	"reflect"
	"strings"

	wavefrontv1alpha1 "github.com/wavefronthq/wavefront-operator-for-kubernetes/pkg/apis/wavefront/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func specChanged(spec *corev1.PodTemplateSpec, crSpec *wavefrontv1alpha1.WavefrontCollectorSpec) bool {
	newSpec := crSpec.DeepCopy()
	copyTemplateToCR(spec, newSpec)
	if !reflect.DeepEqual(crSpec, newSpec) {
		return true
	}
	return false
}

func copyTemplateToCR(template *corev1.PodTemplateSpec, crSpec *wavefrontv1alpha1.WavefrontCollectorSpec) {
	//TODO: implement

	// Image
	crSpec.Image = ""
	if len(template.Spec.Containers) == 1 {
		crSpec.Image = template.Spec.Containers[0].Image
	}

	// Daemon
	crSpec.Daemon = false
	if len(template.Spec.Containers) == 1 {
		for _, command := range template.Spec.Containers[0].Command {
			if strings.Contains(command, "daemon=true") {
				crSpec.Daemon = true
				break
			}
		}
	}

	//TODO: disableUpdate

	// Resources
	if len(template.Spec.Containers) == 1 {
		crSpec.Resources = template.Spec.Containers[0].Resources
	}

	// Env
	crSpec.Env = nil
	if len(template.Spec.Containers) == 1 {
		crSpec.Env = template.Spec.Containers[0].Env
	}

	// Tolerations
	crSpec.Tolerations = nil
	if template.Spec.Tolerations != nil {
		in, out := &template.Spec.Tolerations, &crSpec.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}

	// configName
	crSpec.ConfigName = ""
	if len(template.Spec.Volumes) > 0 {
		for _, volume := range template.Spec.Volumes {
			if volume.ConfigMap != nil {
				crSpec.ConfigName = volume.ConfigMap.LocalObjectReference.Name
				break
			}
		}
	}
}
