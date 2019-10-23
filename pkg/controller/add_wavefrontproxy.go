package controller

import (
	"github.com/wavefronthq/wavefront-operator/pkg/controller/wavefrontproxy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, wavefrontproxy.Add)
}
