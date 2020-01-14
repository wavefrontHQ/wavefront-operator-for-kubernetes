package controller

import (
	"github.com/wavefronthq/wavefront-operator-for-kubernetes/pkg/controller/wavefrontcollector"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, wavefrontcollector.Add)
}
