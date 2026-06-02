//go:build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type cpuCounters struct {
	total uint64
	idle  uint64
}

func (c *Collector) readCPUPercent() (float64, error) {
	current, err := readProcCPU()
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.prevCPU == nil {
		prev := current
		c.prevCPU = &prev
		return 0, nil
	}

	totalDelta := counterDelta(current.total, c.prevCPU.total)
	if totalDelta == 0 {
		*c.prevCPU = current
		return 0, nil
	}

	idleDelta := counterDelta(current.idle, c.prevCPU.idle)
	*c.prevCPU = current

	busy := float64(totalDelta-idleDelta) / float64(totalDelta) * 100

	return clampPercent(busy), nil
}

func counterDelta(current, previous uint64) uint64 {
	if current < previous {
		return 0
	}
	return current - previous
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func readProcCPU() (cpuCounters, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpuCounters{}, fmt.Errorf("read proc stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return cpuCounters{}, fmt.Errorf("parse cpu line: too few fields")
		}

		var values []uint64
		for _, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return cpuCounters{}, fmt.Errorf("parse cpu field: %w", err)
			}
			values = append(values, value)
		}

		var total uint64
		for _, value := range values {
			total += value
		}

		idle := values[3]
		if len(values) > 4 {
			idle += values[4]
		}

		return cpuCounters{total: total, idle: idle}, nil
	}
	if err := scanner.Err(); err != nil {
		return cpuCounters{}, fmt.Errorf("scan proc stat: %w", err)
	}

	return cpuCounters{}, fmt.Errorf("parse proc stat: missing cpu line")
}
