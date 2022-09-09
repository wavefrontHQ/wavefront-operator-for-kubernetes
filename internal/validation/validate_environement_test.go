package validation

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEnvironment(t *testing.T) {
	t.Run("No existing collector daemonset", func(t *testing.T) {
		appsV1 := setup()
		assert.NoError(t, ValidateEnvironment(appsV1))
	})

	t.Run("Return error when existing collector daemonset", func(t *testing.T) {
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-collector",
				Namespace: "wavefront-collector",
			},
		}

		appsV1 := setup(collectorDaemonSet)
		validationError := ValidateEnvironment(appsV1)
		assert.NotNilf(t, validationError, "expected validation error")
		assert.Equal(t, "Detected Collector DaemonSet running in wavefront-collector namespace. Please uninstall before installing operator", validationError.Error())
	})
}

func setup(initObjs ...runtime.Object) typedappsv1.AppsV1Interface {
	return k8sfake.NewSimpleClientset(initObjs...).AppsV1()
}
