package apis

import (
	"github.com/wavefronthq/wavefront-operator-for-kubernetes/pkg/apis/wavefront/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
