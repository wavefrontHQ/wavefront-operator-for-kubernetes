package testhelper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
)

func StubSenderFactory(sender metric.Sender, err error) metric.SenderFactory {
	return func(addr string) (metric.Sender, error) {
		return sender, err
	}
}

type StubSender struct {
	SendMetricErr error
	FlushErr      error
}

func (s *StubSender) SendMetric(_ string, _ float64, _ int64, _ string, _ map[string]string) error {
	return s.SendMetricErr
}

func (s *StubSender) Flush() error {
	return nil
}

func (s *StubSender) Close() {}

type MockSender struct {
	SentMetrics []metric.Metric
	Flushes     int
	Closes      int
}

func (e *MockSender) SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error {
	e.SentMetrics = append(e.SentMetrics, metric.Metric{
		Name:   name,
		Value:  value,
		Source: source,
		Tags:   tags,
	})
	return nil
}

func (e *MockSender) Flush() error {
	e.Flushes += 1
	return nil
}

func (e *MockSender) Close() {
	e.Closes += 1
}

func RequireMetric(t *testing.T, metrics []metric.Metric, metricName string) metric.Metric {
	t.Helper()
	for _, m := range metrics {
		if m.Name == metricName {
			return m
		}
	}
	t.Fatalf("could not find a metric with the name \"%s\"", metricName)
	return metric.Metric{}
}

func RequireMetricTag(t *testing.T, m metric.Metric, tagName, tagValue string) {
	t.Helper()
	require.Equalf(t, tagValue, m.Tags[tagName], "expected tag \"%s\" to equal \"%s\"", tagName, tagValue)
}

func RequireMetricValue(t *testing.T, m metric.Metric, value float64) {
	t.Helper()
	require.Equalf(t, value, m.Value, "expected metric value to equal %f", value)
}

func RequireAllMetricsHaveCommonAttributes(t *testing.T, ms []metric.Metric, clusterName string) {
	t.Helper()
	for _, m := range ms {
		require.Equal(t, clusterName, m.Source, "source must be the cluster name")
		require.Equal(t, clusterName, m.Tags["cluster"], "cluster tag must be the cluster name")
	}
}
