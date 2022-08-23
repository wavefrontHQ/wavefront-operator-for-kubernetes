package status

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
)

func NewTestStatusSender() *statusSender {
	statusSender, _ := NewStatusSender("http://myproxy.com")
	statusSender.wfSender = senders.NewTestSender()
	return statusSender
}

func TestSendWfStatus(t *testing.T) {
	t.Run("Test send empty wavefront status", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()
		fakeStatusSender.SendStatus(wf.WavefrontStatus{}, "my_cluster")
		assert.Equal(t, "Metric: kubernetes.operator.status 0.000000 source=\"my_cluster\"", getMetrics(fakeStatusSender))
	})

	t.Run("Test send healthy wavefront status", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()
		fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"}, "my_cluster")
		assert.Equal(t, "Metric: kubernetes.operator.status 1.000000 source=\"my_cluster\" message=\"1/1 components are healthy\" status=\"Healthy\"", getMetrics(fakeStatusSender))

	})

	t.Run("Test send unhealthy wavefront status", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()
		fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"}, "my_cluster")
		assert.Equal(t, "Metric: kubernetes.operator.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy\" status=\"Unhealthy\"", getMetrics(fakeStatusSender))
	})

	t.Run("Test send wavefront status with point tag exceeds length limit", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()

		fakeStatusSender.SendStatus(wf.WavefrontStatus{
			Status: "Unhealthy",
			Message: "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; "},
			"my_cluster",
		)
		assert.Equal(t, "Metric: kubernetes.operator.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E\" status=\"Unhealthy\"", getMetrics(fakeStatusSender))
	})


}

func getMetrics(sender *statusSender) string {
	return strings.TrimSpace(sender.wfSender.(*senders.TestSender).GetReceivedLines())
}
