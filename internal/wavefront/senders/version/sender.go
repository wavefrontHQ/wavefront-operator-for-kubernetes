package version

import "github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/senders"

type Sender struct{}

func NewSender(client senders.MetricClient) *Sender {
	return &Sender{}
}

func (s *Sender) SendVersion(version string, cluster string) error {
	return nil
}
