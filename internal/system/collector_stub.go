//go:build !linux

package system

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (Info, error) {
	return Info{}, ErrUnsupported
}

func (c *Collector) CollectIOCounters() (IOCounters, error) {
	return IOCounters{}, ErrUnsupported
}
