package version_test

import (
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders/version"
	"testing"
)

func TestSender(t *testing.T) {
	t.Run("sends simple semantic versions to wavefront", func(t *testing.T) {
		expectedMetricLine := testhelper.NewMockMetricClient(testhelper.AssertContainsLine(
			"kubernetes.observability.version 2.013000 source=\"somecluster\"",
		))

		_ = version.NewSender(expectedMetricLine).SendVersion("2.1.3", "somecluster")

		expectedMetricLine.Verify(t)
	})
	// TODO: find sth doesn't match(bad version)
	// TODO: speculative test, parse float fails
}
