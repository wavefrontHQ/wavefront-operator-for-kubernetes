package status

import (
	"fmt"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
	"strings"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
)

func Metrics(clusterName string, status wf.WavefrontStatus) ([]senders.Metric, error) {
	return []senders.Metric{
		metricsStatus(status, clusterName),
		loggingStatus(status, clusterName),
		proxyStatus(status, clusterName),
		sendOperatorStatus(status, clusterName),
	}, nil
}

func sendOperatorStatus(status wf.WavefrontStatus, clusterName string) senders.Metric {
	tags := map[string]string{}
	if len(status.Message) > 0 {
		tags["message"] = status.Message
	}
	if len(status.Status) > 0 {
		tags["status"] = status.Status
	}

	healthy := 0.0
	if status.Status == health.Healthy {
		healthy = 1.0
	}

	return metricWithTruncatedTags(healthy, clusterName, tags, "kubernetes.observability.status")
}

func metricsStatus(status wf.WavefrontStatus, clusterName string) senders.Metric {
	return componentStatusMetric(clusterName, map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true}, "Metrics", status.ResourceStatuses)
}

func loggingStatus(status wf.WavefrontStatus, clusterName string) senders.Metric {
	return componentStatusMetric(clusterName, map[string]bool{util.LoggingName: true}, "Logging", status.ResourceStatuses)
}

func proxyStatus(status wf.WavefrontStatus, clusterName string) senders.Metric {
	return componentStatusMetric(clusterName, map[string]bool{util.ProxyName: true}, "Proxy", status.ResourceStatuses)
}

func componentStatusMetric(clusterName string, resourcesInComponent map[string]bool, componentName string, resourceStatuses []wf.ResourceStatus) senders.Metric {
	componentStatuses := filterComponents(resourceStatuses, resourcesInComponent)
	var healthValue float64
	tags := map[string]string{}
	if !resourcesPresent(componentStatuses) {
		tags["status"] = "not enabled"
		tags["message"] = fmt.Sprintf("%s component is not enabled", componentName)
		healthValue = 2.0
	} else if resourcesHealthy(componentStatuses) {
		tags["status"] = "healthy"
		tags["message"] = fmt.Sprintf("%s component is healthy", componentName)
		healthValue = 1.0
	} else {
		tags["status"] = "unhealthy"
		tags["message"] = strings.Join(resourceMessages(componentStatuses), "; ")
		healthValue = 0.0
	}
	return metricWithTruncatedTags(healthValue, clusterName, tags, fmt.Sprintf("kubernetes.observability.%s.status", strings.ToLower(componentName)))
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

func metricWithTruncatedTags(value float64, source string, tags map[string]string, name string) senders.Metric {
	return senders.Metric{Name: name, Value: value, Source: source, Tags: truncateTags(tags)}
}

func truncateTags(tags map[string]string) map[string]string {
	for name := range tags {
		maxLen := util.MaxTagLength - len(name) - len("=")
		if len(tags[name]) > maxLen {
			tags[name] = tags[name][:maxLen]
		}
	}
	return tags
}
