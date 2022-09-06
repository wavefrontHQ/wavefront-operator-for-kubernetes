package status

import (
	"fmt"
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

	err := statusSender.WavefrontSender.SendMetric("kubernetes.operator.status", healthy, 0, clusterName, tags)
	if err != nil {
		return err
	}
	return statusSender.WavefrontSender.Flush()
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
