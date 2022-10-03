package senders_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"
	"testing"
)

func TestWavefrontProxySender(t *testing.T) {
	t.Run("passes on wfsdk.Sender creation errors", func(t *testing.T) {
		_, err := senders.NewWavefrontClient("")

		assert.Error(t, err)
	})
}
