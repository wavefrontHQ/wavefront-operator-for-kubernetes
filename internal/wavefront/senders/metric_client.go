package senders

type MetricClient interface {
	MetricSender
	Flush() error
	Close()
}

type MetricSender interface {
	SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error
}
