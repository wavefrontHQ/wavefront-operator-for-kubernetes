package metric_test

import (
	"math"
	"strings"
	"testing"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
)

var tooLong = strings.Repeat("0123456789", int(math.Ceil(float64(util.MaxTagLength)/10)))[:util.MaxTagLength]

func TestCommon(t *testing.T) {
	t.Run("truncates tags", func(t *testing.T) {
		processedMetric := metric.Common("someCluster", []metric.Metric{{Name: "a", Tags: map[string]string{"longTag": tooLong}}})[0]
		require.Equal(t,
			tooLong[:util.MaxTagLength-len("longTag=")],
			processedMetric.Tags["longTag"],
		)
	})

	t.Run("sets source to cluster", func(t *testing.T) {
		processedMetrics := metric.Common("someCluster", []metric.Metric{{Name: "a"}})[0]
		require.Equal(t, "someCluster", processedMetrics.Source)
	})

	t.Run("sets cluster tag", func(t *testing.T) {
		processedMetrics := metric.Common("someCluster", []metric.Metric{{Name: "a"}})[0]
		require.Equal(t, "someCluster", processedMetrics.Tags["cluster"])
	})

	t.Run("long clusters get truncated", func(t *testing.T) {
		processedMetric := metric.Common(tooLong, []metric.Metric{{Name: "a"}})[0]
		require.Equal(t,
			tooLong[:util.MaxTagLength-len("cluster=")],
			processedMetric.Tags["cluster"],
		)
	})
}
