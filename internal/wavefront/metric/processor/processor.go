package processor

import (
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/util"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
)

type Processor func([]metric.Metric) []metric.Metric

func Common(clusterName string) Processor {
	return Combine(
		SetSource(clusterName),
		SetTag("cluster", clusterName),
		TruncateTags,
	)
}

func Combine(processors ...Processor) Processor {
	return func(metrics []metric.Metric) []metric.Metric {
		for _, process := range processors {
			metrics = process(metrics)
		}
		return metrics
	}
}

func EachMetric(do func(metric.Metric) metric.Metric) Processor {
	return func(ms []metric.Metric) []metric.Metric {
		next := ms[:0]
		for _, m := range ms {
			next = append(next, do(m))
		}
		return next
	}
}

var TruncateTags = EachMetric(func(metric metric.Metric) metric.Metric {
	metric.Tags = truncateTags(metric.Tags)
	return metric
})

func truncateTags(tags map[string]string) map[string]string {
	for name := range tags {
		maxLen := util.MaxTagLength - len(name) - len("=")
		if len(tags[name]) > maxLen {
			tags[name] = tags[name][:maxLen]
		}
	}
	return tags
}

func SetTag(name, value string) Processor {
	return EachMetric(func(metric metric.Metric) metric.Metric {
		if metric.Tags == nil {
			metric.Tags = map[string]string{}
		}
		metric.Tags[name] = value
		return metric
	})
}

func SetSource(source string) Processor {
	return EachMetric(func(metric metric.Metric) metric.Metric {
		metric.Source = source
		return metric
	})
}
