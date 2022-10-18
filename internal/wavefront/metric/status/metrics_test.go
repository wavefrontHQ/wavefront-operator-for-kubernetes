package status_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric/status"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
)

func TestMetrics(t *testing.T) {
	t.Run("returns empty wavefront status", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{})
		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  status.UNHEALTHY_VALUE,
			Source: "my_cluster",
			Tags:   map[string]string{"cluster": "my_cluster"},
		})
	})

	t.Run("returns installing wavefront status", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: health.Installing, Message: "Installing Components"})
		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  status.INSTALLING_VALUE,
			Source: "my_cluster",
			Tags: map[string]string{
				"cluster": "my_cluster",
				"status":  "Installing",
				"message": "Installing Components",
			},
		})
	})

	t.Run("returns healthy wavefront status", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"})
		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  status.HEALTHY_VALUE,
			Source: "my_cluster",
			Tags: map[string]string{
				"cluster": "my_cluster",
				"status":  "Healthy",
				"message": "1/1 components are healthy",
			},
		})
	})

	t.Run("returns unhealthy wavefront status", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"})
		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  status.UNHEALTHY_VALUE,
			Source: "my_cluster",
			Tags: map[string]string{
				"cluster": "my_cluster",
				"status":  "Unhealthy",
				"message": "0/1 components are healthy",
			},
		})
	})

	t.Run("returns wavefront status with point tag exceeds length limit", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Status: "Unhealthy",
			Message: "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; " +
				"0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; ",
		})

		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   "kubernetes.observability.status",
			Value:  status.UNHEALTHY_VALUE,
			Source: "my_cluster",
			Tags: map[string]string{
				"cluster": "my_cluster",
				"status":  "Unhealthy",
				"message": "0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. Error: this is a dummy error message with its length exceeds 256 and characters; 0/1 components are healthy. E",
			},
		})
	})

	t.Run("metrics component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Metrics", []string{util.ClusterCollectorName, util.NodeCollectorName})

		t.Run("sends unhealthy status when cluster and node collector are unhealthy", func(t *testing.T) {
			ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{
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
			require.Contains(t, ms, metric.Metric{
				Name:   "kubernetes.observability.metrics.status",
				Value:  status.UNHEALTHY_VALUE,
				Source: "my_cluster",
				Tags: map[string]string{
					"cluster": "my_cluster",
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

	t.Run("returns healthy status", func(t *testing.T) {
		wfStatus := wf.WavefrontStatus{Status: health.Healthy}
		for _, componentName := range resourceNames {
			wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
				Name:    componentName,
				Healthy: true,
			})
		}

		ms, err := status.Metrics("my_cluster", wfStatus)

		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   metricName,
			Value:  status.HEALTHY_VALUE,
			Source: "my_cluster",
			Tags: map[string]string{
				"cluster": "my_cluster",
				"status":  "healthy",
				"message": fmt.Sprintf("%s component is healthy", componentName),
			},
		})
	})

	for _, testComponentName := range resourceNames {
		t.Run(fmt.Sprintf("returns unhealthy status when %s is unhealthy", testComponentName), func(t *testing.T) {
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

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			require.Contains(t, ms, metric.Metric{
				Name:   metricName,
				Value:  status.UNHEALTHY_VALUE,
				Source: "my_cluster",
				Tags: map[string]string{
					"cluster": "my_cluster",
					"status":  "unhealthy",
					"message": errorStr,
				},
			})
		})
	}

	for _, testComponentName := range resourceNames {
		t.Run(fmt.Sprintf("returns installing status when %s is unhealthy and is installing", testComponentName), func(t *testing.T) {
			errorStr := fmt.Sprintf("%s has an error", testComponentName)

			wfStatus := wf.WavefrontStatus{Status: health.Unhealthy}
			for _, componentName := range resourceNames {
				if testComponentName == componentName {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:       componentName,
						Message:    errorStr,
						Healthy:    false,
						Installing: true,
					})
				} else {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:    componentName,
						Healthy: true,
					})
				}
			}

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			require.Contains(t, ms, metric.Metric{
				Name:   metricName,
				Value:  status.INSTALLING_VALUE,
				Source: "my_cluster",
				Tags: map[string]string{
					"cluster": "my_cluster",
					"status":  "installing",
					"message": errorStr,
				},
			})
		})
	}

	t.Run("returns not enabled status if component statuses are not present", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Status:           health.Unhealthy,
			ResourceStatuses: []wf.ResourceStatus{},
		})

		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   metricName,
			Value:  status.NOT_ENABLED_VALUE,
			Source: "my_cluster",
			Tags: map[string]string{
				"cluster": "my_cluster",
				"status":  "not enabled",
				"message": fmt.Sprintf("%s component is not enabled", componentName),
			},
		})
	})
}
