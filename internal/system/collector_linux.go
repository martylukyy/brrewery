//go:build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Collector struct {
	mu          sync.Mutex
	prevCPU     *cpuCounters
	prevMountIO map[string]mountIOSample
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (Info, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return Info{}, fmt.Errorf("hostname: %w", err)
	}

	uptime, err := readUptime()
	if err != nil {
		return Info{}, err
	}

	load, err := readLoadAvg()
	if err != nil {
		return Info{}, err
	}

	memory, err := readMemory()
	if err != nil {
		return Info{}, err
	}

	disks, err := c.readMonitoredDisks(uptime)
	if err != nil {
		return Info{}, err
	}

	network, err := readNetworkCounters()
	if err != nil {
		return Info{}, err
	}

	diskIO, err := readDiskIOCounters()
	if err != nil {
		return Info{}, err
	}

	cpuPercent, err := c.readCPUPercent()
	if err != nil {
		return Info{}, err
	}

	cpuName, err := readCPUModelName()
	if err != nil {
		return Info{}, err
	}

	info := Info{
		Hostname:      hostname,
		UptimeSeconds: uptime,
		CPUCount:      runtime.NumCPU(),
		CPUName:       cpuName,
		CPUPercent:    cpuPercent,
		Load:          load,
		Memory:        memory,
		Disks:         disks,
		Network:       network,
		DiskIO:        diskIO,
	}
	return info, nil
}

func (c *Collector) readMonitoredDisks(uptime float64) ([]DiskUsage, error) {
	mounts, err := monitoredFstabMounts()
	if err != nil {
		return nil, err
	}
	if len(mounts) == 0 {
		mounts = []string{"/"}
	}

	// fstab lists intended mounts; only report filesystems that are actually
	// mounted (and each backing device once) so statfs fall-through to root and
	// shared devices don't inflate the reported usage. If mountinfo is somehow
	// unreadable, fall back to the unfiltered fstab list.
	if deviceByMount, err := mountedDeviceByMount(); err == nil {
		mounts = activeMonitoredMounts(mounts, deviceByMount)
	}

	disks := make([]DiskUsage, 0, len(mounts))
	for _, mount := range mounts {
		usage, err := readDiskUsage(mount)
		if err != nil {
			continue
		}

		ioBusy, err := c.readMountIOBusy(mount, uptime)
		if err != nil {
			ioBusy = 0
		}
		usage.IOBusyPercent = ioBusy

		ioCounters, err := readMountIOCounters(mount)
		if err == nil {
			usage.IOReadBytes = ioCounters.ReadBytes
			usage.IOWriteBytes = ioCounters.WriteBytes
		}
		disks = append(disks, usage)
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("no mounted filesystems from fstab")
	}

	return disks, nil
}

func readUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("read uptime: %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("parse uptime: empty")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse uptime: %w", err)
	}
	return seconds, nil
}

func readLoadAvg() (LoadAvg, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return LoadAvg{}, fmt.Errorf("read loadavg: %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return LoadAvg{}, fmt.Errorf("parse loadavg: expected 3 values")
	}
	one, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("parse load 1m: %w", err)
	}
	five, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("parse load 5m: %w", err)
	}
	fifteen, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("parse load 15m: %w", err)
	}
	return LoadAvg{One: one, Five: five, Fifteen: fifteen}, nil
}

func readMemory() (Memory, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return Memory{}, fmt.Errorf("read meminfo: %w", err)
	}
	defer file.Close()

	values := map[string]uint64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		fields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(fields) < 1 {
			continue
		}
		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		values[key] = kb * 1024
	}
	if err := scanner.Err(); err != nil {
		return Memory{}, fmt.Errorf("scan meminfo: %w", err)
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	if available == 0 {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	if total == 0 {
		return Memory{}, fmt.Errorf("parse meminfo: missing MemTotal")
	}

	used := total - available
	if used > total {
		used = total
	}

	return Memory{
		TotalBytes:     total,
		AvailableBytes: available,
		UsedBytes:      used,
		UsedPercent:    percent(used, total),
	}, nil
}

func readDiskUsage(mount string) (DiskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mount, &stat); err != nil {
		return DiskUsage{}, fmt.Errorf("statfs %s: %w", mount, err)
	}

	usage := diskUsageFromStatfs(stat.Blocks, stat.Bfree, stat.Bavail, uint64(stat.Bsize))
	usage.Mount = mount
	return usage, nil
}

// diskUsageFromStatfs derives df(1)-compatible figures from raw statfs block
// counts. Used space is total minus *all* free blocks (Bfree), not just the
// blocks available to unprivileged users (Bavail): the difference is the
// root-reserved blocks (~5% on ext4), which are free, not used. The percentage
// basis likewise excludes reserved blocks (used / (used + available)), matching
// `df` Use% and tools like gdu instead of reporting reserved space as used.
func diskUsageFromStatfs(blocks, bfree, bavail, bsize uint64) DiskUsage {
	total := blocks * bsize
	available := bavail * bsize

	var used uint64
	if blocks > bfree {
		used = (blocks - bfree) * bsize
	}

	return DiskUsage{
		TotalBytes:     total,
		UsedBytes:      used,
		AvailableBytes: available,
		UsedPercent:    percent(used, used+available),
	}
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}
