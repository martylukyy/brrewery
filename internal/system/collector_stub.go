//go:build !linux

package system

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (Info, error) {
	return Info{}, ErrUnsupported
}
