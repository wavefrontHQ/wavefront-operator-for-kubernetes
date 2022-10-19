package status_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/health"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric/status"

	wf "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/api/v1alpha1"
)

func TestMetrics(t *testing.T) {
	t.Run("have common attributes", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{})

		require.NoError(t, err)
		testhelper.RequireAllMetricsHaveCommonAttributes(t, ms, "my_cluster")
	})

	t.Run("defaults to unhealthy when status is empty", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{})

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.UNHEALTHY_VALUE)
	})

	t.Run("has installing status when operator status is installing", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: health.Installing, Message: "Installing Components"})

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.INSTALLING_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "Installing")
		testhelper.RequireMetricTag(t, m, "message", "Installing Components")
	})

	t.Run("has healthy status when operator status is healthy", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: "Healthy", Message: "1/1 components are healthy"})

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.HEALTHY_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "Healthy")
		testhelper.RequireMetricTag(t, m, "message", "1/1 components are healthy")
	})

	t.Run("has unhealthy status when operator status is unhealthy", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{Status: "Unhealthy", Message: "0/1 components are healthy"})

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.UNHEALTHY_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "Unhealthy")
		testhelper.RequireMetricTag(t, m, "message", "0/1 components are healthy")
	})

	t.Run("metrics component", func(t *testing.T) {
		ReportsSubComponentMetric(t, "Metrics", []string{util.ClusterCollectorName, util.NodeCollectorName})
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

	for _, testResourceName := range resourceNames {
		t.Run(fmt.Sprintf("has healthy status when %s is healthy", testResourceName), func(t *testing.T) {
			wfStatus := wf.WavefrontStatus{Status: health.Healthy}
			for _, resourceName := range resourceNames {
				wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
					Name:    resourceName,
					Healthy: true,
				})
			}

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			m := testhelper.RequireMetric(t, ms, metricName)
			testhelper.RequireMetricValue(t, m, status.HEALTHY_VALUE)
			testhelper.RequireMetricTag(t, m, "status", "healthy")
			testhelper.RequireMetricTag(t, m, "message", fmt.Sprintf("%s component is healthy", componentName))
		})
	}

	for _, testResourceName := range resourceNames {
		t.Run(fmt.Sprintf("has unhealthy status when %s is unhealthy", testResourceName), func(t *testing.T) {
			errorStr := fmt.Sprintf("%s has an error", testResourceName)

			wfStatus := wf.WavefrontStatus{Status: health.Unhealthy}
			for _, resourceName := range resourceNames {
				if testResourceName == resourceName {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:    resourceName,
						Message: errorStr,
						Healthy: false,
					})
				} else {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:    resourceName,
						Healthy: true,
					})
				}
			}

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			m := testhelper.RequireMetric(t, ms, metricName)
			testhelper.RequireMetricValue(t, m, status.UNHEALTHY_VALUE)
			testhelper.RequireMetricTag(t, m, "status", "unhealthy")
			testhelper.RequireMetricTag(t, m, "message", errorStr)
		})
	}

	for _, testResourceName := range resourceNames {
		t.Run(fmt.Sprintf("has installing status when %s is unhealthy and is installing", testResourceName), func(t *testing.T) {
			errorStr := fmt.Sprintf("%s has an error", testResourceName)

			wfStatus := wf.WavefrontStatus{Status: health.Unhealthy}
			for _, resourceName := range resourceNames {
				if testResourceName == resourceName {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:       resourceName,
						Message:    errorStr,
						Healthy:    false,
						Installing: true,
					})
				} else {
					wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
						Name:    resourceName,
						Healthy: true,
					})
				}
			}

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			m := testhelper.RequireMetric(t, ms, metricName)
			testhelper.RequireMetricValue(t, m, status.INSTALLING_VALUE)
			testhelper.RequireMetricTag(t, m, "status", "installing")
			testhelper.RequireMetricTag(t, m, "message", errorStr)
		})
	}

	t.Run("has not enabled status if component statuses are not present", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Status:           health.Unhealthy,
			ResourceStatuses: []wf.ResourceStatus{},
		})

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, metricName)
		testhelper.RequireMetricValue(t, m, status.NOT_ENABLED_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "not enabled")
		testhelper.RequireMetricTag(t, m, "message", fmt.Sprintf("%s component is not enabled", componentName))
	})

	if len(resourceNames) > 1 {
		t.Run("has unhealthy status when all resources are unhealthy", func(t *testing.T) {
			wfStatus := wf.WavefrontStatus{Status: health.Unhealthy}
			var errorStrs []string
			for _, resourceName := range resourceNames {
				errorStr := fmt.Sprintf("%s has an error", resourceName)
				errorStrs = append(errorStrs, errorStr)
				wfStatus.ResourceStatuses = append(wfStatus.ResourceStatuses, wf.ResourceStatus{
					Name:    resourceName,
					Message: errorStr,
					Healthy: false,
				})
			}

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			m := testhelper.RequireMetric(t, ms, metricName)
			testhelper.RequireMetricValue(t, m, status.UNHEALTHY_VALUE)
			testhelper.RequireMetricTag(t, m, "status", "unhealthy")
			testhelper.RequireMetricTag(t, m, "message", strings.Join(errorStrs, "; "))
		})
	}
}
