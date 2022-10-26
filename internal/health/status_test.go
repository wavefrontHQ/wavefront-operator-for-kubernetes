package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "testNamespace"

func TestReconcileReportHealthStatus(t *testing.T) {
	t.Run("report health status when all components are healthy", func(t *testing.T) {
		wavefront := defaultWF()
		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ProxyName,
				Namespace: wavefront.Spec.Namespace,
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
				Namespace: wavefront.Spec.Namespace,
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
				Namespace: wavefront.Spec.Namespace,
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		client := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		wavefront.Spec.DataExport.WavefrontProxy.Enable = true
		wavefront.Spec.DataCollection.Metrics.Enable = true

		status := GenerateWavefrontStatus(client, wavefront)
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
		wavefront := defaultWF()

		proxyDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ProxyName,
				Namespace: wavefront.Spec.Namespace,
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
				Namespace: wavefront.Spec.Namespace,
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
				Namespace: wavefront.Spec.Namespace,
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		client := setup(proxyDeployment, collectorDeployment, collectorDaemonSet)
		wavefront.Spec.DataExport.WavefrontProxy.Enable = true
		wavefront.Spec.DataCollection.Metrics.Enable = true

		status := GenerateWavefrontStatus(client, wavefront)

		assert.Equal(t, Unhealthy, status.Status)
		assert.Equal(t, "not enough instances of wavefront-proxy are running (0/1)", status.Message)
	})

	t.Run("report health status with less components", func(t *testing.T) {
		wavefront := defaultWF()

		collectorDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.ClusterCollectorName,
				Namespace: wavefront.Spec.Namespace,
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
				Namespace: wavefront.Spec.Namespace,
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberReady:            3,
			},
		}

		client := setup(collectorDeployment, collectorDaemonSet)
		wavefront.Spec.DataCollection.Metrics.Enable = true

		status := GenerateWavefrontStatus(client, wavefront)

		assert.Equal(t, Healthy, status.Status)
		assert.Equal(t, "All components are healthy", status.Message)
	})

	t.Run("report health status when no components are running", func(t *testing.T) {
		client := setup()

		wfCR := defaultWF()
		wfCR.Spec.DataExport.WavefrontProxy.Enable = true
		wfCR.Spec.DataCollection.Metrics.Enable = true
		status := GenerateWavefrontStatus(client, wfCR)

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
		wavefront := defaultWF()
		client := setup()

		wavefront.CreationTimestamp.Time = time.Now().Add(-MaxInstallTime).Add(time.Second * 10)
		wavefront.Spec.DataCollection.Metrics.Enable = true
		wavefront.Spec.DataExport.WavefrontProxy.Enable = true
		status := GenerateWavefrontStatus(client, wavefront)

		assert.Equal(t, Installing, status.Status)
		assert.Equal(t, "Installing components", status.Message)
		for _, resourceStatus := range status.ResourceStatuses {
			assert.True(t, resourceStatus.Installing)
		}
	})

	t.Run("report health status as unhealthy after MaxInstallTime has elapsed", func(t *testing.T) {
		wavefront := defaultWF()
		client := setup()

		wavefront.CreationTimestamp.Time = pastMaxInstallTime().Add(time.Second * 10)
		wavefront.Spec.DataCollection.Metrics.Enable = true
		wavefront.Spec.DataExport.WavefrontProxy.Enable = true
		status := GenerateWavefrontStatus(client, wavefront)

		assert.Equal(t, Unhealthy, status.Status)
	})

	t.Run("logging", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "logging",
		}
		wavefront := defaultWF()
		wavefront.Spec.DataCollection.Logging.Enable = true
		RespondsToOOMKilled(t, wavefront,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Labels:    labels,
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "DaemonSet",
						Name: util.LoggingName,
					}},
				},
				Spec: corev1.PodSpec{},
			},
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      util.LoggingName,
					Labels:    labels,
				},
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 1,
					NumberReady:            1,
				},
			},
		)
	})

	t.Run("node collector", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "node-collector",
		}
		wavefront := defaultWF()
		wavefront.Spec.DataCollection.Metrics.Enable = true
		RespondsToOOMKilled(t, wavefront,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "DaemonSet",
						Name: util.NodeCollectorName,
					}},
					Namespace: testNamespace,
					Labels:    labels,
				},
			},
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      util.NodeCollectorName,
					Labels:    labels,
				},
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 1,
					NumberReady:            1,
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      util.ClusterCollectorName,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
					Replicas:          1,
				},
			},
		)
	})

	t.Run("cluster collector", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "cluster-collector",
		}
		wavefront := defaultWF()
		wavefront.Spec.DataCollection.Metrics.Enable = true
		RespondsToOOMKilled(t, wavefront,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{},
			},
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      util.NodeCollectorName,
				},
				Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 1,
					NumberReady:            1,
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      util.ClusterCollectorName,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
					Replicas:          1,
				},
			},
		)
	})

	t.Run("proxy", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/name":      "wavefront",
			"app.kubernetes.io/component": "proxy",
		}
		wavefront := defaultWF()
		wavefront.Spec.DataExport.WavefrontProxy.Enable = true
		RespondsToOOMKilled(t, wavefront,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Labels:    labels,
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "Deployment",
						Name: util.ProxyName,
					}},
				},
				Spec: corev1.PodSpec{},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      util.ProxyName,
					Labels:    labels,
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
					Replicas:          1,
				},
			},
		)
	})

	t.Run("controller-manager", func(t *testing.T) {
		wavefront := defaultWF()
		RespondsToOOMKilled(t, wavefront,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "wavefront",
						"app.kubernetes.io/component": "controller-manager",
					},
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "Deployment",
						Name: util.OperatorName,
					}},
				},
				Spec: corev1.PodSpec{},
			},
		)
	})
}

