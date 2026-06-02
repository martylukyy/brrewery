//go:build linux

package system

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

type fstabEntry struct {
	device  string
	mount   string
	fstype  string
	options string
}

// monitoredFstabMounts returns unique mount points from /etc/fstab that should be
// monitored (swap and EFI partitions are excluded).
func monitoredFstabMounts() ([]string, error) {
	data, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return nil, fmt.Errorf("read fstab: %w", err)
	}
	return monitoredMountsFromFstab(string(data))
}

func monitoredMountsFromFstab(content string) ([]string, error) {
	seen := make(map[string]struct{})
	var mounts []string

	for _, entry := range parseFstabEntries(content) {
		if !shouldMonitorFstabEntry(entry) {
			continue
		}
		if _, ok := seen[entry.mount]; ok {
			continue
		}
		seen[entry.mount] = struct{}{}
		mounts = append(mounts, entry.mount)
	}

	slices.Sort(mounts)
	return mounts, nil
}

func parseFstabEntries(content string) []fstabEntry {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var entries []fstabEntry

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		entries = append(entries, fstabEntry{
			device:  fields[0],
			mount:   unescapeFstabField(fields[1]),
			fstype:  fields[2],
			options: fields[3],
		})
	}

	return entries
}

func unescapeFstabField(field string) string {
	return strings.ReplaceAll(field, "\\040", " ")
}

func shouldMonitorFstabEntry(entry fstabEntry) bool {
	if entry.mount == "" || !strings.HasPrefix(entry.mount, "/") {
		return false
	}
	if strings.Contains(entry.fstype, ",") {
		return false
	}
	if strings.EqualFold(entry.fstype, "swap") {
		return false
	}
	if strings.Contains(strings.ToLower(entry.options), "swap") {
		return false
	}
	if hasAnyMountOption(entry.options, "noauto", "user", "users") {
		return false
	}
	if !isSupportedMonitoredFSType(entry.fstype) {
		return false
	}
	if isEFIFstabEntry(entry) {
		return false
	}
	return true
}

func hasAnyMountOption(options string, want ...string) bool {
	for _, opt := range strings.Split(strings.ToLower(options), ",") {
		trimmed := strings.TrimSpace(opt)
		for _, w := range want {
			if trimmed == w {
				return true
			}
		}
	}
	return false
}

func isSupportedMonitoredFSType(fsType string) bool {
	switch strings.ToLower(strings.TrimSpace(fsType)) {
	case "ext2", "ext3", "ext4", "xfs", "btrfs", "zfs", "f2fs", "jfs", "reiserfs", "nilfs2":
		return true
	default:
		return false
	}
}

func isEFIFstabEntry(entry fstabEntry) bool {
	mount := strings.ToLower(entry.mount)
	if mount == "/boot/efi" || mount == "/efi" || strings.HasPrefix(mount, "/boot/efi/") {
		return true
	}

	switch strings.ToLower(entry.fstype) {
	case "vfat", "msdos", "fat", "fat32", "efivarfs":
		return strings.Contains(mount, "efi")
	default:
		return false
	}
}
