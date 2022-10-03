package version_test

import (
	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders/version"
	"testing"
)

func TestSender(t *testing.T) {
	t.Run("sends simple semantic versions to wavefront", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(
			"kubernetes.observability.version 2.010300 source=\"somecluster\"",
		))

		_ = version.Send(expectedMetricLine, "somecluster", "2.1.3")

		expectedMetricLine.Verify(t)
	})

	t.Run("rejects bad semvers", func(t *testing.T) {
		expectNoSend := testhelper.NewMockMetricClient(testhelper.AssertEmpty)

		require.EqualError(t,
			version.Send(expectNoSend, "somecluster", "2.a.b"),
			version.InvalidVersion.Error(),
		)

		expectNoSend.Verify(t)
	})

	t.Run("rejects minor versions larger than 2 digits", func(t *testing.T) {
		expectNoSend := testhelper.NewMockMetricClient(testhelper.AssertEmpty)

		require.EqualError(t,
			version.Send(expectNoSend, "somecluster", "2.100.0"),
			version.MinorVersionTooLarge.Error(),
		)

		expectNoSend.Verify(t)
	})

	t.Run("rejects patch versions larger than 2 digits", func(t *testing.T) {
		expectNoSend := testhelper.NewMockMetricClient(testhelper.AssertEmpty)

		require.EqualError(t,
			version.Send(expectNoSend, "somecluster", "2.0.100"),
			version.PatchVersionTooLarge.Error(),
		)

		expectNoSend.Verify(t)
	})
}
