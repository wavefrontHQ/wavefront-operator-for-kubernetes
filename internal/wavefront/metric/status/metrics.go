package status

import (
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
)

const (
	UNHEALTHY_VALUE = iota * 1.0
	INSTALLING_VALUE
	HEALTHY_VALUE
	NOT_ENABLED_VALUE
)

func Metrics(clusterName string, operatorVersion string, status wf.WavefrontStatus) ([]metric.Metric, error) {
	ms := []metric.Metric{
		metricsStatus(status),
		loggingStatus(status),
		proxyStatus(status),
	}

	componentStatuses := map[string]string{}
	for _, m := range ms {
		name := m.ComponentName
		if val, ok := m.Tags["status"]; ok {
			componentStatuses[name] = val
		}
	}

	ms = append(ms, integrationStatus(status, componentStatuses, operatorVersion))

	return metric.Common(clusterName, ms), nil
}

func integrationStatus(status wf.WavefrontStatus, componentStatuses map[string]string, operatorVersion string) metric.Metric {
	tags := map[string]string{}
	if len(operatorVersion) == 0 {
		tags["version"] = "unknown"
		log.Log.Info("operator version is not set")
	} else {
		tags["version"] = operatorVersion
	}
	if len(status.Message) > 0 {
		tags["message"] = status.Message
	}
	if len(status.Status) > 0 {
		tags["status"] = status.Status
	}

	healthy := UNHEALTHY_VALUE
	if status.Status == health.Installing {
		healthy = INSTALLING_VALUE
	} else if status.Status == health.Healthy {
		healthy = HEALTHY_VALUE
	}

	for component, componentStatus := range componentStatuses {
		tags[component] = componentStatus
	}

	return metric.Metric{Name: "kubernetes.observability.status", Value: healthy, Tags: tags}
}

func metricsStatus(status wf.WavefrontStatus) metric.Metric {
	return componentStatusMetric(
		map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true},
		"Metrics",
		status.ResourceStatuses,
	)
}

func loggingStatus(status wf.WavefrontStatus) metric.Metric {
	return componentStatusMetric(
		map[string]bool{util.LoggingName: true},
		"Logging",
		status.ResourceStatuses,
	)
}

func proxyStatus(status wf.WavefrontStatus) metric.Metric {
	return componentStatusMetric(
		map[string]bool{util.ProxyName: true},
		"Proxy",
		status.ResourceStatuses,
	)
}

func componentStatusMetric(resourcesInComponent map[string]bool, componentName string, resourceStatuses []wf.ResourceStatus) metric.Metric {
	componentStatuses := filterComponents(resourceStatuses, resourcesInComponent)
	var healthValue float64
	tags := map[string]string{}
	if !resourcesPresent(componentStatuses) {
		tags["status"] = health.NotEnabled
		tags["message"] = fmt.Sprintf("%s component is not enabled", componentName)
		healthValue = NOT_ENABLED_VALUE
	} else if resourcesHealthy(componentStatuses) {
		tags["status"] = health.Healthy
		tags["message"] = fmt.Sprintf("%s component is healthy", componentName)
		healthValue = HEALTHY_VALUE
	} else if resourceInstalling(componentStatuses) {
		tags["status"] = health.Installing
		tags["message"] = strings.Join(resourceMessages(componentStatuses), "; ")
		healthValue = INSTALLING_VALUE
	} else {
		tags["status"] = health.Unhealthy
		tags["message"] = strings.Join(resourceMessages(componentStatuses), "; ")
		healthValue = UNHEALTHY_VALUE
	}

	componentName = strings.ToLower(componentName)
	return metric.Metric{
		Name:          fmt.Sprintf("kubernetes.observability.%s.status", componentName),
		Value:         healthValue,
		Tags:          tags,
		ComponentName: componentName,
	}
}

func resourceInstalling(statuses []wf.ResourceStatus) bool {
	installing := false
	for _, status := range statuses {
		installing = installing || status.Installing
	}
	return installing
}

func resourceMessages(statuses []wf.ResourceStatus) []string {
	var messages []string
	for _, status := range statuses {
		if len(status.Message) > 0 {
			messages = append(messages, status.Message)
		}
	}
	return messages
}

func resourcesHealthy(statuses []wf.ResourceStatus) bool {
	healthy := true
	for _, status := range statuses {
		healthy = healthy && status.Healthy
	}
	return healthy
}

func resourcesPresent(statuses []wf.ResourceStatus) bool {
	present := false
	for range statuses {
		present = true
	}
	return present
}

func filterComponents(resourceStatuses []wf.ResourceStatus, resourcesInComponent map[string]bool) []wf.ResourceStatus {
	var filtered []wf.ResourceStatus
	for _, componentStatus := range resourceStatuses {
		if resourcesInComponent[componentStatus.Name] {
			filtered = append(filtered, componentStatus)
		}
	}
	return filtered
}
