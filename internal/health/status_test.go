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

		wfCR := &wf.Wavefront{}
		UpdateWavefrontStatus(appsV1, deploymentStatuses, daemonSetStatuses, wfCR)
		assert.Equal(t, "Healthy", wfCR.Status.Status)
		assert.Equal(t, "(3/3) wavefront components are healthy", wfCR.Status.Message)
		assert.True(t, deploymentStatuses["wavefront-proxy"].Healthy)
		assert.Equal(t, "Running (1/1)", deploymentStatuses["wavefront-proxy"].Status)
		assert.True(t, deploymentStatuses["wavefront-cluster-collector"].Healthy)
		assert.Equal(t, "Running (1/1)", deploymentStatuses["wavefront-proxy"].Status)
		assert.True(t, daemonSetStatuses["wavefront-node-collector"].Healthy)
		assert.Equal(t, "Running (3/3)", daemonSetStatuses["wavefront-node-collector"].Status)
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

		wfCR := &wf.Wavefront{}
		UpdateWavefrontStatus(appsV1, deploymentStatuses, daemonSetStatuses, wfCR)
		assert.Equal(t, "Unhealthy", wfCR.Status.Status)
		assert.Equal(t, "not enough instances of wavefront-proxy are running (0/1)", wfCR.Status.Message)
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

		wfCR := &wf.Wavefront{}
		UpdateWavefrontStatus(appsV1, deploymentStatuses, daemonSetStatuses, wfCR)
		assert.Equal(t, "Healthy", wfCR.Status.Status)
		assert.Equal(t, "(2/2) wavefront components are healthy", wfCR.Status.Message)
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

		wfCR := &wf.Wavefront{}
		UpdateWavefrontStatus(appsV1, deploymentStatuses, daemonSetStatuses, wfCR)
		assert.Equal(t, "Healthy", wfCR.Status.Status)
		assert.Equal(t, "(3/3) wavefront components are healthy", wfCR.Status.Message)
		assert.True(t, deploymentStatuses["wavefront-proxy"].Healthy)
		assert.Equal(t, "Running (1/1)", deploymentStatuses["wavefront-proxy"].Status)
		assert.True(t, daemonSetStatuses["wavefront-node-collector"].Healthy)
		assert.Equal(t, "Running (3/3)", daemonSetStatuses["wavefront-node-collector"].Status)
	})

	t.Run("report health status when no components are running", func(t *testing.T) {
		appsV1 := setup()
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-proxy":             {},
			"wavefront-cluster-collector": {},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": {},
		}

		wfCR := &wf.Wavefront{}
		UpdateWavefrontStatus(appsV1, deploymentStatuses, daemonSetStatuses, wfCR)
		assert.Equal(t, "Unhealthy", wfCR.Status.Status)
		assert.Equal(t, "", wfCR.Status.Message)
		assert.False(t, deploymentStatuses["wavefront-proxy"].Healthy)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-proxy"].Replicas)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-proxy"].AvailableReplicas)
		assert.Equal(t, "Not running", deploymentStatuses["wavefront-proxy"].Status)
		assert.False(t, deploymentStatuses["wavefront-cluster-collector"].Healthy)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-cluster-collector"].Replicas)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-cluster-collector"].AvailableReplicas)
		assert.Equal(t, "Not running", deploymentStatuses["wavefront-cluster-collector"].Status)
		assert.False(t, daemonSetStatuses["wavefront-node-collector"].Healthy)
		assert.Equal(t, int32(0), daemonSetStatuses["wavefront-node-collector"].DesiredNumberScheduled)
		assert.Equal(t, int32(0), daemonSetStatuses["wavefront-node-collector"].NumberReady)
		assert.Equal(t, "Not running", daemonSetStatuses["wavefront-node-collector"].Status)
	})

	t.Run("report health status when components are deleted", func(t *testing.T) {
		appsV1 := setup()
		deploymentStatuses := map[string]*wf.DeploymentStatus{
			"wavefront-proxy": &wf.DeploymentStatus{
				Message:           "previous proxy message",
				Status:            "Running (1/1)",
				Healthy:           true,
				Replicas:          1,
				AvailableReplicas: 1,
			},
			"wavefront-cluster-collector": {
				Message:           "previous collector message",
				Status:            "Running (1/1)",
				Healthy:           true,
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		daemonSetStatuses := map[string]*wf.DaemonSetStatus{
			"wavefront-node-collector": &wf.DaemonSetStatus{
				Message:                "previous collector message",
				Status:                 "Running (1/1)",
				Healthy:                true,
				DesiredNumberScheduled: 1,
				NumberReady:            1,
			},
		}

		wfCR := &wf.Wavefront{}
		UpdateWavefrontStatus(appsV1, deploymentStatuses, daemonSetStatuses, wfCR)
		assert.Equal(t, "Unhealthy", wfCR.Status.Status)
		assert.Equal(t, "", wfCR.Status.Message)
		assert.False(t, deploymentStatuses["wavefront-proxy"].Healthy)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-proxy"].Replicas)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-proxy"].AvailableReplicas)
		assert.Equal(t, "Not running", deploymentStatuses["wavefront-proxy"].Status)
		assert.False(t, deploymentStatuses["wavefront-cluster-collector"].Healthy)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-cluster-collector"].Replicas)
		assert.Equal(t, int32(0), deploymentStatuses["wavefront-cluster-collector"].AvailableReplicas)
		assert.Equal(t, "Not running", deploymentStatuses["wavefront-cluster-collector"].Status)
		assert.False(t, daemonSetStatuses["wavefront-node-collector"].Healthy)
		assert.Equal(t, int32(0), daemonSetStatuses["wavefront-node-collector"].DesiredNumberScheduled)
		assert.Equal(t, int32(0), daemonSetStatuses["wavefront-node-collector"].NumberReady)
		assert.Equal(t, "Not running", daemonSetStatuses["wavefront-node-collector"].Status)
	})

}

func setup(initObjs ...runtime.Object) typedappsv1.AppsV1Interface {
	return k8sfake.NewSimpleClientset(initObjs...).AppsV1()
}
