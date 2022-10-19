package metric_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
)

func TestTruncateTags(t *testing.T) {
	t.Run("truncates tags where name=value is greater than util.MaxTagLength", func(t *testing.T) {
		require.Equal(t,
			map[string]string{"k": "v"},
			metric.TruncateTags(3, map[string]string{"k": "vv"}),
		)
	})

	t.Run("does not truncate short tags", func(t *testing.T) {
		require.Equal(t,
			map[string]string{"k": "vv"},
			metric.TruncateTags(4, map[string]string{"k": "vv"}),
		)
	})
}
