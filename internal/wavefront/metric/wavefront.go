package metric

import "github.com/wavefronthq/wavefront-sdk-go/senders"

func WavefrontSenderFactory(options ...senders.Option) SenderFactory {
	return func(addr string) (Sender, error) {
		return senders.NewSender(addr, options...)
	}
}
