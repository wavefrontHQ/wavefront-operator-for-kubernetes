package health

import (
	"context"
	"fmt"
	strings "strings"
	"time"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

const (
	Healthy        = "Healthy"
	Unhealthy      = "Unhealthy"
	Installing     = "Installing"
	MaxInstallTime = time.Minute * 2
)

func GenerateWavefrontStatus(appsV1 typedappsv1.AppsV1Interface, componentsToCheck map[string]string, wavefrontStartTime time.Time) wf.WavefrontStatus {
	status := wf.WavefrontStatus{}
	var componentHealth []bool
	var unhealthyMessages []string
	var componentStatuses []wf.ResourceStatus
	var componentStatus wf.ResourceStatus

	for name, resourceType := range componentsToCheck {
		if resourceType == util.Deployment {
			componentStatus = deploymentStatus(appsV1, name)
			componentStatuses = append(componentStatuses, componentStatus)
		}
		if resourceType == util.DaemonSet {
			componentStatus = daemonSetStatus(appsV1, name)
			componentStatuses = append(componentStatuses, componentStatus)
		}
		componentHealth = append(componentHealth, componentStatus.Healthy)
		if !componentStatus.Healthy && len(componentStatus.Message) > 0 {
			unhealthyMessages = append(unhealthyMessages, componentStatus.Message)
		}
	}

	status.ResourceStatuses = componentStatuses
	if boolCount(false, componentHealth...) == 0 {
		status.Status = Healthy
		status.Message = "All components are healthy"
	} else if wavefrontStartTime.Add(MaxInstallTime).Before(time.Now()) {
		status.Status = Installing
		status.Message = "Installing components"
	} else {
		status.Status = Unhealthy
		status.Message = strings.Join(unhealthyMessages, "; ")
	}

	return status
}

func deploymentStatus(appsV1 typedappsv1.AppsV1Interface, name string) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: name,
	}

	deployment, err := appsV1.Deployments(util.Namespace).Get(context.Background(), name, v1.GetOptions{})

	if err != nil {
		componentStatus.Healthy = false
		componentStatus.Status = fmt.Sprintf("Not running")
		return componentStatus
	}

	componentStatus.Status = fmt.Sprintf("Running (%d/%d)", deployment.Status.AvailableReplicas, deployment.Status.Replicas)

	if deployment.Status.AvailableReplicas < deployment.Status.Replicas {
		componentStatus.Healthy = false
		componentStatus.Message = fmt.Sprintf("not enough instances of %s are running (%d/%d)", componentStatus.Name, deployment.Status.AvailableReplicas, deployment.Status.Replicas)
	} else {
		componentStatus.Healthy = true
	}
	return componentStatus
}

func daemonSetStatus(appsV1 typedappsv1.AppsV1Interface, name string) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: name,
	}
	daemonSet, err := appsV1.DaemonSets(util.Namespace).Get(context.Background(), name, v1.GetOptions{})

	if err != nil {
		componentStatus.Healthy = false
		componentStatus.Status = fmt.Sprintf("Not running")
		return componentStatus
	}

	componentStatus.Status = fmt.Sprintf("Running (%d/%d)", daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled)

	if daemonSet.Status.NumberReady < daemonSet.Status.DesiredNumberScheduled {
		componentStatus.Healthy = false
		componentStatus.Message = fmt.Sprintf("not enough instances of %s are running (%d/%d)", componentStatus.Name, daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled)
	} else {
		componentStatus.Healthy = true
	}

	return componentStatus
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
