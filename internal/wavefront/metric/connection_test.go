package metric_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/testhelper"

	"github.com/wavefrontHQ/wavefront-operator-for-kubernetes/internal/wavefront/metric"
)

func TestConnection(t *testing.T) {
	t.Run("adds http:// prefix to addresses that don't have them", func(t *testing.T) {
		connectedAddr := ""

		_ = metric.NewConnection(func(addr string) (metric.Sender, error) {
			connectedAddr = addr
			return nil, nil
		}).Connect("example.com")

		require.Equal(t, "http://example.com", connectedAddr)
	})

	t.Run("does not add http:// to addresses that already have them", func(t *testing.T) {
		connectedAddr := ""

		_ = metric.NewConnection(func(addr string) (metric.Sender, error) {
			connectedAddr = addr
			return nil, nil
		}).Connect("http://example.com")

		require.Equal(t, "http://example.com", connectedAddr)
	})

	t.Run("does not add http:// to addresses that have an https:// prefix", func(t *testing.T) {
		connectedAddr := ""

		_ = metric.NewConnection(func(addr string) (metric.Sender, error) {
			connectedAddr = addr
			return nil, nil
		}).Connect("https://example.com")

		require.Equal(t, "https://example.com", connectedAddr)
	})

	t.Run("does not send metrics making a new sender fails", func(t *testing.T) {
		mockSender := &testhelper.MockSender{}
		conn := metric.NewConnection(testhelper.StubSenderFactory(mockSender, errors.New("could not create sender")))

		_ = conn.Connect("example.com")

		conn.Send([]metric.Metric{{Name: "some.metric"}})
		conn.Flush()

		require.Equal(t, 0, len(mockSender.SentMetrics))
	})

	t.Run("reports error when making a new sender fails", func(t *testing.T) {
		expectedErr := errors.New("could not create sender")
		conn := metric.NewConnection(testhelper.StubSenderFactory(&testhelper.StubSender{}, errors.New("could not create sender")))

		require.Error(t, conn.Connect("example.com"), expectedErr.Error())
	})

	t.Run("connecting to the same address multiple times does only creates a new sender the first time", func(t *testing.T) {
		var newSenders int
		conn := metric.NewConnection(func(addr string) (metric.Sender, error) {
			newSenders += 1
			return &testhelper.StubSender{}, nil
		})

		_ = conn.Connect("example.com")
		_ = conn.Connect("example.com")

		require.Equal(t, 1, newSenders)
	})

	t.Run("closes other connections when connecting to a new source", func(t *testing.T) {
		mockSenders := map[string]*testhelper.MockSender{
			"http://example.com/1": {},
			"http://example.com/2": {},
		}
		conn := metric.NewConnection(func(addr string) (metric.Sender, error) {
			return mockSenders[addr], nil
		})

		_ = conn.Connect("http://example.com/1")
		_ = conn.Connect("http://example.com/2")

		require.Equal(t, 1, mockSenders["http://example.com/1"].Closes)
		require.Equal(t, 0, mockSenders["http://example.com/2"].Closes)
	})

	t.Run("connecting to another source sends metrics to the newest source", func(t *testing.T) {
		mockSenders := map[string]*testhelper.MockSender{
			"http://example.com/1": {},
			"http://example.com/2": {},
		}
		conn := metric.NewConnection(func(addr string) (metric.Sender, error) {
			return mockSenders[addr], nil
		})

		_ = conn.Connect("http://example.com/1")
		_ = conn.Connect("http://example.com/2")

		conn.Send([]metric.Metric{{Name: "some.metric"}})
		conn.Flush()

		require.Equal(t, 0, len(mockSenders["http://example.com/1"].SentMetrics))
		require.Equal(t, 1, len(mockSenders["http://example.com/2"].SentMetrics))
	})

	t.Run("sends metrics to the wfsdk.Sender", func(t *testing.T) {
		metrics := []metric.Metric{{Name: "some.metric"}}
		mockSender := &testhelper.MockSender{}
		conn := NewTestConnection(mockSender)
		_ = conn.Connect("example.com")

		conn.Send(metrics)
		conn.Flush()

		require.Equal(t, metrics, mockSender.SentMetrics)
	})

	t.Run("flushes metrics on send", func(t *testing.T) {
		mockSender := &testhelper.MockSender{}
		conn := NewTestConnection(mockSender)
		_ = conn.Connect("example.com")

		conn.Send([]metric.Metric{{Name: "some.metric"}})
		conn.Flush()

		require.Equal(t, 1, mockSender.Flushes)
	})

	t.Run("does not send more metrics to the sender after closing", func(t *testing.T) {
		mockSender := &testhelper.MockSender{}
		conn := NewTestConnection(mockSender)
		conn.Close()

		conn.Send([]metric.Metric{{Name: "some.metric"}})
		conn.Flush()

		require.Equal(t, 0, len(mockSender.SentMetrics))
	})

	t.Run("closes the sender when closing the connection", func(t *testing.T) {
		mockSender := &testhelper.MockSender{}
		conn := NewTestConnection(mockSender)
		conn.Close()

		require.Equal(t, 1, mockSender.Closes)
	})

	t.Run("closing the connection without being connected does not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			metric.NewConnection(testhelper.StubSenderFactory(nil, nil)).Close()
		})
	})

	t.Run("creates a new sender for the same address after closing", func(t *testing.T) {
		mockSender := &testhelper.MockSender{}
		conn := NewTestConnection(mockSender)
		conn.Close()

		_ = conn.Connect("example.com")

		conn.Send([]metric.Metric{{Name: "some.metric"}})
		conn.Flush()

		require.Equal(t, 1, len(mockSender.SentMetrics))
	})

	t.Run("handles concurrency", func(t *testing.T) {
		conn := NewTestConnection(&testhelper.StubSender{})

		const runs = 100_000

		go func() {
			for i := 0; i < runs; i++ {
				require.NoError(t, conn.Connect(fmt.Sprintf("http://foo.bar/%d", i)))
			}
		}()

		go func() {
			for i := 0; i < runs; i++ {
				conn.Send([]metric.Metric{{Name: "a"}, {Name: "b"}})
			}
		}()

		go func() {
			for i := 0; i < runs; i++ {
				conn.Flush()
			}
		}()

		for i := 0; i < runs; i++ {
			conn.Close()
		}
	})
}

func NewTestConnection(sender metric.Sender) *metric.Connection {
	conn := metric.NewConnection(testhelper.StubSenderFactory(sender, nil))
	_ = conn.Connect("example.com")
	return conn
}
