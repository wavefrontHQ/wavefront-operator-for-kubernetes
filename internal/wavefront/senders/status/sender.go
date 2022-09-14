package status

import (
	"fmt"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"strconv"
	"strings"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	wfsdk "github.com/wavefronthq/wavefront-sdk-go/senders"
)

type StatusSender struct {
	WavefrontSender wfsdk.Sender
}

func NewStatusSender(wavefrontProxyAddress string) (*StatusSender, error) {

	s := strings.Split(wavefrontProxyAddress, ":")
	host, portStr := s[0], s[1]
	port, err := strconv.Atoi(portStr)

	if err != nil {
		return nil, fmt.Errorf("error parsing proxy port: %s", err.Error())
	}

	wavefrontSender, err := wfsdk.NewProxySender(&wfsdk.ProxyConfiguration{
		Host:        host,
		MetricsPort: port,
	})

	if err != nil {
		return nil, err
	}
	return &StatusSender{
		wavefrontSender,
	}, nil
}

func (statusSender StatusSender) SendStatus(status wf.WavefrontStatus, clusterName string) error {
	_ = statusSender.sendMetricsStatus(status, clusterName)
	_ = statusSender.sendLoggingStatus(status, clusterName)
	_ = statusSender.sendProxyStatus(status, clusterName)

	err := statusSender.sendOperatorStatus(status, clusterName)
	if err != nil {
		return err
	}
	return statusSender.WavefrontSender.Flush()
}

func (statusSender StatusSender) sendOperatorStatus(status wf.WavefrontStatus, clusterName string) error {
	tags := map[string]string{
		"cluster": clusterName,
	}
	if len(status.Message) > 0 {
		tags["message"] = truncateMessage(status.Message)
	}
	if len(status.Status) > 0 {
		tags["status"] = status.Status
	}

	healthy := 0.0
	if status.Status == health.Healthy {
		healthy = 1.0
	}

	err := statusSender.WavefrontSender.SendMetric("kubernetes.operator-system.status", healthy, 0, clusterName, tags)
	if err != nil {
		return err
	}
	return nil
}

func (statusSender StatusSender) sendMetricsStatus(status wf.WavefrontStatus, clusterName string) error {
	return statusSender.sendComponentStatus(
		status.ComponentStatuses,
		clusterName,
		map[string]bool{util.ClusterCollectorName: true, util.NodeCollectorName: true},
		"Metrics",
		"metrics",
	)
}

func (statusSender StatusSender) sendLoggingStatus(status wf.WavefrontStatus, clusterName string) error {
	return statusSender.sendComponentStatus(
		status.ComponentStatuses,
		clusterName,
		map[string]bool{util.LoggingName: true},
		"Logging",
		"logging",
	)
}

func (statusSender StatusSender) sendProxyStatus(status wf.WavefrontStatus, clusterName string) error {
	return statusSender.sendComponentStatus(
		status.ComponentStatuses,
		clusterName,
		map[string]bool{util.ProxyName: true},
		"Proxy",
		"proxy",
	)
}

func (statusSender StatusSender) sendComponentStatus(componentStatuses []wf.ComponentStatus, clusterName string, componentSet map[string]bool, name string, metricName string) error {
	tags := map[string]string{
		"cluster": clusterName,
	}
	present := false
	for _, componentStatus := range componentStatuses {
		if componentSet[componentStatus.Name] {
			present = true
		}
	}

	healthy := true
	for _, componentStatus := range componentStatuses {
		if componentSet[componentStatus.Name] {
			healthy = healthy && componentStatus.Healthy
		}
	}
	for _, componentStatus := range componentStatuses {
		if len(componentStatus.Message) > 0 && componentSet[componentStatus.Name] {
			if len(tags["message"]) > 0 {
				tags["message"] += "; "
			}
			tags["message"] += componentStatus.Message
		}
	}
	if healthy && present {
		tags["message"] = fmt.Sprintf("%s component is healthy", name)
	}

	if !present {
		tags["status"] = "Not Enabled"
	} else if healthy {
		tags["status"] = health.Healthy
	} else {
		tags["status"] = health.Unhealthy
	}
	var healthValue float64
	if !present {
		healthValue = 2.0
	} else if healthy {
		healthValue = 1.0
	}
	return statusSender.WavefrontSender.SendMetric(fmt.Sprintf("kubernetes.operator-system.%s.status", metricName), healthValue, 0, clusterName, tags)
}

func (statusSender StatusSender) Close() {
	statusSender.WavefrontSender.Close()
}

func truncateMessage(message string) string {
	maxPointTagLength := 255 - len("=") - len("message")
	if len(message) >= maxPointTagLength {
		return message[0:maxPointTagLength]
	}
	return message
}
