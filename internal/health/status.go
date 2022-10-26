package health

import (
	"context"
	"fmt"
	strings "strings"
	"time"

	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	Healthy        = "Healthy"
	Unhealthy      = "Unhealthy"
	Installing     = "Installing"
	MaxInstallTime = time.Minute * 5
	OOMTimeout     = time.Minute * 5
)

type Client interface {
	AppsV1() appsv1.AppsV1Interface
	CoreV1() corev1.CoreV1Interface
}

func GenerateWavefrontStatus(client Client, wavefront *wf.Wavefront) wf.WavefrontStatus {
	var componentStatuses []wf.ResourceStatus
	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		componentStatuses = append(componentStatuses, deploymentStatus(
			client.AppsV1().Deployments(wavefront.Spec.Namespace),
			client.CoreV1().Pods(wavefront.Spec.Namespace),
			util.ProxyName,
		))
	}
	if wavefront.Spec.DataCollection.Metrics.Enable {
		componentStatuses = append(componentStatuses, deploymentStatus(
			client.AppsV1().Deployments(wavefront.Spec.Namespace),
			client.CoreV1().Pods(wavefront.Spec.Namespace),
			util.ClusterCollectorName,
		))
		componentStatuses = append(componentStatuses, daemonSetStatus(
			client.AppsV1().DaemonSets(wavefront.Spec.Namespace),
			client.CoreV1().Pods(wavefront.Spec.Namespace),
			util.NodeCollectorName,
		))
	}
	if wavefront.Spec.DataCollection.Logging.Enable {
		componentStatuses = append(componentStatuses, daemonSetStatus(
			client.AppsV1().DaemonSets(wavefront.Spec.Namespace),
			client.CoreV1().Pods(wavefront.Spec.Namespace),
			util.LoggingName,
		))
	}
	componentStatuses = append(componentStatuses, deploymentStatus(
		client.AppsV1().Deployments(wavefront.Spec.Namespace),
		client.CoreV1().Pods(wavefront.Spec.Namespace),
		util.OperatorName,
	))
	return reportAggregateStatus(componentStatuses, wavefront.GetCreationTimestamp().Time)
}

func reportAggregateStatus(componentStatuses []wf.ResourceStatus, createdAt time.Time) wf.WavefrontStatus {
	status := wf.WavefrontStatus{}
	var componentHealth []bool
	var unhealthyMessages []string

	for _, componentStatus := range componentStatuses {
		componentHealth = append(componentHealth, componentStatus.Healthy)
		if !componentStatus.Healthy && len(componentStatus.Message) > 0 {
			unhealthyMessages = append(unhealthyMessages, componentStatus.Message)
		}
	}

	status.ResourceStatuses = componentStatuses
	if boolCount(false, componentHealth...) == 0 {
		status.Status = Healthy
		status.Message = "All components are healthy"
	} else if time.Now().Sub(createdAt) < MaxInstallTime {
		status.Status = Installing
		status.Message = "Installing components"
		for i, _ := range status.ResourceStatuses {
			status.ResourceStatuses[i].Installing = true
		}
	} else {
		status.Status = Unhealthy
		status.Message = strings.Join(unhealthyMessages, "; ")
	}
	return status
}

func deploymentStatus(deployments appsv1.DeploymentInterface, pods corev1.PodInterface, name string) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: name,
	}
	deployment, err := deployments.Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return reportNotRunning(componentStatus)
	}
	componentStatus = reportStatusFromApp(
		componentStatus,
		deployment.Status.AvailableReplicas,
		deployment.Status.Replicas,
	)
	componentStatus = reportStatusFromPod(
		componentStatus,
		pods,
		deployment.Labels["app.kubernetes.io/component"],
	)
	return componentStatus
}

func daemonSetStatus(daemonsets appsv1.DaemonSetInterface, pods corev1.PodInterface, name string) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: name,
	}
	daemonSet, err := daemonsets.Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return reportNotRunning(componentStatus)
	}
	componentStatus = reportStatusFromApp(
		componentStatus,
		daemonSet.Status.NumberReady,
		daemonSet.Status.DesiredNumberScheduled,
	)
	componentStatus = reportStatusFromPod(
		componentStatus,
		pods,
		daemonSet.Labels["app.kubernetes.io/component"],
	)
	return componentStatus
}

func reportNotRunning(componentStatus wf.ResourceStatus) wf.ResourceStatus {
	componentStatus.Healthy = false
	componentStatus.Status = fmt.Sprintf("Not running")
	return componentStatus
}

func reportStatusFromApp(componentStatus wf.ResourceStatus, ready int32, desired int32) wf.ResourceStatus {
	componentStatus.Healthy = true
	componentStatus.Status = fmt.Sprintf("Running (%d/%d)", ready, desired)

	if ready < desired {
		componentStatus.Healthy = false
		componentStatus.Message = fmt.Sprintf(
			"not enough instances of %s are running (%d/%d)",
			componentStatus.Name, ready, desired,
		)
	}

	return componentStatus
}

func reportStatusFromPod(componentStatus wf.ResourceStatus, pods corev1.PodInterface, labelComponent string) wf.ResourceStatus {
	podsList, err := pods.List(context.Background(), metav1.ListOptions{
		LabelSelector: componentPodSelector(labelComponent).String(),
	})
	if err != nil {
		log.Log.Error(err, "error getting pod status")
		return componentStatus
	}
	for _, pod := range podsList.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if oomKilledRecently(containerStatus.LastTerminationState.Terminated) {
				componentStatus.Healthy = false
				componentStatus.Status = Unhealthy
				componentStatus.Message = fmt.Sprintf("%s OOMKilled in the last %s", labelComponent, OOMTimeout)
			}
		}
	}
	return componentStatus
}

func componentPodSelector(componentName string) labels.Selector {
	nameSelector, _ := labels.NewRequirement("app.kubernetes.io/name", selection.Equals, []string{"wavefront"})
	componentSelector, _ := labels.NewRequirement("app.kubernetes.io/component", selection.Equals, []string{componentName})
	return labels.NewSelector().Add(*nameSelector, *componentSelector)
}

func oomKilledRecently(terminated *apicorev1.ContainerStateTerminated) bool {
	return terminated != nil &&
		terminated.ExitCode == 137 &&
		time.Now().Sub(terminated.FinishedAt.Time) < OOMTimeout
}

func boolCount(b bool, statuses ...bool) int {
	n := 0
	for _, v := range statuses {
		if v == b {
			n++
		}
	}
	return n
}
