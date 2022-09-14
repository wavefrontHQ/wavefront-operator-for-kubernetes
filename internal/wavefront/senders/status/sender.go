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
		status.ComponentStatuses,
		clusterName,
		map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true},
		"Metrics",
		"metrics",
	)
}

func (s Sender) sendLoggingStatus(status wf.WavefrontStatus, clusterName string) error {
	return s.sendComponentStatus(
		status.ComponentStatuses,
		clusterName,
		map[string]bool{util.LoggingName: true},
		"Logging",
		"logging",
	)
}

func (s Sender) sendProxyStatus(status wf.WavefrontStatus, clusterName string) error {
	return s.sendComponentStatus(
		status.ComponentStatuses,
		clusterName,
		map[string]bool{util.ProxyName: true},
		"Proxy",
		"proxy",
	)
}

func (s Sender) sendComponentStatus(componentStatuses []wf.ComponentStatus, clusterName string, componentSet map[string]bool, name string, metricName string) error {
	tags := map[string]string{}
	present := false
	forComponents(componentStatuses, componentSet, func(_ wf.ComponentStatus) {
		present = true
	})

	healthy := true
	forComponents(componentStatuses, componentSet, func(status wf.ComponentStatus) {
		healthy = healthy && status.Healthy
	})

	var messages []string
	forComponents(componentStatuses, componentSet, func(status wf.ComponentStatus) {
		if len(status.Message) > 0 {
			messages = append(messages, status.Message)
		}
	})

	var healthValue float64
	if !present {
		tags["status"] = "not enabled"
		tags["message"] = fmt.Sprintf("%s component is not enabled", name)
		healthValue = 2.0
	} else if healthy {
		tags["status"] = "healthy"
		tags["message"] = fmt.Sprintf("%s component is healthy", name)
		healthValue = 1.0
	} else if present {
		tags["status"] = "unhealthy"
		tags["message"] = strings.Join(messages, "; ")
		healthValue = 0.0
	}

	return s.sendMetric(fmt.Sprintf("kubernetes.operator-system.%s.status", metricName), healthValue, clusterName, tags)
}

func forComponents(componentStatuses []wf.ComponentStatus, componentSet map[string]bool, do func(status wf.ComponentStatus)) {
	for _, componentStatus := range componentStatuses {
		if componentSet[componentStatus.Name] {
			do(componentStatus)
		}
	}
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
