package testhelper

import "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"

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
