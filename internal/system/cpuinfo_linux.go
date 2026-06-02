//go:build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func readCPUModelName() (string, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", fmt.Errorf("read cpuinfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "model name") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		if name != "" {
			return name, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan cpuinfo: %w", err)
	}

	return "", fmt.Errorf("parse cpuinfo: missing model name")
}
