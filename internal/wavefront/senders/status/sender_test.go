package status_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders/status"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"

	"github.com/stretchr/testify/assert"
	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
)

func TestWavefrontProxySender(t *testing.T) {
	t.Run("errors no host or port is supplied", func(t *testing.T) {
		_, err := status.NewWavefrontProxySender("")

		assert.EqualError(t, err, "error: host and port required")
	})

	t.Run("errors when the port is not supplied", func(t *testing.T) {
		_, err := status.NewWavefrontProxySender("somehostwithoutaport")

		assert.EqualError(t, err, "error parsing proxy port: port required")
	})

	t.Run("errors when the port is valid", func(t *testing.T) {
		_, err := status.NewWavefrontProxySender("somehost:notaport")

		assert.EqualError(t, err, "error parsing proxy port: strconv.Atoi: parsing \"notaport\": invalid syntax")
	})
}

func TestSender(t *testing.T) {
	t.Run("sends empty wavefront status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 0.000000 source=\"my_cluster\""))
		fakeStatusSender := status.NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends healthy wavefront status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 1.000000 source=\"my_cluster\" message=\"1/1 components are healthy\" status=\"Healthy\""))
		fakeStatusSender := status.NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends unhealthy wavefront status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy\" status=\"Unhealthy\""))
		fakeStatusSender := status.NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends wavefront status with point tag exceeds length limit", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E\" status=\"Unhealthy\""))
		fakeStatusSender := status.NewSender(expectedMetricLine)

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
		fakeStatusSender := status.NewSender(&testhelper.StubMetricClient{
			SendMetricError: errors.New("send error"),
		})

		assert.EqualError(t, fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy"}, "my_cluster"), "send error")
	})

	t.Run("reports an error when it fails to flush", func(t *testing.T) {
		fakeStatusSender := status.NewSender(&testhelper.StubMetricClient{
			FlushError: errors.New("flush error"),
		})

		assert.EqualError(t, fakeStatusSender.SendStatus(wf.WavefrontStatus{Status: "Healthy"}, "my_cluster"), "flush error")
	})

	t.Run("metrics component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Metrics", "metrics", []string{util.ClusterCollectorName, util.NodeCollectorName})

		t.Run("sends unhealthy status when cluster and node collector are unhealthy", func(t *testing.T) {
			expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"cluster collector has an error; node collector has an error\" status=\"Unhealthy\""))
			fakeStatusSender := status.NewSender(expectedMetricLine)

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
	})

	t.Run("logging component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Logging", "logging", []string{util.LoggingName})
	})

	t.Run("proxy component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Proxy", "proxy", []string{util.ProxyName})
	})
}

func ReportsSubComponentMetric(t *testing.T, groupName string, metricSegment string, componentNames []string) {
	metricName := fmt.Sprintf("kubernetes.operator-system.%s.status", metricSegment)

	t.Run("sends healthy status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(fmt.Sprintf("%s 1.000000 source=\"my_cluster\" message=\"%s component is healthy\" status=\"Healthy\"", metricName, groupName)))
		fakeStatusSender := status.NewSender(expectedMetricLine)

		wfStatus := wf.WavefrontStatus{Status: health.Healthy}
		for _, componentName := range componentNames {
			wfStatus.ComponentStatuses = append(wfStatus.ComponentStatuses, wf.ComponentStatus{
				Name:    componentName,
				Healthy: true,
			})
		}
		_ = fakeStatusSender.SendStatus(wfStatus, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	for _, testComponentName := range componentNames {
		t.Run(fmt.Sprintf("sends unhealthy status when %s is unhealthy", testComponentName), func(t *testing.T) {
			errorStr := fmt.Sprintf("%s has an error", testComponentName)
			expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(fmt.Sprintf("%s 0.000000 source=\"my_cluster\" message=\"%s\" status=\"Unhealthy\"", metricName, errorStr)))
			fakeStatusSender := status.NewSender(expectedMetricLine)
			wfStatus := wf.WavefrontStatus{Status: health.Unhealthy}
			for _, componentName := range componentNames {
				if testComponentName == componentName {
					wfStatus.ComponentStatuses = append(wfStatus.ComponentStatuses, wf.ComponentStatus{
						Name:    componentName,
						Message: errorStr,
						Healthy: false,
					})
				} else {
					wfStatus.ComponentStatuses = append(wfStatus.ComponentStatuses, wf.ComponentStatus{
						Name:    componentName,
						Healthy: true,
					})
				}
			}

			_ = fakeStatusSender.SendStatus(wfStatus, "my_cluster")

			expectedMetricLine.Verify(t)
		})
	}

	t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(fmt.Sprintf("%s 2.000000 source=\"my_cluster\" status=\"Not Enabled\"", metricName)))
		fakeStatusSender := status.NewSender(expectedMetricLine)

		_ = fakeStatusSender.SendStatus(
			wf.WavefrontStatus{
				Status:            health.Unhealthy,
				ComponentStatuses: []wf.ComponentStatus{},
			},
			"my_cluster",
		)

		expectedMetricLine.Verify(t)
	})
}
