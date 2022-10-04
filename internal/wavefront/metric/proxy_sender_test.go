package metric_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
)

func TestWavefrontProxySender(t *testing.T) {
	t.Run("passes on wfsdk.Sender creation errors", func(t *testing.T) {
		_, err := metric.NewWavefrontSender("")

		assert.Error(t, err)
	})
}
