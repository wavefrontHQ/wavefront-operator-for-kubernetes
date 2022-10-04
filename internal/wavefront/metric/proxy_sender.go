package metric

import (
	"strings"

	wfsdk "github.com/wavefronthq/wavefront-sdk-go/senders"
)

func NewWavefrontSender(wavefrontProxyAddress string) (Sender, error) {
	if !strings.HasPrefix("http://", wavefrontProxyAddress) {
		wavefrontProxyAddress = "http://" + wavefrontProxyAddress
	}
	client, err := wfsdk.NewSender(wavefrontProxyAddress)
	if err != nil {
		return nil, err
	}
	return func(metrics []Metric) error {
		for _, metric := range metrics {
			err := client.SendMetric(metric.Name, metric.Value, 0, metric.Source, metric.Tags)
			if err != nil {
				return err
			}
		}
		return client.Flush()
	}, nil
}
