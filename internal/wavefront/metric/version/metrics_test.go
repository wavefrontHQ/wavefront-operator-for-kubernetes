package version_test

import (
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric/version"
)

func TestMetrics(t *testing.T) {
	t.Run("have common attributes", func(t *testing.T) {
		ms, err := version.Metrics("somecluster", "2.1.3")

		require.NoError(t, err)
		testhelper.RequireAllMetricsHaveCommonAttributes(t, ms, "somecluster")
	})

	t.Run("encodes a valid SemVer into a metric value", func(t *testing.T) {
		ms, err := version.Metrics("somecluster", "2.1.3")

		require.NoError(t, err)
		m := testhelper.RequireMetric(t, ms, "kubernetes.observability.version")
		testhelper.RequireMetricValue(t, m, 2.010300)
	})

	t.Run("rejects bad semvers", func(t *testing.T) {
		ms, err := version.Metrics("somecluster", "2.a.b")

		require.EqualError(t, err, version.InvalidVersion.Error())
		require.Empty(t, ms)
	})

	t.Run("rejects minor versions larger than 2 digits", func(t *testing.T) {
		ms, err := version.Metrics("somecluster", "2.100.0")

		require.EqualError(t, err, version.MinorVersionTooLarge.Error())
		require.Empty(t, ms)
	})

	t.Run("rejects patch versions larger than 2 digits", func(t *testing.T) {
		ms, err := version.Metrics("somecluster", "2.0.100")

		require.EqualError(t, err, version.PatchVersionTooLarge.Error())
		require.Empty(t, ms)
	})
}
