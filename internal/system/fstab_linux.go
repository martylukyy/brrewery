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

// mountedDeviceByMount maps each active mount point to its backing device ID
// (major:minor) from /proc/self/mountinfo. When a point is mounted more than
// once the last (topmost, statfs-visible) entry wins.
func mountedDeviceByMount() (map[string]string, error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, fmt.Errorf("read mountinfo: %w", err)
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		mountPoint := unescapeFstabField(fields[4])
		result[mountPoint] = fields[2]
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan mountinfo: %w", err)
	}
	return result, nil
}

// activeMonitoredMounts filters candidate mount points down to those that are
// actually mounted, keeping a single mount per backing device.
//
// /etc/fstab lists intended mounts, not active ones: statfs() on an unmounted
// path silently reports the parent filesystem (usually root), so an unmounted
// data drive would duplicate root's usage. Several mounts can also share one
// device (btrfs subvolumes, bind mounts), where statfs reports the whole
// filesystem for each — counting that device once avoids inflating the totals.
func activeMonitoredMounts(candidates []string, deviceByMount map[string]string) []string {
	seenDevice := make(map[string]struct{}, len(candidates))
	active := make([]string, 0, len(candidates))
	for _, mount := range candidates {
		device, mounted := deviceByMount[mount]
		if !mounted {
			continue
		}
		if device != "" {
			if _, dup := seenDevice[device]; dup {
				continue
			}
			seenDevice[device] = struct{}{}
		}
		active = append(active, mount)
	}
	return active
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
