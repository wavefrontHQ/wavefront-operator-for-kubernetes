package status

import (
	"fmt"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
	"strings"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
)

func Sender(clusterName string, status wf.WavefrontStatus) senders.Sender {
	return func(sendMetric senders.SendMetric) error {

		sends := []func(senders.SendMetric, wf.WavefrontStatus, string) error{
			sendMetricsStatus,
			sendLoggingStatus,
			sendProxyStatus,
			sendOperatorStatus,
		}
		for _, send := range sends {
			err := send(sendMetric, status, clusterName)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func sendOperatorStatus(sendMetric senders.SendMetric, status wf.WavefrontStatus, clusterName string) error {
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

	err := sendWithTruncate(sendMetric, healthy, clusterName, tags, "kubernetes.observability.status")
	if err != nil {
		return err
	}
	return nil
}

func sendMetricsStatus(sendMetric senders.SendMetric, status wf.WavefrontStatus, clusterName string) error {
	return sendComponentStatus(sendMetric, clusterName, map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true}, "Metrics", status.ResourceStatuses)
}

func sendLoggingStatus(sendMetric senders.SendMetric, status wf.WavefrontStatus, clusterName string) error {
	return sendComponentStatus(sendMetric, clusterName, map[string]bool{util.LoggingName: true}, "Logging", status.ResourceStatuses)
}

func sendProxyStatus(sendMetric senders.SendMetric, status wf.WavefrontStatus, clusterName string) error {
	return sendComponentStatus(sendMetric, clusterName, map[string]bool{util.ProxyName: true}, "Proxy", status.ResourceStatuses)
}

func sendComponentStatus(sendMetric senders.SendMetric, clusterName string, resourcesInComponent map[string]bool, componentName string, resourceStatuses []wf.ResourceStatus) error {
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
	return sendWithTruncate(sendMetric, healthValue, clusterName, tags, fmt.Sprintf("kubernetes.observability.%s.status", strings.ToLower(componentName)))
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

func sendWithTruncate(sendMetric senders.SendMetric, value float64, source string, tags map[string]string, name string) error {
	return sendMetric(name, value, 0, source, truncateTags(tags))
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
