package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func TestReconcileReportHealthStatus(t *testing.T) {
	t.Run("report health status when all components are healthy", func(t *testing.T) {
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-cluster-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-node-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-proxy":             {},
			"wavefront-cluster-collector": {},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": {},
		}

		warnings, healthy, message := UpdateComponentStatuses(appsV1, deploymentStatuses, daemonSetStatuses, &wf.Wavefront{})
		assert.True(t, healthy)
		assert.Equal(t, "(3/3) wavefront components are healthy.", message)
		assert.True(t, deploymentStatuses["wavefront-proxy"].Healthy)
		assert.Equal(t, "Running (1/1)", deploymentStatuses["wavefront-proxy"].Status)
		assert.True(t, deploymentStatuses["wavefront-cluster-collector"].Healthy)
		assert.Equal(t, "Running (1/1)", deploymentStatuses["wavefront-proxy"].Status)
		assert.True(t, daemonSetStatuses["wavefront-node-collector"].Healthy)
		assert.Equal(t, "Running (3/3)", daemonSetStatuses["wavefront-node-collector"].Status)
		assert.False(t, warnings)
	})

	t.Run("report health status when one component is unhealthy", func(t *testing.T) {
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 0,
			},
		}
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-cluster-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-node-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-proxy":             {},
			"wavefront-cluster-collector": {},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": {},
		}

		warnings, healthy, message := UpdateComponentStatuses(appsV1, deploymentStatuses, daemonSetStatuses, &wf.Wavefront{})
		assert.False(t, warnings)
		assert.False(t, healthy)
		assert.Equal(t, "(2/3) wavefront components are healthy.", message)
	})

	t.Run("report health status with less components", func(t *testing.T) {
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-cluster-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-node-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(collectorDeployment, collectorDaemonSet)
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-cluster-collector": {},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": {},
		}

		warnings, healthy, message := UpdateComponentStatuses(appsV1, deploymentStatuses, daemonSetStatuses, &wf.Wavefront{})
		assert.False(t, warnings)
		assert.True(t, healthy)
		assert.Equal(t, "(2/2) wavefront components are healthy.", message)
	})

	t.Run("clear out previous values when updating status", func(t *testing.T) {
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-cluster-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-node-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-proxy": &wf.DeploymentStatus{
				Message: "previous proxy message",
				Status:  "Running (0/1)",
				Healthy: false,
			},

			"wavefront-cluster-collector": {},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": &wf.DaemonSetStatus{
				Message: "previous collector message",
				Status:  "Running (0/1)",
				Healthy: false,
			},
		}

		warnings, healthy, message := UpdateComponentStatuses(appsV1, deploymentStatuses, daemonSetStatuses, &wf.Wavefront{})
		assert.False(t, warnings)
		assert.True(t, healthy)
		assert.Equal(t, "(3/3) wavefront components are healthy.", message)
		assert.True(t, deploymentStatuses["wavefront-proxy"].Healthy)
		assert.Equal(t, "Running (1/1)", deploymentStatuses["wavefront-proxy"].Status)
		assert.True(t, daemonSetStatuses["wavefront-node-collector"].Healthy)
		assert.Equal(t, "Running (3/3)", daemonSetStatuses["wavefront-node-collector"].Status)
	})

	t.Run("clear out previous values when updating status", func(t *testing.T) {
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-proxy",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-cluster-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wavefront-node-collector",
				Namespace: "wavefront",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-proxy": &wf.DeploymentStatus{
				Message: "previous proxy message",
				Status:  "Running (0/1)",
				Healthy: false,
			},

			"wavefront-cluster-collector": {},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": &wf.DaemonSetStatus{
				Message: "previous collector message",
				Status:  "Running (0/1)",
				Healthy: false,
			},
		}

		warnings, healthy, message := UpdateComponentStatuses(appsV1, deploymentStatuses, daemonSetStatuses, &wf.Wavefront{
			Spec: wf.WavefrontSpec{
				DataExport: wf.DataExport{
					ExternalWavefrontProxy: wf.ExternalWavefrontProxy{Url: "testURL"},
					WavefrontProxy:         wf.WavefrontProxy{Enable: true},
				},
			},
		})
		assert.True(t, healthy)
		assert.Contains(t, message, "Warning")
		assert.True(t, warnings)
	})
}

func setup(initObjs ...runtime.Object) typedappsv1.AppsV1Interface {
	return k8sfake.NewSimpleClientset(initObjs...).AppsV1()
}
