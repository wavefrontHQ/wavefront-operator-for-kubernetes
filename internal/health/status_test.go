package health

import (
	"testing"
	"time"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

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
				Name:      util.ProxyName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ClusterCollectorName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.NodeCollectorName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		componentsToCheck := map[string]string{
			util.ProxyName:            util.Deployment,
			util.ClusterCollectorName: util.Deployment,
			util.NodeCollectorName:    util.DaemonSet,
		}

		status := GenerateWavefrontStatus(appsV1, componentsToCheck, time.Now())
		assert.Equal(t, Healthy, status.Status)
		assert.Equal(t, "All components are healthy", status.Message)

		proxyStatus := getComponentStatusWithName(util.ProxyName, status.ResourceStatuses)
		assert.True(t, proxyStatus.Healthy)
		assert.Equal(t, "Running (1/1)", proxyStatus.Status)

		clusterCollectorStatus := getComponentStatusWithName(util.ClusterCollectorName, status.ResourceStatuses)
		assert.True(t, clusterCollectorStatus.Healthy)
		assert.Equal(t, "Running (1/1)", clusterCollectorStatus.Status)

		nodeCollectorStatus := getComponentStatusWithName(util.NodeCollectorName, status.ResourceStatuses)
		assert.True(t, nodeCollectorStatus.Healthy)
		assert.Equal(t, "Running (3/3)", nodeCollectorStatus.Status)
	})

	t.Run("report health status when one component is unhealthy", func(t *testing.T) {
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ProxyName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 0,
			},
		}
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ClusterCollectorName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.NodeCollectorName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		componentsToCheck := map[string]string{
			util.ProxyName:            util.Deployment,
			util.ClusterCollectorName: util.Deployment,
			util.NodeCollectorName:    util.DaemonSet,
		}
		status := GenerateWavefrontStatus(appsV1, componentsToCheck, pastMaxInstallTime())

		assert.Equal(t, Unhealthy, status.Status)
		assert.Equal(t, "not enough instances of wavefront-proxy are running (0/1)", status.Message)
	})

	t.Run("report health status with less components", func(t *testing.T) {
		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ClusterCollectorName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				AvailableReplicas: 1,
			},
		}
		collectorDaemonSet := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.NodeCollectorName,
				Namespace: util.Namespace(),
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		appsV1 := setup(collectorDeployment, collectorDaemonSet)
		componentsToCheck := map[string]string{
			util.ClusterCollectorName: util.Deployment,
			util.NodeCollectorName:    util.DaemonSet,
		}
		status := GenerateWavefrontStatus(appsV1, componentsToCheck, time.Now())

		assert.Equal(t, Healthy, status.Status)
		assert.Equal(t, "All components are healthy", status.Message)
	})

	t.Run("report health status when no components are running", func(t *testing.T) {
		appsV1 := setup()
		componentsToCheck := map[string]string{
			util.ProxyName:            util.Deployment,
			util.ClusterCollectorName: util.Deployment,
			util.NodeCollectorName:    util.DaemonSet,
		}
		status := GenerateWavefrontStatus(appsV1, componentsToCheck, pastMaxInstallTime())

		assert.Equal(t, Unhealthy, status.Status)
		assert.Equal(t, "", status.Message)
		proxyStatus := getComponentStatusWithName(util.ProxyName, status.ResourceStatuses)
		assert.False(t, proxyStatus.Healthy)
		assert.Equal(t, "Not running", proxyStatus.Status)

		clusterCollectorStatus := getComponentStatusWithName(util.ClusterCollectorName, status.ResourceStatuses)
		assert.False(t, clusterCollectorStatus.Healthy)
		assert.Equal(t, "Not running", clusterCollectorStatus.Status)

		nodeCollectorStatus := getComponentStatusWithName(util.NodeCollectorName, status.ResourceStatuses)
		assert.False(t, nodeCollectorStatus.Healthy)
		assert.Equal(t, "Not running", nodeCollectorStatus.Status)
	})

	t.Run("report health status as installing until MaxInstallTime has elapsed", func(t *testing.T) {
		appsV1 := setup()
		componentsToCheck := map[string]string{
			util.ProxyName:            util.Deployment,
			util.ClusterCollectorName: util.Deployment,
			util.NodeCollectorName:    util.DaemonSet,
		}
		status := GenerateWavefrontStatus(appsV1, componentsToCheck, time.Now().Add(-MaxInstallTime).Add(time.Second*10))

		assert.Equal(t, Installing, status.Status)
		assert.Equal(t, "Installing components", status.Message)
		for _, resourceStatus := range status.ResourceStatuses {
			assert.True(t, resourceStatus.Installing)
		}
	})

	t.Run("report health status as unhealthy after MaxInstallTime has elapsed", func(t *testing.T) {
		appsV1 := setup()
		componentsToCheck := map[string]string{
			util.ProxyName:            util.Deployment,
			util.ClusterCollectorName: util.Deployment,
			util.NodeCollectorName:    util.DaemonSet,
		}
		status := GenerateWavefrontStatus(appsV1, componentsToCheck, pastMaxInstallTime().Add(time.Second*10))

		assert.Equal(t, Unhealthy, status.Status)
	})
}

func pastMaxInstallTime() time.Time {
	return time.Now().Add(-MaxInstallTime).Add(-time.Second * 10)
}

func setup(initObjs ...runtime.Object) typedappsv1.AppsV1Interface {
	return k8sfake.NewSimpleClientset(initObjs...).AppsV1()
}

func getComponentStatusWithName(name string, componentStatuses []wf.ResourceStatus) wf.ResourceStatus {
	for _, componentStatus := range componentStatuses {
		if componentStatus.Name == name {
			return componentStatus
		}
	}
	return wf.ResourceStatus{}
}
