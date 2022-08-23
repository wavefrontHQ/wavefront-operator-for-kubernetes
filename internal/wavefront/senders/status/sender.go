package status

import (
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	wfsdk "github.com/wavefronthq/wavefront-sdk-go/senders"
)

type Sender interface {
	SendStatus(status wf.WavefrontStatus, clusterName string) error
	Close()
}

func NewStatusSender(wavefrontUrl string) (*statusSender, error) {
	wavefrontSender, err := wfsdk.NewSender(wavefrontUrl)
	if err != nil {
		return nil, err
	}
	return &statusSender{
		wavefrontSender,
	}, nil
}

type statusSender struct {
	wfSender wfsdk.Sender
}

func truncateMessage(message string) string {
	maxPointTagLength := 255 - len("=") - len("message")
	if len(message) >= maxPointTagLength {
		return message[0:maxPointTagLength]
	}
	return message
}

func (statusSender statusSender) SendStatus(status wf.WavefrontStatus, clusterName string) error {
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

    return statusSender.wfSender.SendMetric("kubernetes.operator.status", healthy, 0, clusterName, tags)
}

func (statusSender statusSender) Close() {
	statusSender.wfSender.Close()
}
