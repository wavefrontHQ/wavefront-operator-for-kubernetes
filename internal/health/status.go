package health

import (
	"fmt"
	strings "strings"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

const (
	Healthy   = "Healthy"
	Unhealthy = "Unhealthy"
)

func GenerateWavefrontStatus(appsV1 typedappsv1.AppsV1Interface, componentsToCheck map[string]string) wf.WavefrontStatus {
	status := wf.WavefrontStatus{}
	var componentHealth []bool
	var unhealthyMessages []string
	var componentStatuses []wf.ComponentStatus
	var componentStatus wf.ComponentStatus

	for name, resourceType := range componentsToCheck {
		if resourceType == "Deployment" {
			componentStatus = deploymentStatus(appsV1, name)
			componentStatuses = append(componentStatuses, componentStatus)
		}
		if resourceType == "DaemonSet" {
			componentStatus = daemonSetStatus(appsV1, name)
			componentStatuses = append(componentStatuses, componentStatus)
		}
		componentHealth = append(componentHealth, componentStatus.Healthy)
		if !componentStatus.Healthy && len(componentStatus.Message) > 0 {
			unhealthyMessages = append(unhealthyMessages, componentStatus.Message)
		}
	}

	status.ComponentStatuses = componentStatuses
	if boolCount(false, componentHealth...) == 0 {
		status.Status = Healthy
		status.Message = fmt.Sprintf("(%d/%d) wavefront components are healthy", boolCount(true, componentHealth...), len(componentHealth))
	} else {
		status.Status = Unhealthy
		status.Message = strings.Join(unhealthyMessages, "; ")
	}

	return status
}

func deploymentStatus(appsV1 typedappsv1.AppsV1Interface, name string) wf.ComponentStatus {
	componentStatus := wf.ComponentStatus{
		Name: name,
	}

	deployment, err := appsV1.Deployments("wavefront").Get(context.Background(), name, v1.GetOptions{})

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

func daemonSetStatus(appsV1 typedappsv1.AppsV1Interface, name string) wf.ComponentStatus {
	componentStatus := wf.ComponentStatus{
		Name: name,
	}
	daemonSet, err := appsV1.DaemonSets("wavefront").Get(context.Background(), name, v1.GetOptions{})

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

func resetDaemonSetStatus(daemonSetStatus *wf.DaemonSetStatus) {
	daemonSetStatus.DaemonSetName = ""
	daemonSetStatus.Healthy = false
	daemonSetStatus.Status = ""
	daemonSetStatus.Message = ""
	daemonSetStatus.DesiredNumberScheduled = 0
	daemonSetStatus.NumberReady = 0
}

func resetDeploymentStatus(deploymentStatus *wf.DeploymentStatus) {
	deploymentStatus.DeploymentName = ""
	deploymentStatus.Healthy = false
	deploymentStatus.Status = ""
	deploymentStatus.Message = ""
	deploymentStatus.Replicas = 0
	deploymentStatus.AvailableReplicas = 0
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
