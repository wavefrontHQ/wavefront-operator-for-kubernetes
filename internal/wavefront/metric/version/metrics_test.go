package version_test

import (
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric/version"
)

func TestSender(t *testing.T) {
	t.Run("sends simple semantic versions to wavefront", func(t *testing.T) {
		ms, err := version.Metrics("somecluster", "2.1.3")

		require.NoError(t, err)
		require.Contains(t, ms, metric.Metric{
			Name:   "kubernetes.observability.version",
			Value:  2.010300,
			Source: "somecluster",
			Tags:   nil,
		})
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
