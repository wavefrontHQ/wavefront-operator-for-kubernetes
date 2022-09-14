package status

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	wfsdk "github.com/wavefronthq/wavefront-sdk-go/senders"
)

type MetricClient interface {
	SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error
	Flush() error
	Close()
}

type Sender struct {
	client MetricClient
}

func NewWavefrontProxySender(wavefrontProxyAddress string) (*Sender, error) {
	if len(wavefrontProxyAddress) == 0 {
		return nil, errors.New("error: host and port required")
	}
	parts := strings.Split(wavefrontProxyAddress, ":")
	if len(parts) < 2 {
		return nil, errors.New("error parsing proxy port: port required")
	}
	host, portStr := parts[0], parts[1]
	port, err := strconv.Atoi(portStr)

	if err != nil {
		return nil, fmt.Errorf("error parsing proxy port: %s", err.Error())
	}

	client, err := wfsdk.NewProxySender(&wfsdk.ProxyConfiguration{
		Host:        host,
		MetricsPort: port,
	})

	if err != nil {
		return nil, err
	}

	return NewSender(client), nil
}

func NewSender(client MetricClient) *Sender {
	return &Sender{client: client}
}

func (s Sender) SendStatus(status wf.WavefrontStatus, clusterName string) error {
	sends := []func(wf.WavefrontStatus, string) error{
		s.sendMetricsStatus,
		s.sendLoggingStatus,
		s.sendProxyStatus,
		s.sendOperatorStatus,
	}
	for _, send := range sends {
		err := send(status, clusterName)
		if err != nil {
			return err
		}
	}
	return s.client.Flush()
}

func (s Sender) sendOperatorStatus(status wf.WavefrontStatus, clusterName string) error {
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

	err := s.sendMetric("kubernetes.operator-system.status", healthy, clusterName, tags)
	if err != nil {
		return err
	}
	return nil
}

func (s Sender) sendMetricsStatus(status wf.WavefrontStatus, clusterName string) error {
	return s.sendComponentStatus(
		status.ResourceStatuses,
		clusterName,
		map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true},
		"Metrics",
	)
}

func (s Sender) sendLoggingStatus(status wf.WavefrontStatus, clusterName string) error {
	return s.sendComponentStatus(
		status.ResourceStatuses,
		clusterName,
		map[string]bool{util.LoggingName: true},
		"Logging",
	)
}

func (s Sender) sendProxyStatus(status wf.WavefrontStatus, clusterName string) error {
	return s.sendComponentStatus(
		status.ResourceStatuses,
		clusterName,
		map[string]bool{util.ProxyName: true},
		"Proxy",
	)
}

func (s Sender) sendComponentStatus(resourceStatuses []wf.ResourceStatus, clusterName string, resourcesInComponent map[string]bool, componentName string) error {
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
	return s.sendMetric(fmt.Sprintf("kubernetes.operator-system.%s.status", strings.ToLower(componentName)), healthValue, clusterName, tags)
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

func (s Sender) sendMetric(name string, value float64, source string, tags map[string]string) error {
	return s.client.SendMetric(name, value, 0, source, truncateTags(tags))
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

func (s Sender) Close() {
	s.client.Close()
}
