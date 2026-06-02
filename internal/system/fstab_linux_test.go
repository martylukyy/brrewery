//go:build linux

package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitoredMountsFromFstab(t *testing.T) {
	t.Parallel()

	const sample = `
# comment
UUID=root-uuid / ext4 errors=remount-ro 0 1
UUID=efi-uuid /boot/efi vfat umask=0077 0 1
UUID=swap-uuid none swap sw 0 0
UUID=data-uuid /mnt/storage ext4 defaults 0 2
/dev/sr0 /media/cdrom0 udf,iso9660 user,noauto 0 0
`

	mounts, err := monitoredMountsFromFstab(sample)
	require.NoError(t, err)
	assert.Equal(t, []string{"/", "/mnt/storage"}, mounts)
}

func TestMonitoredFstabMounts_readsHostFstab(t *testing.T) {
	mounts, err := monitoredFstabMounts()
	require.NoError(t, err)
	require.Contains(t, mounts, "/")
	assert.NotContains(t, mounts, "/boot/efi")

	info, err := NewCollector().Collect()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(info.Disks), 2)

	mountSet := make(map[string]struct{}, len(info.Disks))
	for _, disk := range info.Disks {
		mountSet[disk.Mount] = struct{}{}
	}

	for _, mount := range mounts {
		_, err := readDiskUsage(mount)
		if err != nil {
			continue
		}
		assert.Contains(t, mountSet, mount)
	}
}

func TestShouldMonitorFstabEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry fstabEntry
		want  bool
	}{
		{
			name:  "root ext4",
			entry: fstabEntry{mount: "/", fstype: "ext4"},
			want:  true,
		},
		{
			name:  "swap by type",
			entry: fstabEntry{mount: "none", fstype: "swap", options: "sw"},
			want:  false,
		},
		{
			name:  "efi partition",
			entry: fstabEntry{mount: "/boot/efi", fstype: "vfat"},
			want:  false,
		},
		{
			name:  "efi path prefix",
			entry: fstabEntry{mount: "/boot/efi/custom", fstype: "vfat"},
			want:  false,
		},
		{
			name:  "data mount",
			entry: fstabEntry{mount: "/mnt/storage", fstype: "ext4"},
			want:  true,
		},
		{
			name:  "odd mixed fs skipped",
			entry: fstabEntry{mount: "/media/cdrom0", fstype: "udf,iso9660", options: "user,noauto"},
			want:  false,
		},
		{
			name:  "noauto mount skipped",
			entry: fstabEntry{mount: "/mnt/archive", fstype: "ext4", options: "defaults,noauto"},
			want:  false,
		},
		{
			name:  "relative mount skipped",
			entry: fstabEntry{mount: "mnt", fstype: "ext4"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldMonitorFstabEntry(tt.entry))
		})
	}
}
