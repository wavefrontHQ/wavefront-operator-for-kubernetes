package status_test

import (
	"errors"
	"fmt"
	"strings"
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
		sender := status.NewSender(expectedMetricLine)

		_ = sender.SendStatus(wf.WavefrontStatus{}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends healthy wavefront status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 1.000000 source=\"my_cluster\" message=\"1/1 components are healthy\" status=\"Healthy\""))
		sender := status.NewSender(expectedMetricLine)

		_ = sender.SendStatus(wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends unhealthy wavefront status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy\" status=\"Unhealthy\""))
		sender := status.NewSender(expectedMetricLine)

		_ = sender.SendStatus(wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"}, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	t.Run("sends wavefront status with point tag exceeds length limit", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.status 0.000000 source=\"my_cluster\" message=\"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E\" status=\"Unhealthy\""))
		sender := status.NewSender(expectedMetricLine)

		_ = sender.SendStatus(wf.WavefrontStatus{
			Status: "Unhealthy",
			Message: "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; "},
			"my_cluster",
		)

		expectedMetricLine.Verify(t)
	})

	t.Run("reports an error when it fails to send", func(t *testing.T) {
		sender := status.NewSender(&testhelper.StubMetricClient{
			SendMetricError: errors.New("send error"),
		})

		assert.EqualError(t, sender.SendStatus(wf.WavefrontStatus{Status: "Healthy"}, "my_cluster"), "send error")
	})

	t.Run("reports an error when it fails to flush", func(t *testing.T) {
		sender := status.NewSender(&testhelper.StubMetricClient{
			FlushError: errors.New("flush error"),
		})

		assert.EqualError(t, sender.SendStatus(wf.WavefrontStatus{Status: "Healthy"}, "my_cluster"), "flush error")
	})

	t.Run("metrics component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Metrics", []string{util.ClusterCollectorName, util.NodeCollectorName})

		t.Run("sends unhealthy status when cluster and node collector are unhealthy", func(t *testing.T) {
			expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine("kubernetes.operator-system.metrics.status 0.000000 source=\"my_cluster\" message=\"cluster collector has an error; node collector has an error\" status=\"unhealthy\""))
			sender := status.NewSender(expectedMetricLine)

			_ = sender.SendStatus(
				wf.WavefrontStatus{
					Status:  health.Unhealthy,
					Message: "",
					ResourceStatuses: []wf.ResourceStatus{
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
		ReportsSubComponentMetric(t, "Logging", []string{util.LoggingName})
	})

	t.Run("proxy component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Proxy", []string{util.ProxyName})
	})
}

func ReportsSubComponentMetric(t *testing.T, componentName string, resourceNames []string) {
	metricName := fmt.Sprintf("kubernetes.operator-system.%s.status", strings.ToLower(componentName))

	t.Run("sends healthy status", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(
			fmt.Sprintf("%s 1.000000 source=\"my_cluster\" message=\"%s component is healthy\" status=\"healthy\"", metricName, componentName),
		))
		sender := status.NewSender(expectedMetricLine)

		wfStatus := wf.WavefrontStatus{Status: health.Healthy}
		for _, componentName := range resourceNames {
			wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
				Name:    componentName,
				Healthy: true,
			})
		}
		_ = sender.SendStatus(wfStatus, "my_cluster")

		expectedMetricLine.Verify(t)
	})

	for _, testComponentName := range resourceNames {
		t.Run(fmt.Sprintf("sends unhealthy status when %s is unhealthy", testComponentName), func(t *testing.T) {
			errorStr := fmt.Sprintf("%s has an error", testComponentName)
			expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(
				fmt.Sprintf("%s 0.000000 source=\"my_cluster\" message=\"%s\" status=\"unhealthy\"", metricName, errorStr),
			))
			sender := status.NewSender(expectedMetricLine)
			wfStatus := wf.WavefrontStatus{Status: health.Unhealthy}
			for _, componentName := range resourceNames {
				if testComponentName == componentName {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:    componentName,
						Message: errorStr,
						Healthy: false,
					})
				} else {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:    componentName,
						Healthy: true,
					})
				}
			}

			_ = sender.SendStatus(wfStatus, "my_cluster")

			expectedMetricLine.Verify(t)
		})
	}

	t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(
			fmt.Sprintf("%s 2.000000 source=\"my_cluster\" message=\"%s component is not enabled\" status=\"not enabled\"", metricName, componentName),
		))
		sender := status.NewSender(expectedMetricLine)

		_ = sender.SendStatus(
			wf.WavefrontStatus{
				Status:           health.Unhealthy,
				ResourceStatuses: []wf.ResourceStatus{},
			},
			"my_cluster",
		)

		expectedMetricLine.Verify(t)
	})
}
