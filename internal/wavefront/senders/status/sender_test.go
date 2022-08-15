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

func TestStoreEmptyStatus(t *testing.T) {
	fakeStatusSender := NewTestStatusSender()
	fakeStatusSender.SendStatus(wf.WavefrontStatus{}, "my_cluster")
	assert.Equal(t, "", getMetrics(fakeStatusSender))
}

func getMetrics(sender *statusSender) string {
	return strings.TrimSpace(sender.wfSender.(*senders.TestSender).GetReceivedLines())
}
