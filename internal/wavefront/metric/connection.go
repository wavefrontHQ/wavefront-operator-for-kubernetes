package metric

import (
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Sender interface {
	SendMetric(name string, value float64, ts int64, source string, tags map[string]string) error
	Flush() error
	Close()
}

type SenderFactory func(addr string) (Sender, error)

type Connection struct {
	newSender SenderFactory
	mu        sync.Mutex
	addr      string
	sender    Sender
	metrics   map[string]Metric
}

func NewConnection(newSender SenderFactory) *Connection {
	c := &Connection{
		newSender: newSender,
		metrics:   map[string]Metric{},
	}
	go flushLoop(c)
	return c
}

func flushLoop(c *Connection) {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			c.Flush()
		}
	}
}

func (c *Connection) connected() bool {
	return c.sender != nil
}

func (c *Connection) Connect(addr string) error {
	addr = normalizeAddr(addr)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.addr == addr {
		return nil
	}
	c.close()
	sender, err := c.newSender(addr)
	if err != nil {
		return err
	}
	c.addr = addr
	c.sender = sender
	return nil
}

func normalizeAddr(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "http://" + addr
}

func (c *Connection) Send(metrics []Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, metric := range metrics {
		c.metrics[metric.Name] = metric
	}
}

func (c *Connection) Flush() {
	sendBatch(c.extractBatch())
}

func sendBatch(sender Sender, metrics map[string]Metric) {
	if sender == nil {
		return
	}
	for _, metric := range metrics {
		err := sender.SendMetric(metric.Name, metric.Value, 0, metric.Source, metric.Tags)
		if err != nil {
			log.Log.Error(err, "error sending metrics")
		}
	}
	err := sender.Flush()
	if err != nil {
		log.Log.Error(err, "error flushing metrics")
	}
}

func (c *Connection) extractBatch() (Sender, map[string]Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dst := make(map[string]Metric, len(c.metrics))
	for key, metric := range c.metrics {
		dst[key] = metric
	}
	return c.sender, dst
}

func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.close()
}

// close is not thread safe and must only be called when already holding c.mu
func (c *Connection) close() {
	if c.connected() {
		c.sender.Close()
	}
	c.sender = nil
	c.addr = ""
}
