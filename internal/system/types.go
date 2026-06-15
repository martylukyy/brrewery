package system

type Info struct {
	Hostname      string          `json:"hostname"`
	UptimeSeconds float64         `json:"uptime_seconds"`
	CPUCount      int             `json:"cpu_count"`
	CPUName       string          `json:"cpu_name"`
	CPUPercent    float64         `json:"cpu_percent"`
	Load          LoadAvg         `json:"load"`
	Memory        Memory          `json:"memory"`
	Disks         []DiskUsage     `json:"disks"`
	Network       NetworkCounters `json:"network"`
}

// NetworkCounters are cumulative totals from /proc/net/dev (non-loopback interfaces).
type NetworkCounters struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

// DiskIOCounters are cumulative I/O totals for a mount's backing block device,
// read from /sys/block/<dev>/stat.
type DiskIOCounters struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadOps    uint64 `json:"read_ops"`
	WriteOps   uint64 `json:"write_ops"`
}

type LoadAvg struct {
	One     float64 `json:"1m"`
	Five    float64 `json:"5m"`
	Fifteen float64 `json:"15m"`
}

type Memory struct {
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	UsedPercent    float64 `json:"used_percent"`
}

type DiskUsage struct {
	Mount          string  `json:"mount"`
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedPercent    float64 `json:"used_percent"`
	IODevice       string  `json:"io_device,omitempty"`
	IOBusyPercent  float64 `json:"io_busy_percent,omitempty"`
	ReadBytes      uint64  `json:"read_bytes"`
	WriteBytes     uint64  `json:"write_bytes"`
	ReadOps        uint64  `json:"read_ops"`
	WriteOps       uint64  `json:"write_ops"`
}
