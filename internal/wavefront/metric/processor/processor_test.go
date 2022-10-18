package processor_test

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric/processor"
)

var tooLong = strings.Repeat("0123456789", int(math.Ceil(float64(util.MaxTagLength)/10)))[:util.MaxTagLength]

func TestCommon(t *testing.T) {
	t.Run("truncates tags", func(t *testing.T) {
		processedMetric := processor.Common("someCluster")([]metric.Metric{{Name: "a", Tags: map[string]string{"longTag": tooLong}}})[0]
		require.Equal(t,
			tooLong[:util.MaxTagLength-len("longTag=")],
			processedMetric.Tags["longTag"],
		)
	})

	t.Run("sets source to cluster", func(t *testing.T) {
		processedMetrics := processor.Common("someCluster")([]metric.Metric{{Name: "a"}})[0]
		require.Equal(t, "someCluster", processedMetrics.Source)
	})

	t.Run("sets cluster tag", func(t *testing.T) {
		processedMetrics := processor.Common("someCluster")([]metric.Metric{{Name: "a"}})[0]
		require.Equal(t, "someCluster", processedMetrics.Tags["cluster"])
	})

	t.Run("long clusters get truncated", func(t *testing.T) {
		processedMetric := processor.Common(tooLong)([]metric.Metric{{Name: "a"}})[0]
		require.Equal(t,
			tooLong[:util.MaxTagLength-len("cluster=")],
			processedMetric.Tags["cluster"],
		)
	})
}

func TestCombine(t *testing.T) {
	t.Run("applies all processors", func(t *testing.T) {
		process := processor.Combine(
			processor.SetTag("aTag", "aValue"),
			processor.SetTag("bTag", "bValue"),
			processor.SetTag("cTag", "cValue"),
		)

		require.Equal(t,
			[]metric.Metric{{Tags: map[string]string{"aTag": "aValue", "bTag": "bValue", "cTag": "cValue"}}},
			process([]metric.Metric{{}}),
		)
	})

	t.Run("applies all processors in the order they are given", func(t *testing.T) {
		process := processor.Combine(
			processor.SetSource("source1"),
			processor.SetSource("source2"),
			processor.SetSource("source3"),
		)

		require.Equal(t,
			[]metric.Metric{{Source: "source3"}},
			process([]metric.Metric{{}}),
		)
	})
}

func TestEachMetric(t *testing.T) {
	t.Run("applies the function to each metric", func(t *testing.T) {
		incrementByOne := processor.EachMetric(func(metric metric.Metric) metric.Metric {
			metric.Value += 1.0
			return metric
		})

		require.Equal(t,
			[]metric.Metric{{Name: "a", Value: 1.0}, {Name: "b", Value: 2.0}},
			incrementByOne([]metric.Metric{{Name: "a", Value: 0.0}, {Name: "b", Value: 1.0}}),
		)
	})
}

func TestTruncateTags(t *testing.T) {
	t.Run("truncates tags where name=value is greater than util.MaxTagLength", func(t *testing.T) {
		require.Equal(t,
			[]metric.Metric{{Name: "a", Tags: map[string]string{"longTag": tooLong[:util.MaxTagLength-len("longTag=")]}}},
			processor.TruncateTags([]metric.Metric{{Name: "a", Tags: map[string]string{"longTag": tooLong}}}),
		)
	})

	t.Run("does not truncate short tags", func(t *testing.T) {
		require.Equal(t,
			[]metric.Metric{{Name: "a", Tags: map[string]string{"shortTag": "shortValue"}}},
			processor.TruncateTags([]metric.Metric{{Name: "a", Tags: map[string]string{"shortTag": "shortValue"}}}),
		)
	})
}

func TestSetTag(t *testing.T) {
	t.Run("overrides existing tags", func(t *testing.T) {
		require.Equal(t,
			[]metric.Metric{{Name: "a", Tags: map[string]string{"someTag": "newValue"}}},
			processor.SetTag("someTag", "newValue")([]metric.Metric{{Name: "a", Tags: map[string]string{"someTag": "oldValue"}}}),
		)
	})

	t.Run("sets the tag when there are no tags", func(t *testing.T) {
		require.Equal(t,
			[]metric.Metric{{Name: "a", Tags: map[string]string{"newTag": "newValue"}}},
			processor.SetTag("newTag", "newValue")([]metric.Metric{{Name: "a", Tags: nil}}),
		)
	})
}

func TestSetSource(t *testing.T) {
	require.Equal(t,
		[]metric.Metric{{Name: "a", Source: "newSource"}},
		processor.SetSource("newSource")([]metric.Metric{{Name: "a", Source: "oldSource"}}),
	)
}
