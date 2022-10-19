package metric

import (
	"strings"
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
	addr      string
	sender    Sender
	metrics   map[string]Metric
}

func NewConnection(newSender SenderFactory) *Connection {
	c := &Connection{
		newSender: newSender,
		metrics:   map[string]Metric{},
	}
	go startFlushLoop(c)
	return c
}

func startFlushLoop(c *Connection) {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			c.FlushMetrics()
		}
	}
}

func (c *Connection) connected() bool {
	return c.sender != nil
}

func (c *Connection) Connect(addr string) error {
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}
	if c.addr == addr {
		return nil
	}
	if c.connected() {
		c.Close()
	}
	sender, err := c.newSender(addr)
	if err != nil {
		return err
	}
	c.addr = addr
	c.sender = sender
	return nil
}

func (c *Connection) Send(metrics []Metric) {
	for _, metric := range metrics {
		c.metrics[metric.Name] = metric
	}
}

func (c *Connection) FlushMetrics() {
	if c.sender == nil {
		return
	}

	for _, metric := range c.metrics {
		err := c.sender.SendMetric(metric.Name, metric.Value, 0, metric.Source, metric.Tags)
		if err != nil {
			log.Log.Error(err, "error sending metrics")
		}
	}

	err := c.sender.Flush()
	if err != nil {
		log.Log.Error(err, "error flushing metrics")
	}
}

func (c *Connection) Close() {
	if !c.connected() {
		return
	}
	c.sender.Close()
	c.sender = nil
	c.addr = ""
}
