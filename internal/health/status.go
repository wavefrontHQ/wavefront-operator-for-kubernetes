package health

import (
	"fmt"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func UpdateDeploymentStatus(appsV1 typedappsv1.AppsV1Interface, deploymentName string, deploymentStatus *wf.DeploymentStatus) {
	deploymentStatus.DeploymentName = deploymentName
	deployment, err := appsV1.Deployments("wavefront").Get(context.Background(), deploymentName, v1.GetOptions{})
	if err != nil {
		deploymentStatus.Healthy = false
		deploymentStatus.Message = err.Error()
		return
	}

	deploymentStatus.Replicas = deployment.Status.Replicas
	deploymentStatus.AvailableReplicas = deployment.Status.AvailableReplicas
	deploymentStatus.Status = fmt.Sprintf("Running (%d/%d)", deployment.Status.AvailableReplicas, deployment.Status.Replicas)

	if deployment.Status.AvailableReplicas < deployment.Status.Replicas {
		deploymentStatus.Healthy = false
		deploymentStatus.Message = fmt.Sprintf("not enough instances of %s are running (%d/%d)", deploymentStatus.DeploymentName, deployment.Status.AvailableReplicas, deployment.Status.Replicas)
	} else {
		deploymentStatus.Healthy = true
		deploymentStatus.Message = "healthy"
	}
}

func UpdateDaemonSetStatus(appsV1 typedappsv1.AppsV1Interface, daemonSetName string, daemonSetStatus *wf.DaemonSetStatus) {
	daemonSetStatus.DaemonSetName = daemonSetName
	daemonSet, err := appsV1.DaemonSets("wavefront").Get(context.Background(), daemonSetName, v1.GetOptions{})
	if err != nil {
		daemonSetStatus.Healthy = false
		daemonSetStatus.Message = err.Error()
		return
	}

	daemonSetStatus.DesiredNumberScheduled = daemonSet.Status.DesiredNumberScheduled
	daemonSetStatus.NumberReady = daemonSet.Status.NumberReady
	daemonSetStatus.Status = fmt.Sprintf("Running (%d/%d)", daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled)

	if daemonSet.Status.NumberReady < daemonSet.Status.DesiredNumberScheduled {
		daemonSetStatus.Healthy = false
		daemonSetStatus.Message = fmt.Sprintf("not enough instances of %s are running (%d/%d)", daemonSetStatus.DaemonSetName, daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled)
	} else {
		daemonSetStatus.Healthy = true
		daemonSetStatus.Message = "healthy"
	}
}
