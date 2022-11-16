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

	t.Run("has installing status when integration status is installing", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Message: "Installing Components",
			Status:  health.Installing,
			ResourceStatuses: append(generateHealthySubComponentMetric([]string{util.ClusterCollectorName, util.NodeCollectorName, util.LoggingName}), wf.ResourceStatus{
				Name:       util.ProxyName,
				Installing: true,
			}),
		})

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.INSTALLING_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "Installing")
		testhelper.RequireMetricTag(t, m, "message", "Installing Components")
		testhelper.RequireMetricTag(t, m, "metrics", "Healthy")
		testhelper.RequireMetricTag(t, m, "logging", "Healthy")
		testhelper.RequireMetricTag(t, m, "proxy", "Installing")
	})

	t.Run("has healthy status when integration status is healthy", func(t *testing.T) {
		wfStatus := wf.WavefrontStatus{
			Status:           "Healthy",
			Message:          "4/4 components are healthy",
			ResourceStatuses: generateHealthySubComponentMetric([]string{util.ClusterCollectorName, util.NodeCollectorName, util.LoggingName, util.ProxyName}),
		}

		ms, err := status.Metrics("my_cluster", wfStatus)

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.HEALTHY_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "Healthy")
		testhelper.RequireMetricTag(t, m, "message", "4/4 components are healthy")
		testhelper.RequireMetricTag(t, m, "metrics", "Healthy")
		testhelper.RequireMetricTag(t, m, "logging", "Healthy")
		testhelper.RequireMetricTag(t, m, "proxy", "Healthy")
	})

	t.Run("has unhealthy status when integration status is unhealthy", func(t *testing.T) {
		ms, err := status.Metrics("my_cluster", wf.WavefrontStatus{
			Status:  "Unhealthy",
			Message: "3/4 components are healthy",
			ResourceStatuses: append(generateHealthySubComponentMetric([]string{util.NodeCollectorName, util.LoggingName, util.ProxyName}), wf.ResourceStatus{
				Name:    util.ClusterCollectorName,
				Healthy: false,
			})},
		)

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.status")
		testhelper.RequireMetricValue(t, m, status.UNHEALTHY_VALUE)
		testhelper.RequireMetricTag(t, m, "status", "Unhealthy")
		testhelper.RequireMetricTag(t, m, "message", "3/4 components are healthy")
		testhelper.RequireMetricTag(t, m, "metrics", "Unhealthy")
		testhelper.RequireMetricTag(t, m, "logging", "Healthy")
		testhelper.RequireMetricTag(t, m, "proxy", "Healthy")
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
			wfStatus.ResourceStatuses = generateHealthySubComponentMetric(resourceNames)

			ms, err := status.Metrics("my_cluster", wfStatus)

			require.NoError(t, err)
			m := testhelper.RequireMetric(t, ms, metricName)
			testhelper.RequireMetricValue(t, m, status.HEALTHY_VALUE)
			testhelper.RequireMetricTag(t, m, "status", health.Healthy)
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
			testhelper.RequireMetricTag(t, m, "status", health.Unhealthy)
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
			testhelper.RequireMetricTag(t, m, "status", health.Installing)
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
		testhelper.RequireMetricTag(t, m, "status", health.NotEnabled)
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
			testhelper.RequireMetricTag(t, m, "status", health.Unhealthy)
			testhelper.RequireMetricTag(t, m, "message", strings.Join(errorStrs, "; "))
		})
	}
}

func generateHealthySubComponentMetric(resourceNames []string) []wf.ResourceStatus {
	statuses := make([]wf.ResourceStatus, len(resourceNames))

	for _, name := range resourceNames {
		statuses = append(statuses, wf.ResourceStatus{
			Name:    name,
			Healthy: true,
		})
	}

	return statuses
}
