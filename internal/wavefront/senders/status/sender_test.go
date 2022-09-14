package status

import (
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
)

func NewTestStatusSender() *StatusSender {
	statusSender, _ := NewStatusSender("myproxy.svc:2878")
	statusSender.WavefrontSender = senders.NewTestSender()
	return statusSender
}

func TestSendWfStatus(t *testing.T) {
	t.Run("sends empty wavefront status", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()
		fakeStatusSender.SendStatus(wf.WavefrontStatus{}, "my_cluster")
		assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.status 0.000000 source=\"my_cluster\"")
	})

	t.Run("sends healthy wavefront status", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()
		fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"}, "my_cluster")
		assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.status 1.000000 source=\"my_cluster\" message=\"1/1 components are healthy\" status=\"Healthy\"")

	})

	t.Run("sends unhealthy wavefront status", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()
		fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"}, "my_cluster")
		assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy\" status=\"Unhealthy\"")
	})

	t.Run("sends wavefront status with point tag exceeds length limit", func(t *testing.T) {
		fakeStatusSender := NewTestStatusSender()

		fakeStatusSender.SendStatus(wf.WavefrontStatus{
			Status: "Unhealthy",
			Message: "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; "},
			"my_cluster",
		)
		assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E\" status=\"Unhealthy\"")
	})

	// TODO test truncating messages for all components
	// TODO test error sending for all components

	t.Run("for metrics", func(t *testing.T) {
		t.Run("sends healthy status", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Healthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.ClusterCollectorName,
							Healthy: true,
						},
						{
							Name:    util.NodeCollectorName,
							Healthy: true,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.metrics.status 1.000000 source=\"my_cluster\" message=\"Metrics component is healthy\" status=\"Healthy\"")
		})

		t.Run("sends unhealthy status when cluster collector is unhealthy", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Unhealthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.ClusterCollectorName,
							Message: "cluster collector has an error",
							Healthy: false,
						},
						{
							Name:    util.NodeCollectorName,
							Healthy: true,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"cluster collector has an error\" status=\"Unhealthy\"")
		})

		t.Run("sends unhealthy status when node collector is unhealthy", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Unhealthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.ClusterCollectorName,
							Healthy: true,
						},
						{
							Name:    util.NodeCollectorName,
							Message: "node collector has an error",
							Healthy: false,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"node collector has an error\" status=\"Unhealthy\"")
		})

		t.Run("sends unhealthy status when cluster and node collector are unhealthy", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Unhealthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.ClusterCollectorName,
							Message: "cluster collector has an error",
							Healthy: false,
						},
						{
							Name:    util.NodeCollectorName,
							Message: "node collector has an error",
							Healthy: false,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"cluster collector has an error; node collector has an error\" status=\"Unhealthy\"")
		})

		t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:            health.Unhealthy,
					Message:           "",
					ComponentStatuses: []wf.ComponentStatus{},
				},
				"my_cluster",
			)

			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.metrics.status 2.000000 source=\"my_cluster\" status=\"Not Enabled\"")
		})
	})

	t.Run("for logging", func(t *testing.T) {
		t.Run("sends healthy status", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Healthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.LoggingName,
							Healthy: true,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.logging.status 1.000000 source=\"my_cluster\" message=\"Logging component is healthy\" status=\"Healthy\"")
		})

		t.Run("sends unhealthy status when logger is unhealthy", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Unhealthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.LoggingName,
							Message: "logger has an error",
							Healthy: false,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.logging.status 0.000000 source=\"my_cluster\" message=\"logger has an error\" status=\"Unhealthy\"")
		})

		t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:            health.Unhealthy,
					Message:           "",
					ComponentStatuses: []wf.ComponentStatus{},
				},
				"my_cluster",
			)

			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.logging.status 2.000000 source=\"my_cluster\" status=\"Not Enabled\"")
		})
	})

	t.Run("for proxy", func(t *testing.T) {
		t.Run("sends healthy status", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Healthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.ProxyName,
							Healthy: true,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.proxy.status 1.000000 source=\"my_cluster\" message=\"Proxy component is healthy\" status=\"Healthy\"")
		})

		t.Run("sends unhealthy status when proxy is unhealthy", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Unhealthy,
					Message: "",
					ComponentStatuses: []wf.ComponentStatus{
						{
							Name:    util.ProxyName,
							Message: "proxy has an error",
							Healthy: false,
						},
					},
				},
				"my_cluster",
			)
			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.proxy.status 0.000000 source=\"my_cluster\" message=\"proxy has an error\" status=\"Unhealthy\"")
		})

		t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
			fakeStatusSender := NewTestStatusSender()

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:            health.Unhealthy,
					Message:           "",
					ComponentStatuses: []wf.ComponentStatus{},
				},
				"my_cluster",
			)

			assert.Contains(t, getMetrics(fakeStatusSender), "Metric: kubernetes.operator-system.proxy.status 2.000000 source=\"my_cluster\" status=\"Not Enabled\"")
		})
	})
}

func getMetrics(sender *StatusSender) string {
	return strings.TrimSpace(sender.WavefrontSender.(*senders.TestSender).GetReceivedLines())
}
