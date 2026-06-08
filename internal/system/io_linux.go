//go:build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const diskSectorSize = 512

func readNetworkCounters() (NetworkCounters, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetworkCounters{}, fmt.Errorf("read net dev: %w", err)
	}
	defer file.Close()

	var total NetworkCounters
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}

		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if skipNetworkInterface(iface) {
			continue
		}

		fields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(fields) < 9 {
			continue
		}

		rxBytes, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		txBytes, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}

		total.RxBytes += rxBytes
		total.TxBytes += txBytes
	}
	if err := scanner.Err(); err != nil {
		return NetworkCounters{}, fmt.Errorf("scan net dev: %w", err)
	}

	return total, nil
}

func skipNetworkInterface(name string) bool {
	switch name {
	case "lo":
		return true
	default:
		return strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "virbr")
	}
}
