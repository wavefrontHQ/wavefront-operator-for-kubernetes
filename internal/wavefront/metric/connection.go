package metric

import (
	"strings"
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
}

func NewConnection(newSender SenderFactory) *Connection {
	return &Connection{newSender: newSender}
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

func (c *Connection) Send(metrics []Metric) error {
	if c.sender == nil {
		return nil
	}
	for _, metric := range metrics {
		err := c.sender.SendMetric(metric.Name, metric.Value, 0, metric.Source, metric.Tags)
		if err != nil {
			return err
		}
	}
	return c.sender.Flush()
}

func (c *Connection) Close() {
	if !c.connected() {
		return
	}
	c.sender.Close()
	c.sender = nil
	c.addr = ""
}
