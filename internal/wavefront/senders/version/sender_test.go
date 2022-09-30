package version_test

import (
	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders/version"
	"testing"
)

func TestSender(t *testing.T) {
	t.Run("sends simple semantic versions to wavefront", func(t *testing.T) {
		expectVersionClient := NewExpectVersionClient(2.013, "somecluster")
		version.NewSender(expectVersionClient).SendVersion("2.1.3", "somecluster")

		expectVersionClient.Verify(t)
	})
}

type ExpectedVersionClient struct {
	expectedVersion float64
	actualVersion   float64
	expectedCluster string
	actualCluster   string
}

func NewExpectVersionClient(expectedVersion float64, expectedCluster string) *ExpectedVersionClient {
	return &ExpectedVersionClient{
		expectedVersion: expectedVersion,
		expectedCluster: expectedCluster,
	}
}

func (e *ExpectedVersionClient) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
	e.actualVersion = value
	e.actualCluster = tags["cluster"]
	return nil
}

func (e *ExpectedVersionClient) Flush() error {
	return nil
}

func (e *ExpectedVersionClient) Close() {}

func (e *ExpectedVersionClient) Verify(t *testing.T) {
	require.Equal(t, e.expectedVersion, e.actualVersion)
	require.Equal(t, e.expectedCluster, e.actualCluster)
}
