package validation

import (
	"fmt"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func ValidateEnvironment(appsV1 typedappsv1.AppsV1Interface) error {
	daemonSet, err := appsV1.DaemonSets("wavefront-collector").Get(context.Background(), "wavefront-collector", v1.GetOptions{})

	if err != nil && daemonSet != nil {
		return fmt.Errorf("Detected Collector DaemonSet running in wavefront-collector namespace. Please uninstall before installing operator")
	}
	return nil
}
