package senders

type metricClient interface {
	SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error
	Flush() error
}

type SendMetric func(name string, value float64, ts int64, source string, tags map[string]string) error

type Sender func(sender SendMetric) error

type MultiSender func(senders ...Sender) error

func sendMetrics(client metricClient) MultiSender {
	return func(senders ...Sender) error {
		for _, send := range senders {
			err := send(client.SendMetric)
			if err != nil {
				return err
			}
		}
		return client.Flush()
	}
}

type Metric struct {
	Name   string
	Value  float64
	Source string
	Tags   map[string]string
}
