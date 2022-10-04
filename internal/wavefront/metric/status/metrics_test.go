package status_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric/status"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
)

func TestSender(t *testing.T) {
	t.Run("sends empty wavefront status", func(t *testing.T) {
		metrics, err := status.Metrics("my_cluster", wf.WavefrontStatus{})
		require.NoError(t, err)
		require.Contains(t, metrics, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  0,
			Source: "my_cluster",
			Tags:   map[string]string{},
		})
	})

	t.Run("sends healthy wavefront status", func(t *testing.T) {
		metrics, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"})
		require.NoError(t, err)
		require.Contains(t, metrics, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  1,
			Source: "my_cluster",
			Tags: map[string]string{
				"status":  "Healthy",
				"message": "1/1 components are healthy",
			},
		})
	})

	t.Run("sends unhealthy wavefront status", func(t *testing.T) {
		metrics, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"})
		require.NoError(t, err)
		require.Contains(t, metrics, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  0,
			Source: "my_cluster",
			Tags: map[string]string{
				"status":  "Unhealthy",
				"message": "0/1 components are healthy",
			},
		})
	})

	t.Run("sends wavefront status with point tag exceeds length limit", func(t *testing.T) {
		metrics, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Status: "Unhealthy",
			Message: "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; ",
		})

		require.NoError(t, err)
		require.Contains(t, metrics, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  0,
			Source: "my_cluster",
			Tags: map[string]string{
				"status":  "Unhealthy",
				"message": "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E",
			},
		})
	})

	t.Run("metrics component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Metrics", []string{util.ClusterCollectorName, util.NodeCollectorName})

		t.Run("sends unhealthy status when cluster and node collector are unhealthy", func(t *testing.T) {
			metrics, err := status.Metrics("my_cluster", wf.WavefrontStatus{
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
			})

			require.NoError(t, err)
			require.Contains(t, metrics, metric.Metric{
				Name:   "kubernetes.observability.metrics.status",
				Value:  0,
				Source: "my_cluster",
				Tags: map[string]string{
					"status":  "unhealthy",
					"message": "cluster collector has an error; node collector has an error",
				},
			})
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
	metricName := fmt.Sprintf("kubernetes.observability.%s.status", strings.ToLower(componentName))

	t.Run("sends healthy status", func(t *testing.T) {
		wfStatus := wf.WavefrontStatus{Status: health.Healthy}
		for _, componentName := range resourceNames {
			wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
				Name:    componentName,
				Healthy: true,
			})
		}

		metrics, err := status.Metrics("my_cluster", wfStatus)

		require.NoError(t, err)
		require.Contains(t, metrics, metric.Metric{
			Name:   metricName,
			Value:  1,
			Source: "my_cluster",
			Tags: map[string]string{
				"status":  "healthy",
				"message": fmt.Sprintf("%s component is healthy", componentName),
			},
		})
	})

	for _, testComponentName := range resourceNames {
		t.Run(fmt.Sprintf("sends unhealthy status when %s is unhealthy", testComponentName), func(t *testing.T) {
			errorStr := fmt.Sprintf("%s has an error", testComponentName)

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

			metrics, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			require.Contains(t, metrics, metric.Metric{
				Name:   metricName,
				Value:  0,
				Source: "my_cluster",
				Tags: map[string]string{
					"status":  "unhealthy",
					"message": errorStr,
				},
			})
		})
	}

	t.Run("sends not enabled status if component statuses are not present", func(t *testing.T) {
		metrics, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Status:           health.Unhealthy,
			ResourceStatuses: []wf.ResourceStatus{},
		})

		require.NoError(t, err)
		require.Contains(t, metrics, metric.Metric{
			Name:   metricName,
			Value:  2,
			Source: "my_cluster",
			Tags: map[string]string{
				"status":  "not enabled",
				"message": fmt.Sprintf("%s component is not enabled", componentName),
			},
		})
	})
}
