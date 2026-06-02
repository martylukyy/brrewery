//go:build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type mountIOSample struct {
	ioTimeMs uint64
	uptime   float64
}

func mountSourceDevice(mount string) (source string, deviceID string, err error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", "", fmt.Errorf("read mountinfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		mountPoint := strings.ReplaceAll(fields[4], "\\040", " ")
		if mountPoint != mount {
			continue
		}

		for i, field := range fields {
			if field != "-" {
				continue
			}
			if i+2 >= len(fields) {
				return "", "", fmt.Errorf("mount %q: missing source device", mount)
			}
			return fields[i+2], fields[2], nil
		}
		return "", "", fmt.Errorf("mount %q: missing mountinfo separator", mount)
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("scan mountinfo: %w", err)
	}

	return "", "", fmt.Errorf("mount %q: not found", mount)
}

// mountDiskIOStatPath returns the backing block-device stat file for a mount.
// Uses the top-level /sys/block/<disk>/stat path so whole-device I/O (e.g. fio on
// /dev/nvme0n1) is reflected, not only traffic attributed to a partition node.
func mountDiskIOStatPath(mount string) (string, error) {
	device, deviceID, err := mountSourceDevice(mount)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(device, "/dev/") {
		devname := filepath.Base(device)
		for len(devname) >= 2 {
			statPath := filepath.Join("/sys/block", devname, "stat")
			if fileReadable(statPath) {
				return statPath, nil
			}
			devname = devname[:len(devname)-1]
		}
	}

	// Some environments (e.g. CI containers) expose source device as /dev/root.
	// Fall back to resolving the mount major:minor through /sys/dev/block.
	resolved, err := resolveDiskFromDeviceID(deviceID)
	if err != nil {
		return "", fmt.Errorf("mount %q: block stat not found for %s: %w", mount, device, err)
	}

	statPath := filepath.Join("/sys/block", resolved, "stat")
	if !fileReadable(statPath) {
		return "", fmt.Errorf("mount %q: missing stat path %s", mount, statPath)
	}

	return statPath, nil
}

func fileReadable(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func resolveDiskFromDeviceID(deviceID string) (string, error) {
	if deviceID == "" {
		return "", fmt.Errorf("empty device id")
	}

	target, err := filepath.EvalSymlinks(filepath.Join("/sys/dev/block", deviceID))
	if err != nil {
		return "", fmt.Errorf("resolve /sys/dev/block/%s: %w", deviceID, err)
	}

	parts := strings.Split(target, string(filepath.Separator))
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "block" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}

	return "", fmt.Errorf("could not map %s to /sys/block disk", deviceID)
}

// blockStatBusyMsIndex is the 0-based field index of "time doing I/Os (ms)" (iostat %util).
func blockStatBusyMsIndex(fieldCount int) (int, error) {
	switch {
	case fieldCount >= 15:
		return 9, nil
	case fieldCount >= 11:
		return 9, nil
	case fieldCount >= 9:
		return 7, nil
	default:
		return 0, fmt.Errorf("parse block stat: only %d fields", fieldCount)
	}
}

func readBlockStatBusyMs(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read block stat: %w", err)
	}

	fields := strings.Fields(string(data))
	idx, err := blockStatBusyMsIndex(len(fields))
	if err != nil {
		return 0, err
	}

	ioTime, err := strconv.ParseUint(fields[idx], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse block stat busy time: %w", err)
	}

	return ioTime, nil
}

// readMountIOBusy returns disk I/O utilization for the mount's backing block device.
// Same basis as iostat %util: 100 * Δbusy_ms / interval_ms, clamped 0–100.
func (c *Collector) readMountIOBusy(mount string, uptime float64) (float64, error) {
	statPath, err := mountDiskIOStatPath(mount)
	if err != nil {
		return 0, err
	}

	ioTimeMs, err := readBlockStatBusyMs(statPath)
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.prevMountIO == nil {
		c.prevMountIO = make(map[string]mountIOSample)
	}

	prev, ok := c.prevMountIO[mount]
	c.prevMountIO[mount] = mountIOSample{ioTimeMs: ioTimeMs, uptime: uptime}
	if !ok {
		return 0, nil
	}

	uptimeDelta := uptime - prev.uptime
	if uptimeDelta <= 0 {
		return 0, nil
	}

	ioDelta := counterDelta(ioTimeMs, prev.ioTimeMs)
	percent := float64(ioDelta) / (uptimeDelta * 10)

	return clampPercent(percent), nil
}
