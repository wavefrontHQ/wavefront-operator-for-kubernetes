package health

import (
	"context"
	"fmt"
	strings "strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
)

const (
	Healthy        = "Healthy"
	Unhealthy      = "Unhealthy"
	Installing     = "Installing"
	NotEnabled     = "Not Enabled"
	MaxInstallTime = time.Minute * 5
	OOMTimeout     = time.Minute * 5
)

func GenerateWavefrontStatus(objClient client.Client, wavefront *wf.Wavefront) wf.WavefrontStatus {
	var componentStatuses []wf.ResourceStatus
	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		componentStatuses = append(componentStatuses, deploymentStatus(
			objClient,
			util.ObjKey(wavefront.Namespace, util.ProxyName),
		))
	}
	if wavefront.Spec.DataCollection.Metrics.Enable {
		componentStatuses = append(componentStatuses, deploymentStatus(
			objClient,
			util.ObjKey(wavefront.Namespace, util.ClusterCollectorName),
		))
		componentStatuses = append(componentStatuses, daemonSetStatus(
			objClient,
			util.ObjKey(wavefront.Namespace, util.NodeCollectorName),
		))
	}
	if wavefront.Spec.DataCollection.Logging.Enable {
		componentStatuses = append(componentStatuses, daemonSetStatus(
			objClient,
			util.ObjKey(wavefront.Namespace, util.LoggingName),
		))
	}
	componentStatuses = append(componentStatuses, deploymentStatus(
		objClient,
		util.ObjKey(wavefront.Namespace, util.OperatorName),
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
	} else if time.Since(createdAt) < MaxInstallTime {
		status.Status = Installing
		status.Message = "Installing components"
		for i := range status.ResourceStatuses {
			status.ResourceStatuses[i].Installing = true
		}
	} else {
		status.Status = Unhealthy
		status.Message = strings.Join(unhealthyMessages, "; ")
	}
	return status
}

func deploymentStatus(objClient client.Client, key client.ObjectKey) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: key.Name,
	}
	var deployment appsv1.Deployment
	err := objClient.Get(context.Background(), key, &deployment)
	if err != nil {
		return reportNotRunning(componentStatus)
	}
	componentStatus = reportStatusFromApp(
		componentStatus,
		deployment.Status.AvailableReplicas,
		*deployment.Spec.Replicas,
	)
	componentStatus = reportStatusFromPod(
		componentStatus,
		objClient,
		key.Namespace,
		deployment.Labels["app.kubernetes.io/component"],
	)
	return componentStatus
}

func daemonSetStatus(objClient client.Client, key client.ObjectKey) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: key.Name,
	}
	var daemonset appsv1.DaemonSet
	err := objClient.Get(context.Background(), key, &daemonset)
	if err != nil {
		return reportNotRunning(componentStatus)
	}
	componentStatus = reportStatusFromApp(
		componentStatus,
		daemonset.Status.NumberReady,
		daemonset.Status.DesiredNumberScheduled,
	)
	componentStatus = reportStatusFromPod(
		componentStatus,
		objClient,
		key.Namespace,
		daemonset.Labels["app.kubernetes.io/component"],
	)
	return componentStatus
}

func reportNotRunning(componentStatus wf.ResourceStatus) wf.ResourceStatus {
	componentStatus.Healthy = false
	componentStatus.Status = "Not running"
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

func reportStatusFromPod(componentStatus wf.ResourceStatus, objClient client.Client, namespace string, labelComponent string) wf.ResourceStatus {
	var podsList corev1.PodList
	err := objClient.List(
		context.Background(),
		&podsList,
		client.InNamespace(namespace),
		componentPodSelector(labelComponent),
	)
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

func componentPodSelector(componentName string) client.MatchingLabels {
	return client.MatchingLabels{
		"app.kubernetes.io/name":      "wavefront",
		"app.kubernetes.io/component": componentName,
	}
}

func oomKilledRecently(terminated *corev1.ContainerStateTerminated) bool {
	return terminated != nil &&
		terminated.ExitCode == 137 &&
		time.Since(terminated.FinishedAt.Time) < OOMTimeout
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
