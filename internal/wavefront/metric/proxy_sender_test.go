package metric_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
	"testing"
)

func TestWavefrontProxySender(t *testing.T) {
	t.Run("passes on wfsdk.Sender creation errors", func(t *testing.T) {
		_, err := metric.NewWavefrontSender("")

		assert.Error(t, err)
	})
}
