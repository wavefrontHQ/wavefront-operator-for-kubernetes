package senders

import (
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"strings"
)

func NewWavefrontMultiSender(wavefrontProxyAddress string) (MultiSender, error) {
	if !strings.HasPrefix("http://", wavefrontProxyAddress) {
		wavefrontProxyAddress = "http://" + wavefrontProxyAddress
	}
	sender, err := senders.NewSender(wavefrontProxyAddress)
	if err != nil {
		return nil, err
	}
	return sendMetrics(sender), nil
}