func RespondsToOOMKilled(t *testing.T, wavefront *wf.Wavefront, pod *corev1.Pod, apps ...runtime.Object) {
	t.Run("unhealthy when it has been OOM killed in the last five minutes", func(t *testing.T) {
		ourPod := *pod
		ourPod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			LastTerminationState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					ExitCode:   137, // OOMKilled
					FinishedAt: metav1.Time{Time: time.Now()},
				},
			},
		}}
		client := setup(append([]runtime.Object{&ourPod}, apps...)...)

		status := GenerateWavefrontStatus(client, wavefront)

		require.Equal(t, Unhealthy, status.Status)
		require.Contains(t, status.Message, "OOMKilled in the last 5m")
		for _, resourceStatus := range status.ResourceStatuses {
			if resourceStatus.Name == util.LoggingName {
				require.Equal(t, Unhealthy, resourceStatus.Status)
			}
		}
	})

	t.Run("healthy when it has not been OOM killed in the last five minutes", func(t *testing.T) {
		ourPod := *pod
		ourPod.Status.ContainerStatuses = []corev1.ContainerStatus{{
			LastTerminationState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					ExitCode:   137, // OOMKilled
					FinishedAt: metav1.Time{Time: time.Now().Add(-OOMTimeout).Add(-10 * time.Second)},
				},
			},
		}}
		client := setup(append([]runtime.Object{&ourPod}, apps...)...)

		status := GenerateWavefrontStatus(client, wavefront)

		require.Equal(t, Healthy, status.Status)
		for _, resourceStatus := range status.ResourceStatuses {
			if resourceStatus.Name == util.LoggingName {
				require.Contains(t, resourceStatus.Status, "Running")
			}
		}
	})

	t.Run("handles when it has not been terminated", func(t *testing.T) {
		ourPod := *pod
		ourPod.Status.ContainerStatuses = []corev1.ContainerStatus{{}}
		client := setup(append([]runtime.Object{&ourPod}, apps...)...)

		wavefront.Spec.DataCollection.Logging.Enable = true

		require.NotPanics(t, func() {
			GenerateWavefrontStatus(client, wavefront)
		})
	})
}

func pastMaxInstallTime() time.Time {
	return time.Now().Add(-MaxInstallTime).Add(-time.Second * 10)
}

func setup(initObjs ...runtime.Object) kubernetes.Interface {
	return k8sfake.NewSimpleClientset(append(initObjs, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      util.OperatorName,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "controller-manager",
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
		},
	})...)
}

func getComponentStatusWithName(name string, componentStatuses []wf.ResourceStatus) wf.ResourceStatus {
	for _, componentStatus := range componentStatuses {
		if componentStatus.Name == name {
			return componentStatus
		}
	}
	return wf.ResourceStatus{}
}

func defaultWF() *wf.Wavefront {
	return &wf.Wavefront{
		Spec: wf.WavefrontSpec{
			Namespace: testNamespace,
		},
	}
}
