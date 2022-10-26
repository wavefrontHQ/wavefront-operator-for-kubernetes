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
	MaxInstallTime = time.Minute * 5
)

func GenerateWavefrontStatus(appsV1 typedappsv1.AppsV1Interface, wavefront *wf.Wavefront) wf.WavefrontStatus {
	componentsToCheck := map[string]string{}

	if wavefront.Spec.DataExport.WavefrontProxy.Enable {
		componentsToCheck[util.ProxyName] = util.Deployment
	}

	if wavefront.Spec.DataCollection.Metrics.Enable {
		componentsToCheck[util.ClusterCollectorName] = util.Deployment
		componentsToCheck[util.NodeCollectorName] = util.DaemonSet
	}

	if wavefront.Spec.DataCollection.Logging.Enable {
		componentsToCheck[util.LoggingName] = util.DaemonSet
	}

	status := wf.WavefrontStatus{}
	var componentHealth []bool
	var unhealthyMessages []string
	var componentStatuses []wf.ResourceStatus
	var componentStatus wf.ResourceStatus

	for name, resourceType := range componentsToCheck {
		if resourceType == util.Deployment {
			componentStatus = deploymentStatus(appsV1.Deployments(wavefront.Spec.Namespace), name)
			componentStatuses = append(componentStatuses, componentStatus)
		}
		if resourceType == util.DaemonSet {
			componentStatus = daemonSetStatus(appsV1.DaemonSets(wavefront.Spec.Namespace), name)
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
	} else if wavefront.GetCreationTimestamp().Time.Add(MaxInstallTime).After(time.Now()) {
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

func deploymentStatus(deployments typedappsv1.DeploymentInterface, name string) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: name,
	}

	deployment, err := deployments.Get(context.Background(), name, v1.GetOptions{})

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

func daemonSetStatus(daemonsets typedappsv1.DaemonSetInterface, name string) wf.ResourceStatus {
	componentStatus := wf.ResourceStatus{
		Name: name,
	}
	daemonSet, err := daemonsets.Get(context.Background(), name, v1.GetOptions{})

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
