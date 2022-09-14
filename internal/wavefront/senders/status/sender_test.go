package status

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
)

func TestWavefrontProxySender(t *testing.T) {
	t.Run("errors no host or port is supplied", func(t *testing.T) {
		_, err := NewWavefrontProxySender("")

		assert.EqualError(t, err, "error: host and port required")
	})

	t.Run("errors when the port is not supplied", func(t *testing.T) {
		_, err := NewWavefrontProxySender("somehostwithoutaport")

		assert.EqualError(t, err, "error parsing proxy port: port required")
	})

	t.Run("errors when the port is valid", func(t *testing.T) {
		_, err := NewWavefrontProxySender("somehost:notaport")

		assert.EqualError(t, err, "error parsing proxy port: strconv.Atoi: parsing \"notaport\": invalid syntax")
	})
}

func TestSender(t *testing.T) {
	t.Run("sends empty wavefront status", func(t *testing.T) {
		expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.status 0.000000 source=\"my_cluster\"")
		fakeStatusSender := NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends healthy wavefront status", func(t *testing.T) {
		expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.status 1.000000 source=\"my_cluster\" message=\"1/1 components are healthy\" status=\"Healthy\"")
		fakeStatusSender := NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends unhealthy wavefront status", func(t *testing.T) {
		expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy\" status=\"Unhealthy\"")
		fakeStatusSender := NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends wavefront status with point tag exceeds length limit", func(t *testing.T) {
		expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E\" status=\"Unhealthy\"")
		fakeStatusSender := NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{
			Status: "Unhealthy",
			Message: "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; "},
			"my_cluster",
		)

		expectedMetricLine.Verify(t)
	})

	t.Run("reports an error when it fails to send", func(t *testing.T) {
		fakeStatusSender := NewSender(&StubMetricSender{
			SendMetricError: errors.New("send error"),
		})

		assert.EqualError(t, fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy"}, "my_cluster"), "send error")
	})

	t.Run("reports an error when it fails to flush", func(t *testing.T) {
		fakeStatusSender := NewSender(&StubMetricSender{
			FlushError: errors.New("flush error"),
		})

		assert.EqualError(t, fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy"}, "my_cluster"), "flush error")
	})

	t.Run("for metrics", func(t *testing.T) {
		t.Run("sends healthy status", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.metrics.status 1.000000 source=\"my_cluster\" message=\"Metrics component is healthy\" status=\"Healthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends unhealthy status when cluster collector is unhealthy", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"cluster collector has an error\" status=\"Unhealthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends unhealthy status when node collector is unhealthy", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"node collector has an error\" status=\"Unhealthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends unhealthy status when cluster and node collector are unhealthy", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"cluster collector has an error; node collector has an error\" status=\"Unhealthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.metrics.status 2.000000 source=\"my_cluster\" status=\"Not Enabled\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:            health.Unhealthy,
					Message:           "",
					ComponentStatuses: []wf.ComponentStatus{},
				},
				"my_cluster",
			)

			expectedMetricLine.Verify(t)
		})
	})

	t.Run("for logging", func(t *testing.T) {
		t.Run("sends healthy status", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.logging.status 1.000000 source=\"my_cluster\" message=\"Logging component is healthy\" status=\"Healthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends unhealthy status when logger is unhealthy", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.logging.status 0.000000 source=\"my_cluster\" message=\"logger has an error\" status=\"Unhealthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.logging.status 2.000000 source=\"my_cluster\" status=\"Not Enabled\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:            health.Unhealthy,
					Message:           "",
					ComponentStatuses: []wf.ComponentStatus{},
				},
				"my_cluster",
			)

			expectedMetricLine.Verify(t)
		})
	})

	t.Run("for proxy", func(t *testing.T) {
		t.Run("sends healthy status", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.proxy.status 1.000000 source=\"my_cluster\" message=\"Proxy component is healthy\" status=\"Healthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends unhealthy status when proxy is unhealthy", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.proxy.status 0.000000 source=\"my_cluster\" message=\"proxy has an error\" status=\"Unhealthy\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
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

			expectedMetricLine.Verify(t)
		})

		t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
			expectedMetricLine := NewExpectedMetricClient("kubernetes.operator-system.proxy.status 2.000000 source=\"my_cluster\" status=\"Not Enabled\"")
			fakeStatusSender := NewSender(expectedMetricLine)

			_ = fakeStatusSender.SendStatus(
				wf.WavefrontStatus{
					Status:            health.Unhealthy,
					Message:           "",
					ComponentStatuses: []wf.ComponentStatus{},
				},
				"my_cluster",
			)

			expectedMetricLine.Verify(t)
		})
	})
}

type ExpectMetricSender struct {
	expectedMetricLine string

	actualMetricLines []string
}

func NewExpectedMetricClient(metricLine string) *ExpectMetricSender {
	return &ExpectMetricSender{
		expectedMetricLine: metricLine,
	}
}

func (e *ExpectMetricSender) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "%s %f source=\"%s\"", name, value, source)
	var tagNames []string
	for name := range tags {
		tagNames = append(tagNames, name)
	}
	sort.Strings(tagNames)
	for _, name := range tagNames {
		fmt.Fprintf(buf, " %s=\"%s\"", name, tags[name])
	}
	e.actualMetricLines = append(e.actualMetricLines, buf.String())
	return nil
}

func (e *ExpectMetricSender) Verify(t *testing.T) {
	t.Helper()
	assert.Contains(t, e.actualMetricLines, e.expectedMetricLine)
}

func (e ExpectMetricSender) Flush() error {
	return nil
}

func (e ExpectMetricSender) Close() {}

type StubMetricSender struct {
	SendMetricError error
	FlushError      error
}

func (s StubMetricSender) SendMetric(_ string, _ float64, _ int64, _ string, _ map[string]string) error {
	return s.SendMetricError
}

func (s StubMetricSender) Flush() error {
	return s.FlushError
}

func (s StubMetricSender) Close() {}
