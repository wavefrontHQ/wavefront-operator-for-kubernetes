package status

import (
	"fmt"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
	"strings"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
)

func Send(client senders.MetricSender, clusterName string, status wf.WavefrontStatus) error {
	sends := []func(senders.MetricSender, wf.WavefrontStatus, string) error{
		sendMetricsStatus,
		sendLoggingStatus,
		sendProxyStatus,
		sendOperatorStatus,
	}
	for _, send := range sends {
		err := send(client, status, clusterName)
		if err != nil {
			return err
		}
	}

	return nil
}

func sendOperatorStatus(client senders.MetricSender, status wf.WavefrontStatus, clusterName string) error {
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

	err := sendMetric(client, healthy, clusterName, tags, "kubernetes.observability.status")
	if err != nil {
		return err
	}
	return nil
}

func sendMetricsStatus(client senders.MetricSender, status wf.WavefrontStatus, clusterName string) error {
	return sendComponentStatus(client, clusterName, map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true}, "Metrics", status.ResourceStatuses)
}

func sendLoggingStatus(client senders.MetricSender, status wf.WavefrontStatus, clusterName string) error {
	return sendComponentStatus(client, clusterName, map[string]bool{util.LoggingName: true}, "Logging", status.ResourceStatuses)
}

func sendProxyStatus(client senders.MetricSender, status wf.WavefrontStatus, clusterName string) error {
	return sendComponentStatus(client, clusterName, map[string]bool{util.ProxyName: true}, "Proxy", status.ResourceStatuses)
}

func sendComponentStatus(client senders.MetricSender, clusterName string, resourcesInComponent map[string]bool, componentName string, resourceStatuses []wf.ResourceStatus) error {
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
	return sendMetric(client, healthValue, clusterName, tags, fmt.Sprintf("kubernetes.observability.%s.status", strings.ToLower(componentName)))
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

func sendMetric(client senders.MetricSender, value float64, source string, tags map[string]string, name string) error {
	return client.SendMetric(name, value, 0, source, truncateTags(tags))
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
